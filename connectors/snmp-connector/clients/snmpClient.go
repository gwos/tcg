package clients

import (
	"errors"
	"fmt"
	snmp "github.com/gosnmp/gosnmp"
	"github.com/gwos/tcg/connectors/snmp-connector/utils"
	"github.com/gwos/tcg/log"
	"strings"
	"time"
)

type SnmpClient struct {
}

type SnmpUnitType string

const (
	Number SnmpUnitType = "number"
	Bit    SnmpUnitType = "bits"
)

const (
	// supported auth protocols
	md5 = "md5"
	sha = "sha"
	// supported privacy protocols
	aes = "aes"
	des = "des"
)

var AvailableMetrics = map[string]*SnmpMetric{
	"ifSpeed": {Mib: "ifSpeed", Oid: "1.3.6.1.2.1.2.2.1.5", Name: "Interface Speed", UnitType: Bit,
		Description: "An estimate of the interface's current bandwidth in bits per second."},
	"ifInOctets": {Mib: "ifInOctets", Oid: "1.3.6.1.2.1.2.2.1.10", Name: "Inbound Octets", UnitType: Number,
		Description: "The total number of octets received on the interface, including framing characters."},
	"ifOutOctets": {Mib: "ifOutOctets", Oid: "1.3.6.1.2.1.2.2.1.16", Name: "Outbound Octets", UnitType: Number,
		Description: "The total number of octets transmitted out of the interface, including framing characters."},
	"ifInErrors": {Mib: "ifInErrors", Oid: "1.3.6.1.2.1.2.2.1.14", Name: "Inbound Errors", UnitType: Number,
		Description: "For packet-oriented interfaces, the number of inbound packets that contained errors" +
			" preventing them from being deliverable to a higher-layer protocol. For character- oriented or fixed-length interfaces," +
			" the number of inbound transmission units that contained errors preventing them from being deliverable to a higher-layer protocol."},
	"ifOutErrors": {Mib: "ifOutErrors", Oid: "1.3.6.1.2.1.2.2.1.20", Name: "Outbound Errors", UnitType: Number,
		Description: "For packet-oriented interfaces, the number of outbound packets that could not be transmitted because of errors." +
			" For character-oriented or fixed-length interfaces, the number of outbound transmission units that could not be transmitted" +
			" because of errors."},
	"ifInDiscards": {Mib: "ifInDiscards", Oid: "1.3.6.1.2.1.2.2.1.13", Name: "Inbound Discards", UnitType: Number,
		Description: "The number of inbound packets which were chosen to be discarded" +
			" even though no errors had been detected to prevent their being deliverable to a higher-layer protocol."},
	"ifOutDiscards": {Mib: "ifOutDiscards", Oid: "1.3.6.1.2.1.2.2.1.19", Name: "Outbound Discards", UnitType: Number,
		Description: "The number of outbound packets which were chosen to be discarded" +
			" even though no errors had been detected to prevent their being transmitted."},
}

type SnmpMetric struct {
	Mib         string
	Oid         string
	Name        string
	UnitType    SnmpUnitType
	Description string
}

type SnmpValue struct {
	Name  string
	Value int
}

type SnmpMetricData struct {
	SnmpMetric SnmpMetric
	Values     []SnmpValue
}

// retrieve all mibs for one target IP
func (client *SnmpClient) GetSnmpData(mibs []string, target string, secData *utils.SecurityData) ([]SnmpMetricData, error) {
	if len(mibs) == 0 {
		return nil, errors.New("no metrics (mibs) provided")
	}

	log.Info("------ starting SNMP metric gathering for target ", target)
	goSnmp, err := setup(target, secData)
	if err != nil {
		log.Error("|snmpClient.go| : [GetSnmpData]: SNMP setup failed: ", err)
		return nil, errors.New("SNMP setup failed")
	}

	err = goSnmp.Connect()
	if err != nil {
		log.Error("|snmpClient.go| : [GetSnmpData]: SNMP connect failed: ", err)
		return nil, errors.New("SNMP connect failed")
	}
	defer goSnmp.Conn.Close()

	var data []SnmpMetricData
	for _, mib := range mibs {
		mibData, e := getSnmpData(mib, goSnmp) // go get the snmp
		if e != nil {
			goSnmp.Conn.Close()
			err = goSnmp.Connect()
			if err != nil {
				log.Error("NEW : |snmpClient.go| : [GetSnmpData]: SNMP connect failed: ", err)
			}
			log.Error("C|snmpClient.go| : [GetSnmpData]: Failed to get data for target ", target, " + mib ", mib, ": ", e)
			continue
		}
		if mibData != nil {
			data = append(data, *mibData)
		}
	}
	log.Info("------ completed for target ", target)
	fmt.Print("Data: ")
	fmt.Println(data)
	return data, nil
}

func setup(target string, secData *utils.SecurityData) (*snmp.GoSNMP, error) {
	if secData.AuthProtocol != "" {
		return setupV3(target, secData)
	} else {
		return setupV2c(target, secData)
	}
}

