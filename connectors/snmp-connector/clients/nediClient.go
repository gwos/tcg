package clients

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"

	"github.com/gwos/tcg/sdk/clients"
	"github.com/rs/zerolog/log"
)

const (
	tableDevices    = "devices"
	tableMonitoring = "monitoring"
	tableInterfaces = "interfaces"

	colName     = "name"
	colDevice   = "device"
	colDevIP    = "devip"
	colMonIP    = "monip"
	colReadComm = "readcomm"
	colIfName   = "ifname"
	colIfIndex  = "ifidx"
	colLastOK   = "lastok"
	colIfStat   = "ifstat"
)

type NediClient struct {
	Server string
}

type Device struct {
	Name      string
	IP        string
	Community string
	LastOK    float64
}

type Monitoring struct {
	Name   string
	IP     string
	Device string
	LastOK float64
}

type Interface struct {
	Name   string
	Device string
	Index  int
	Status int
}

func (client *NediClient) Init(server string) error {
	if server == "" {
		return errors.New("missing NeDi server")
	}
	client.Server = server
	return nil
}

func (client *NediClient) GetDevices() ([]Device, error) {
	monitored, err := client.getMonitoredDevices()
	if err != nil {
		return nil, err
	}
	if len(monitored) == 0 {
		log.Warn().Msg("skipping NeDi Get Devices request due to empty monitored inventory")
		return nil, nil
	}

	path, err := client.getConnectionString(tableDevices, "", "")
	if err != nil {
		log.Err(err).Msg("could not get NeDi connection string")
		return nil, errors.New("failed to get NeDi connection string")
	}

	log.Debug().Msgf("performing NeDi Get Devices request: %s", *path)
	r, err := executeGet(*path)
	if err != nil {
		log.Err(err).
			Str("request", *path).
			Msg("could not execute NeDi request")
		return nil, errors.New("failed to execute NeDi request")
	}
	log.Debug().Bytes("response", r).Msgf("NeDi Get Devices response")

	return parseDevices(r, monitored)
}

func (client *NediClient) GetDeviceInterfaces(device string) ([]Interface, error) {
	if device == "" {
		return nil, errors.New("missing device")
	}

	query := colDevice + " = " + device
	path, err := client.getConnectionString(tableInterfaces, query, "")
	if err != nil {
		log.Err(err).Msg("could not get NeDi connection string")
		return nil, errors.New("failed to get NeDi connection string")
	}

	log.Debug().Msgf("performing NeDi Get Interfaces request: %s", *path)
	r, err := executeGet(*path)
	if err != nil {
		log.Err(err).
			Str("request", *path).
			Msg("could not execute NeDi request")
		return nil, errors.New("failed to execute NeDi request")
	}
	log.Debug().Bytes("response", r).Msgf("NeDi Get Interfaces response")

	return parseInterfaces(r), nil
}

func (client *NediClient) getConnectionString(table string, query string, order string) (*string, error) {
	if client.Server == "" {
		return nil, errors.New("nedi client is not configured")
	}
	if table == "" {
		return nil, errors.New("missing table")
	}
	var q, o string
	if query != "" {
		q = "&q=" + url.QueryEscape(query)
	}
	if order != "" {
		o = "&o=" + url.QueryEscape(order)
	}
	connStr := "http://" + client.Server + "/nedi/query.php?c=1&t=" + table + q + o
	log.Debug().Msgf("connection string: %s", connStr)
	return &connStr, nil
}

func parseDevices(bytes []byte, monitored map[string]Monitoring) ([]Device, error) {
	var devices = make([]Device, 0, len(monitored))
	for _, d := range parseResponse(bytes) {
		var device Device

		name := d[colDevice]
		ip := d[colDevIP]
		community := d[colReadComm]

		switch nameVal := name.(type) {
		case string:
			device.Name = nameVal
		default:
			log.Warn().Msgf("skipping device '%s:%s' of unsupported type",
				colDevice, nameVal)
			continue
		}
		// filter out un-monitored devices
		if mon, ok := monitored[device.Name]; ok {
			device.LastOK = mon.LastOK
		} else {
			continue
		}

		ipVal, err := getInt(ip)
		if err != nil {
			log.Warn().Msgf("skipping device '%s:%s:%d' of unsupported type",
				device.Name, colDevIP, ipVal)
			continue
		}
		device.IP = int2ip(ipVal)

		switch commVal := community.(type) {
		case string:
			device.Community = commVal
		default:
			log.Warn().Msgf("skipping device '%s:%s:%s' of unsupported type",
				device.Name, colReadComm, commVal)
			continue
		}

		devices = append(devices, device)
	}
	return devices, nil
}

