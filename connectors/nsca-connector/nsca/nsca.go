package nsca

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tubemogul/nscatools"
)

func AdaptHandler(h func([]byte) error) DataHandler {
	// It's unclear for now how to process multi-line metrics in the right way.
	// The nscatools.DataPacket provides host, service, and state info for the 1st line only.
	// And PluginOutput contains the rest.
	// The `fullPacket` payload that processed in the HandleClient() by nscatools.DataPacket.Read()
	// contains a lot of noise and looks like protocol defined structure.
	// So, here we try to reconstruct plain metrics payload and process it with own parser.
	return func(p *DataPacketExt) error {
		s := strings.Replace(p.PluginOutput, `\n`, "\n", -1)
		buf := make([]byte, 0, 4+len(p.HostName)+len(p.Service)+len(s))
		buf = append(buf, p.HostName...)
		buf = append(buf, ';')
		buf = append(buf, p.Service...)
		buf = append(buf, ';')
		buf = strconv.AppendInt(buf, int64(p.State), 10)
		buf = append(buf, ';')
		buf = append(buf, s...)
		log.Debug().
			Int16("version", p.Version).
			Uint32("crc", p.Crc).
			Uint32("timestamp", p.Timestamp).
			Int16("state", p.State).
			Str("hostname", p.HostName).
			Str("service", p.Service).
			Str("pluginOutput", p.PluginOutput).
			Bytes("buf", buf).
			Func(func(e *zerolog.Event) {
				println("# NSCA plain packet #")
				println(string(buf))
			}).
			Msg("processing DataPacket")

		err := h(buf)
		if err != nil {
			log.Warn().Err(err).
				Bytes("payload", buf).
				Msg("could not process incoming data")
		}
		return nil
	}
}

func Start(ctx context.Context, handler DataHandler) {
	nscaHost := "0.0.0.0"
	nscaPort := uint16(5667)
	nscaEncrypt := nscatools.EncryptNone
	nscaPassword := ""
	nscaEnvHost := os.Getenv("NSCA_HOST")
	nscaEnvPort := os.Getenv("NSCA_PORT")
	nscaEnvEncrypt := os.Getenv("NSCA_ENCRYPT")
	nscaEnvPassword := os.Getenv("NSCA_PASSWORD")
	if len(nscaEnvHost) > 0 {
		nscaHost = nscaEnvHost
	}
	if len(nscaEnvPort) > 0 {
		if i, err := strconv.Atoi(nscaEnvPort); err == nil {
			nscaPort = uint16(i)
		}
	}
	if len(nscaEnvEncrypt) > 0 {
		if i, err := strconv.Atoi(nscaEnvEncrypt); err == nil {
			nscaEncrypt = int(i)
		}
	}
	if len(nscaEnvPassword) > 0 {
		nscaPassword = nscaEnvPassword
	}

	go StartServerWithContext(ctx,
		NewConfigExt(nscaHost, nscaPort, nscaEncrypt, nscaPassword, handler))
}

func StartServerWithContext(ctx context.Context, conf *ConfigExt) error {
	service := fmt.Sprint(conf.Host, ":", conf.Port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	if err != nil {
		return fmt.Errorf("unable to resolve address: %w", err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("unable to open a TCP listener: %w", err)
	}
	defer listener.Close()

	log.Info().Msgf("start tcp listener %s", service)
	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("stop tcp listener %s", service)
			return nil

		default:
			if err := listener.SetDeadline(time.Now().Add(time.Second)); err != nil {
				return err
			}
			conn, err := listener.Accept()
			if err != nil {
				if os.IsTimeout(err) {
					continue
				}
				log.Err(err).Msg("could not accept")
			}
			defer conn.Close()

			// run as a goroutine
			go HandleClientExt(conf, conn)
		}
	}
}

func HandleClientExt(conf *ConfigExt, conn net.Conn) error {
	// close connection on exit
	defer conn.Close()

	// sends the initialization packet
	ipacket, err := nscatools.NewInitPacket()
	if err != nil {
		log.Err(err).Msg("unable to create the init packet")
		return err
	}
	if err = ipacket.Write(conn); err != nil {
		log.Err(err).Msg("unable to send the init packet")
		return err
	}

	// Retrieves the data from the client
	data := nscatools.NewDataPacket(conf.EncryptionMethod, []byte(conf.Password), ipacket)
	dp := &DataPacketExt{*data}
	if err = dp.Read(conn); err != nil {
		log.Err(err).Msg("unable to read the data packet")
		return err
	}
	if err = conf.PacketHandler(dp); err != nil {
		log.Err(err).Msg("unable to process the data packet in the custom handler")
	}
	return err
}

type DataHandler func(*DataPacketExt) error

type ConfigExt struct {
	nscatools.Config
	PacketHandler DataHandler
}

func NewConfigExt(host string, port uint16, encryption int, password string, handler DataHandler) *ConfigExt {
	c := nscatools.NewConfig(host, port, encryption, password, nil)
	cfg := ConfigExt{*c, handler}
	return &cfg
}

type DataPacketExt struct {
	nscatools.DataPacket
}

func (p *DataPacketExt) Read(conn io.Reader) error {
	// We need to read the full packet 1st to check the crc and decrypt it too
	fullPacket, err := ioutil.ReadAll(conn)
	if err != nil {
		return err
	}

	// if len(fullPacket) != ShortPacketLength && len(fullPacket) != LongPacketLength {
	// 	return fmt.Errorf("Dropping packet with invalid size: %d", len(fullPacket))
	// }

	if err := p.Decrypt(fullPacket); err != nil {
		return err
	}

	log.Debug().Func(func(e *zerolog.Event) {
		println("# NSCA fullPacket #")
		println(string(fullPacket))
	}).Send()

	p.Crc = binary.BigEndian.Uint32(fullPacket[4:8])
	if crc32 := p.CalculateCrc(fullPacket); p.Crc != crc32 {
		return fmt.Errorf("invalid CRC32: possibly due to mismatch password or crypto algorithm")
	}

	p.Timestamp = binary.BigEndian.Uint32(fullPacket[8:12])
	// MaxPacketAge <= 0 means that we don't check it
	// if MaxPacketAge > 0 {
	// 	if p.Timestamp > (p.Ipkt.Timestamp+MaxPacketAge) || p.Timestamp < (p.Ipkt.Timestamp-MaxPacketAge) {
	// 		return fmt.Errorf("Dropping packet with stale timestamp - Max age difference is %d seconds", MaxPacketAge)
	// 	}
	// }

	sep := []byte("\x00") // sep is used to extract only the useful string
	p.Version = int16(binary.BigEndian.Uint16(fullPacket[0:2]))
	p.State = int16(binary.BigEndian.Uint16(fullPacket[12:14]))
	p.HostName = string(bytes.Split(fullPacket[14:78], sep)[0])
	p.Service = string(bytes.Split(fullPacket[78:206], sep)[0])
	p.PluginOutput = string(bytes.Split(fullPacket[206:], sep)[0])

	return nil
}
