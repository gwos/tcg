package clients

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/log"
	"math/big"
	"net"
	"net/http"
	"net/url"
)

const (
	tableDevices    = "devices"
	tableMonitoring = "monitoring"
	tableInterfaces = "interfaces"

	colName     = "name"
	colDevice   = "device"
	colDevIp    = "devip"
	colMonIp    = "monip"
	colReadComm = "readcomm"
	colIfName   = "ifname"
	colIfIndex  = "ifidx"
	colLastOK   = "lastok"
)

type NediClient struct {
	Server string
}

type Device struct {
	Name      string
	Ip        string
	Community string
	LastOK    float64
}

type Monitoring struct {
	Name   string
	Ip     string
	Device string
	LastOK float64
}

type Interface struct {
	Name   string
	Device string
	Index  int
}

func (client *NediClient) Init(server string) error {
	if server == "" {
		return errors.New("missing NeDi server")
	}
	client.Server = server
	return nil
}

func (client *NediClient) GetDevices() ([]Device, error) {
	path, err := client.getConnectionString(tableDevices, "", "")
	if err != nil {
		log.Error("|nediClient.go| : [getDevices]: Failed to get NeDi connection string: ", err)
		return nil, errors.New("failed to get NeDi connection string")
	}

	log.Debug("[NediClient]: Performing NeDi Get Devices request: ", path)
	r, err := executeGet(*path)
	if err != nil {
		log.Error("|nediClient.go| : [getDevices]: Failed to execute NeDi request : ", err)
		return nil, errors.New("failed to execute NeDi request")
	}
	log.Debug("[NediClient]: NeDi Get Devices response: ", r)

	var resp []interface{}
	err = json.Unmarshal(r, &resp)
	if err != nil {
		log.Error("|nediClient.go| : [getDevices]: Failed to parse NeDi response: ", err)
		log.Error("result: ", string(r[:]))
		return nil, errors.New("failed to parse NeDi response")
	}

	return parseDevices(client, resp)
}

func (client *NediClient) GetDeviceInterfaces(device string) ([]Interface, error) {
	if device == "" {
		return nil, errors.New("missing device")
	}

	query := colDevice + " = " + device
	path, err := client.getConnectionString(tableInterfaces, query, "")
	if err != nil {
		log.Error("|nediClient.go| : [GetDeviceInterfaces]: Failed to get NeDi connection string: ", err)
		return nil, errors.New("failed to get NeDi connection string")
	}

	log.Debug("[NediClient]: Performing NeDi Get Interfaces request: ", path)
	r, err := executeGet(*path)
	if err != nil {
		log.Error("|nediClient.go| : [GetDeviceInterfaces]: Failed to execute NeDi request: ", err)
		log.Error("result: ", string(r[:]))
		return nil, errors.New("failed to execute NeDi request")
	}
	log.Debug("[NediClient]: NeDi Get Interfaces response: ", r)

	var resp []interface{}
	err = json.Unmarshal(r, &resp)
	if err != nil {
		log.Error("|nediClient.go| : [getDeviceInterfaces]: Failed to parse NeDi response: ", err)
		log.Error("result: ", string(r[:]))
		return nil, errors.New("failed to parse NeDi response")
	}

	return parseInterfaces(resp), nil
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
	log.Debug("connection string: ", connStr)
	return &connStr, nil
}

func parseDevices(client *NediClient, response []interface{}) ([]Device, error) {
	var devices []Device
	monitoredDevices, err := getMonitoredDevices(client)
	if err != nil {
		return devices, err
	}
	for _, d := range parseResponse(response) {
		var device Device

		name := d[colDevice]
		ip := d[colDevIp]
		community := d[colReadComm]

		switch nameVal := name.(type) {
		case string:
			device.Name = nameVal
		default:
			log.Warn("|nediClient.go| : [parseDevices]: Skipping device: ",
				colDevice, " '", nameVal, "' of unsupported type")
			continue
		}
		// filter out un-monitored devices
		if mon, ok := monitoredDevices[device.Name]; ok {
			device.LastOK = mon.LastOK
		} else {
			continue
		}

		ipVal, err := getInt(ip)
		if err != nil {
			log.Warn("|nediClient.go| : [parseDevices]: Skipping device '", device.Name,
				"': ", colDevIp, " '", ipVal, "' of unsupported type")
			continue
		}
		device.Ip = int2ip(ipVal)

		switch commVal := community.(type) {
		case string:
			device.Community = commVal
		default:
			log.Warn("|nediClient.go| : [parseDevices]: Skipping device '", device.Name,
				"': ", colReadComm, " '", commVal, "' of unsupported type")
			continue
		}

		devices = append(devices, device)
	}
	return devices, nil
}

