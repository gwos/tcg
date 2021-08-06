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
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tubemogul/nscatools"
)

func Handler(p *DataPacketExt) error {
	log.Debug().
		Int16("version", p.Version).
		Uint32("crc", p.Crc).
		Uint32("timestamp", p.Timestamp).
		Int16("state", p.State).
		Str("hostname", p.HostName).
		Str("service", p.Service).
		Str("pluginOutput", p.PluginOutput).
		Msg("processing DataPacket")

	return nil
}

func Start(ctx context.Context) {
	nscaHost := "localhost"
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

	// stdlog.SetOutput(log.Logger)
	// nscatools.StartServer(
	// 	nscatools.NewConfig(nscaHost, nscaPort, nscaEncrypt, nscaPassword, Handler),
	// 	true)
	StartServerWithContext(ctx,
		NewConfigExt(nscaHost, nscaPort, nscaEncrypt, nscaPassword, Handler))
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

			// stdlog.SetOutput(log.Logger)
			// logErr := stdlog.New(log.Logger, "", 0)
			// run as a goroutine
			// go nscatools.HandleClient(conf, conn, logErr)
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
	dp := &DataPacketExt{*data, []byte{}}
	if err = dp.Read(conn); err != nil {
		log.Err(err).Msg("unable to read the data packet")
		return err
	}
	if err = conf.PacketHandler(dp); err != nil {
		log.Err(err).Msg("unable to process the data packet in the custom handler")
	}
	return err
}

type dataHandler func(*DataPacketExt) error

type ConfigExt struct {
	nscatools.Config
	PacketHandler dataHandler
}

func NewConfigExt(host string, port uint16, encryption int, password string, handler dataHandler) *ConfigExt {
	c := nscatools.NewConfig(host, port, encryption, password, nil)
	cfg := ConfigExt{*c, handler}
	return &cfg
}

type DataPacketExt struct {
	nscatools.DataPacket
	fullPacket []byte
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

	log.Debug().
		Bytes("fullPacket", fullPacket).
		Msg("reading nscatools.DataPacket")
	p.fullPacket = append(p.fullPacket, fullPacket...)

	p.Crc = binary.BigEndian.Uint32(fullPacket[4:8])
	// if crc32 := p.CalculateCrc(fullPacket); p.Crc != crc32 {
	// 	return fmt.Errorf("Dropping packet with invalid CRC32 - possibly due to client using wrong password or crypto algorithm?")
	// }

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