func setupV2c(target string, community *utils.SecurityData) (*snmp.GoSNMP, error) {
	err := validate(target, community)

	if err != nil {
		log.Error("|snmpClient.go| : [setupV2c] : Failed to setup snmp v3: ", err)
		return nil, errors.New("validation failed")
	}

	return &snmp.GoSNMP{
		Target:         target,
		Port:           161,
		Community:      community.Name,
		Version:        snmp.Version2c,
		Timeout:        time.Duration(2) * time.Second,
		MaxRepetitions: 2,
		MaxOids:        1,
	}, nil
}

func setupV3(target string, community *utils.SecurityData) (*snmp.GoSNMP, error) {
	err := validate(target, community)
	if err != nil {
		log.Error("|snmpClient.go| : [setupV3] : Failed to setup snmp v3: ", err)
		return nil, errors.New("validation failed")
	}

	var msgFlags snmp.SnmpV3MsgFlags
	var authProtocol snmp.SnmpV3AuthProtocol
	var privProtocol snmp.SnmpV3PrivProtocol

	switch strings.ToLower(community.AuthProtocol) {
	case md5:
		authProtocol = snmp.MD5
		break
	case sha:
		authProtocol = snmp.SHA
		break
	default:
		log.Error("|snmpClient.go| : [setupV3] : Failed to setup snmp v3, unknown authentication protocol: ",
			strings.ToLower(community.AuthProtocol))
		return nil, errors.New("unknown authentication protocol")
	}

	if community.PrivacyProtocol != "" {
		switch strings.ToLower(community.PrivacyProtocol) {
		case aes:
			privProtocol = snmp.AES
			// NoPriv, DES implemented, AES planned
			log.Warn("|snmpClient.go| : [setupV3] : AES privacy protocol may be unsupported yet.")
			break
		case des:
			privProtocol = snmp.DES
			break
		default:
			log.Error("|snmpClient.go| : [setupV3] : Failed to setup snmp v3, unknown privacy protocol: ",
				strings.ToLower(community.PrivacyProtocol))
			return nil, errors.New("unknown privacy protocol")
		}
		msgFlags = snmp.AuthPriv
	} else {
		msgFlags = snmp.AuthNoPriv
	}

	goSnmp := &snmp.GoSNMP{
		Target:        target,
		Port:          161,
		Version:       snmp.Version3,
		SecurityModel: snmp.UserSecurityModel,
		MsgFlags:      msgFlags,
		SecurityParameters: &snmp.UsmSecurityParameters{
			UserName:                 community.Name,
			AuthenticationProtocol:   authProtocol,
			AuthenticationPassphrase: community.AuthPassword,
			PrivacyProtocol:          privProtocol,
			PrivacyPassphrase:        community.PrivacyPassword,
		},
	}

	return goSnmp, nil
}

func validate(target string, secData *utils.SecurityData) error {
	valid := true
	if target == "" {
		log.Error("|snmpClient.go| : [validate] : Validation failed: target required")
		valid = false
	}
	if secData == nil {
		log.Error("|snmpClient.go| : [validate] : Validation failed: security data required")
		valid = false
	} else {
		if secData.Name == "" {
			log.Error("|snmpClient.go| : [validate] : Validation failed: name required")
			valid = false
		}
		if secData.AuthProtocol != "" && secData.AuthPassword == "" {
			log.Error("|snmpClient.go| : [validate] : Validation failed: authentication password required")
			valid = false
		}
		if secData.PrivacyProtocol != "" && secData.PrivacyPassword == "" {
			log.Error("|snmpClient.go| : [validate] : Validation failed: privacy password required")
			valid = false
		}
	}
	if valid {
		return nil
	}
	return errors.New("missing required data")
}

func getSnmpData(mib string, goSnmp *snmp.GoSNMP) (*SnmpMetricData, error) {
	if mib == "" {
		return nil, errors.New("missing mib")
	}

	log.Info("-- start getting MIB: ", mib)

	snmpMetric := AvailableMetrics[mib]
	if snmpMetric == nil {
		return nil, errors.New("unsupported metric " + mib)
	}

	if goSnmp == nil {
		return nil, errors.New("snmp client is not configured")
	}
	if goSnmp.Conn == nil {
		log.Error("|snmpClient.go| : [getSnmpData]: SNMP no connection ")
		return nil, errors.New("SNMP connect failed")
	}

	var data SnmpMetricData
	data.SnmpMetric = *snmpMetric

	walkHandler := func(dataUnit snmp.SnmpPDU) error {
		log.Info("-- walk Handler: data unit name: ", dataUnit.Name, " value: ", dataUnit.Value)
		var val SnmpValue
		val.Name = dataUnit.Name
		switch v := dataUnit.Value.(type) {
		case uint:
			val.Value = int(v)
			log.Info("*** parsed value for ", val.Name, ": ", val.Value)
			break
		default:
			log.Warn("|snmpClient.go| : [getSnmpData]: Value '", v, "' of unsupported type for ", dataUnit.Name)
			break
		}
		data.Values = append(data.Values, val)
		return nil
	}

	oid := data.SnmpMetric.Oid
	err := goSnmp.Walk(oid, walkHandler)
	if err != nil {
		return nil, err
	} else {
		log.Info("-- end getting MIB: ", mib)
	}

	return &data, nil
}