func parseInterfaces(response []interface{}) []Interface {
	var interfaces []Interface
	for _, i := range parseResponse(response) {
		var iFace Interface

		name := i[colIfName]
		device := i[colDevice]
		index := i[colIfIndex]

		switch nameVal := name.(type) {
		case string:
			iFace.Name = nameVal
		default:
			log.Warn("|nediClient.go| : [parseInterfaces]: Skipping interface: ",
				colIfName, " '", nameVal, "' of unsupported type")
			continue
		}

		switch devVal := device.(type) {
		case string:
			iFace.Device = devVal
		default:
			log.Warn("|nediClient.go| : [parseInterfaces]: Skipping interface '", iFace.Name,
				"': ", colDevice, " '", devVal, "' of unsupported type")
			continue
		}

		idxVal, err := getInt(index)
		if err != nil {
			log.Warn("|nediClient.go| : [parseInterfaces]: Skipping interface '", iFace.Name,
				"': ", colIfIndex, " '", idxVal, "' of unsupported type")
			continue
		}
		iFace.Index = idxVal

		interfaces = append(interfaces, iFace)
	}
	return interfaces
}

func parseResponse(response []interface{}) []map[string]interface{} {
	log.Debug("[NediClient]: Parsing NeDi response: ", response)

	var res []map[string]interface{}
	for i, r := range response {
		log.Debug("[NediClient]: Parsing NeDi response object: ", r)
		if i == 0 {
			log.Debug("[NediClient]: Skipping system information")
			continue
		}
		switch o := r.(type) {
		case map[string]interface{}:
			res = append(res, o)
		default:
			log.Warn("|nediClient.go| : [parseResponse]: Skipping response object: ", o, ". Unsupported type")
			continue
		}
	}
	log.Debug("[NediClient]: Parsing NeDi response completed: ", res)
	return res
}

func executeGet(url string) ([]byte, error) {
	s, r, err := clients.SendRequest(http.MethodGet, url, nil, nil, nil)
	if err != nil || s != 200 || r == nil {
		if err != nil {
			log.Error(err)
		}
		if s != 200 {
			log.Error("Response status: ", s)
		}
		if r == nil {
			log.Error("Response is nil")
		} else {
			log.Error("Response body: ", r)
		}
		return nil, errors.New("failed to get from NeDi")
	}
	return r, nil
}

func getInt(val interface{}) (int, error) {
	switch numVal := val.(type) {
	case int:
		return numVal, nil
	case int8:
		return int(numVal), nil
	case int16:
		return int(numVal), nil
	case int32:
		return int(numVal), nil
	case int64:
		return int(numVal), nil
	case float32:
		return int(numVal), nil
	case float64:
		return int(numVal), nil
	case big.Int:
		return int(numVal.Int64()), nil
	default:
		return 0, errors.New("unsupported type: ")
	}
}

func int2ip(val int) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, uint32(val))
	return ip.String()
}

func getMonitoredDevices(client *NediClient) (map[string]Monitoring, error) {
	// TODO: remove this, use log.SetFlags(log.LstdFlags | log.Lshortfile)
	codePoint := "|nediClient.go| : [getMonitoring]: "
	path, err := client.getConnectionString(tableMonitoring, "", "")
	if err != nil {
		msg := "failed to get NeDi monitoring connection string"
		log.Error(codePoint, msg, err)
		return nil, errors.New(msg)
	}

	log.Debug("[NediClient]: Performing NeDi Get Monitoring request: ", path)
	response, err := executeGet(*path)
	if err != nil {
		msg := "Failed to execute NeDi monitoring request: "
		log.Error(codePoint, msg, err)
		return nil, errors.New(msg)
	}
	log.Debug("[NediClient]: NeDi Get Monitoring response: ", response)

	var resp []interface{}
	err = json.Unmarshal(response, &resp)
	if err != nil {
		msg := "Failed to Parse NeDi monitoring response : "
		log.Error(codePoint, msg, err)
		log.Error("result: ", string(response[:]))
		return nil, errors.New(msg)
	}

	monitors := make(map[string]Monitoring)
	for _, fields := range parseResponse(resp) {
		var monitor Monitoring
		monitor.Name = fields[colName].(string)
		monitor.Device = fields[colDevice].(string)
		monitor.LastOK = fields[colLastOK].(float64)

		ip := fields[colMonIp]
		ipVal, err := getInt(ip)
		if err != nil {
			log.Warn("|nediClient.go| : [parseDevices]: Skipping monitoring '", monitor.Name,
				"': ", colMonIp, " '", ipVal, "' of unsupported type")
			continue
		}
		monitor.Ip = int2ip(ipVal)
		monitors[monitor.Device] = monitor
	}
	return monitors, nil
}