func parseInterfaces(response []byte) []Interface {
	var interfaces = make([]Interface, 0)
	for _, i := range parseResponse(response) {
		var iFace Interface

		name := i[colIfName]
		device := i[colDevice]
		index := i[colIfIndex]
		status := i[colIfStat]

		switch nameVal := name.(type) {
		case string:
			iFace.Name = nameVal
		default:
			log.Warn().Msgf("skipping interface '%s:%s' of unsupported type",
				colIfName, nameVal)
			continue
		}

		switch devVal := device.(type) {
		case string:
			iFace.Device = devVal
		default:
			log.Warn().Msgf("skipping interface '%s:%s:%s' of unsupported type",
				iFace.Name, colDevice, devVal)
			continue
		}

		iFace.Status = -1
		if statVal, err := getInt(status); err == nil {
			iFace.Status = statVal
		} else {
			log.Warn().Msgf("skipping interface '%s:%s:%d' of unsupported type",
				iFace.Name, colIfStat, status)
		}

		idxVal, err := getInt(index)
		if err != nil {
			log.Warn().Msgf("skipping interface '%s:%s:%d' of unsupported type",
				iFace.Name, colIfIndex, idxVal)
			continue
		}
		iFace.Index = idxVal

		interfaces = append(interfaces, iFace)
	}
	return interfaces
}

func parseResponse(bytes []byte) []map[string]interface{} {
	log.Debug().
		Bytes("response", bytes).
		Msg("parsing NeDi response")

	var response []interface{}
	if err := json.Unmarshal(bytes, &response); err != nil {
		log.Err(err).Bytes("response", bytes).Msg("could not parse NeDi response")
		return nil
	}

	var res []map[string]interface{}
	for i, r := range response {
		log.Debug().
			Interface("obj", r).
			Msg("parsing NeDi response object")
		if i == 0 {
			log.Debug().Msg("skipping system information")
			continue
		}
		switch o := r.(type) {
		case map[string]interface{}:
			res = append(res, o)
		default:
			log.Warn().
				Interface("obj", o).
				Msg("skipping response object of unsupported type")
			continue
		}
	}
	log.Debug().
		Interface("res", res).
		Msg("parsing NeDi response completed")
	return res
}

func executeGet(url string) ([]byte, error) {
	s, r, err := clients.SendRequest(http.MethodGet, url, nil, nil, nil)
	if err != nil || s != 200 || r == nil {
		log.Error().
			Err(err).
			Int("status", s).
			Bytes("response", r).
			Msg("could not send request")
		return nil, errors.New("failed to get from NeDi")
	}
	return r, nil
}

func getInt(v interface{}) (int, error) {
	switch v := v.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case big.Int:
		return int(v.Int64()), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

func int2ip(val int) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, uint32(val))
	return ip.String()
}

func (client *NediClient) getMonitoredDevices() (map[string]Monitoring, error) {
	path, err := client.getConnectionString(tableMonitoring, "", "")
	if err != nil {
		msg := "failed to get NeDi monitoring connection string"
		log.Err(err).Msg(msg)
		return nil, errors.New(msg)
	}

	log.Debug().Msgf("performing NeDi Get Monitoring request: %s", *path)
	response, err := executeGet(*path)
	if err != nil {
		msg := "could not execute NeDi monitoring request"
		log.Err(err).Msg(msg)
		return nil, errors.New(msg)
	}
	log.Debug().Bytes("response", response).Msgf("NeDi Get Monitoring response")

	monitors := make(map[string]Monitoring)
	for _, fields := range parseResponse(response) {
		var monitor Monitoring
		monitor.Name = fields[colName].(string)
		monitor.Device = fields[colDevice].(string)
		monitor.LastOK = fields[colLastOK].(float64)

		ip := fields[colMonIP]
		ipVal, err := getInt(ip)
		if err != nil {
			log.Warn().Msgf("skipping monitoring '%s:%s:%d' of unsupported type",
				monitor.Name, colMonIP, ipVal)
			continue
		}
		monitor.IP = int2ip(ipVal)
		monitors[monitor.Device] = monitor
	}
	return monitors, nil
}
