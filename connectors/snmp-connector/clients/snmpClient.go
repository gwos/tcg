package clients

import (
	"errors"
	"strings"
	"time"

	snmp "github.com/gosnmp/gosnmp"
	"github.com/gwos/tcg/connectors/snmp-connector/utils"
	"github.com/rs/zerolog/log"
)

type SnmpClient struct {
}

type SnmpUnitType string

const (
	// supported auth protocols
	md5 = "md5"
	sha = "sha"
	// supported privacy protocols
	aes = "aes"
	des = "des"
)

const (
	IfSpeed       = "ifSpeed"
	IfInErrors    = "ifInErrors"
	IfOutErrors   = "ifOutErrors"
	IfInDiscards  = "ifInDiscards"
	IfOutDiscards = "ifOutDiscards"

	IfInOctets    = "ifInOctets"
	IfOutOctets   = "ifOutOctets"
	IfHCInOctets  = "ifHCInOctets"
	IfHCOutOctets = "ifHCOutOctets"

	BytesIn     = "bytesIn"
	BytesOut    = "bytesOut"
	BytesInX64  = "bytesInX64"
	BytesOutX64 = "bytesOutX64"

	BytesPerSecondIn  = "bytesPerSecondIn"
	BytesPerSecondOut = "bytesPerSecondOut"
)

var AvailableMetrics = map[string]*SnmpMetric{
	IfSpeed: {Key: "ifSpeed", Mib: "ifSpeed", Oid: "1.3.6.1.2.1.2.2.1.5", Name: "Interface Speed",
		Description: "An estimate of the interface's current bandwidth in bits per second."},
	IfInErrors: {Key: "ifInErrors", Mib: "ifInErrors", Oid: "1.3.6.1.2.1.2.2.1.14", Name: "Inbound Errors",
		Description: "For packet-oriented interfaces, the number of inbound packets that contained errors" +
			" preventing them from being deliverable to a higher-layer protocol. For character- oriented or fixed-length interfaces," +
			" the number of inbound transmission units that contained errors preventing them from being deliverable to a higher-layer protocol."},
	IfOutErrors: {Key: "ifOutErrors", Mib: "ifOutErrors", Oid: "1.3.6.1.2.1.2.2.1.20", Name: "Outbound Errors",
		Description: "For packet-oriented interfaces, the number of outbound packets that could not be transmitted because of errors." +
			" For character-oriented or fixed-length interfaces, the number of outbound transmission units that could not be transmitted" +
			" because of errors."},
	IfInDiscards: {Key: "ifInDiscards", Mib: "ifInDiscards", Oid: "1.3.6.1.2.1.2.2.1.13", Name: "Inbound Discards",
		Description: "The number of inbound packets which were chosen to be discarded" +
			" even though no errors had been detected to prevent their being deliverable to a higher-layer protocol."},
	IfOutDiscards: {Key: "ifOutDiscards", Mib: "ifOutDiscards", Oid: "1.3.6.1.2.1.2.2.1.19", Name: "Outbound Discards",
		Description: "The number of outbound packets which were chosen to be discarded" +
			" even though no errors had been detected to prevent their being transmitted."},

	IfInOctets: {Key: "bytesIn", Mib: "ifInOctets", Oid: "1.3.6.1.2.1.2.2.1.10", Name: "Inbound Bytes",
		Description: "The total number of octets received on the interface, including framing characters."},
	IfOutOctets: {Key: "bytesOut", Mib: "ifOutOctets", Oid: "1.3.6.1.2.1.2.2.1.16", Name: "Outbound Bytes",
		Description: "The total number of octets transmitted out of the interface, including framing characters."},
	IfHCInOctets: {Key: "bytesInX64", Mib: "ifHCInOctets", Oid: "1.3.6.1.2.1.31.1.1.1.6", Name: "Inbound Bytes 64-bit",
		Description: "The total number of octets received on the interface, including framing characters." +
			"This object is a 64-bit version of ifInOctets."},
	IfHCOutOctets: {Key: "bytesOutX64", Mib: "ifHCOutOctets", Oid: "1.3.6.1.2.1.31.1.1.1.10", Name: "Outbound Bytes 64-bit",
		Description: "The total number of octets transmitted out of the interface, including framing characters. " +
			"This object is a 64-bit version of ifOutOctets."},
}

var NonMibMetrics = map[string]*SnmpMetric{
	BytesPerSecondIn: {Key: "bytesPerSecondIn", Mib: "-", Name: "Inbound Bytes Per Second",
		Description: "The number of inbound bytes per second for the interface"},
	BytesPerSecondOut: {Key: "bytesPerSecondOut", Mib: "-", Name: "Outbound Bytes Per Second",
		Description: "The number of outbound bytes per second for the interface"},
}

type SnmpMetric struct {
	Key         string
	Mib         string
	Oid         string
	Name        string
	UnitType    SnmpUnitType
	Description string
}

type SnmpValue struct {
	Name  string
	Value int64
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

	log.Info().Msgf("------ starting SNMP metric gathering for target '%s'", target)
	goSnmp, err := setup(target, secData)
	if err != nil {
		log.Err(err).Msg("SNMP setup failed")
		return nil, errors.New("SNMP setup failed")
	}

	err = goSnmp.Connect()
	if err != nil {
		log.Err(err).Msg("SNMP connect failed")
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
				log.Err(err).Msg("SNMP connect failed")
			}
			log.Err(e).Msgf("could not get data for target '%s' + mib '%s'", target, mib)
			continue
		}
		if mibData != nil {
			data = append(data, *mibData)
		}
	}
	log.Info().Msgf("------ completed for target '%s'", target)
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
		log.Err(err).Msg("could not setup snmp v2c")
		return nil, errors.New("validation failed")
	}

	return &snmp.GoSNMP{
		Target:         target,
		Port:           161,
		Community:      community.Name,
		Version:        snmp.Version2c,
		Timeout:        time.Second * 2,
		MaxRepetitions: 2,
		MaxOids:        1,
	}, nil
}

func setupV3(target string, community *utils.SecurityData) (*snmp.GoSNMP, error) {
	err := validate(target, community)
	if err != nil {
		log.Err(err).Msg("could not setup snmp v3")
		return nil, errors.New("validation failed")
	}

	var msgFlags snmp.SnmpV3MsgFlags
	var authProtocol snmp.SnmpV3AuthProtocol
	var privProtocol snmp.SnmpV3PrivProtocol

	switch strings.ToLower(community.AuthProtocol) {
	case md5:
		authProtocol = snmp.MD5
	case sha:
		authProtocol = snmp.SHA
	default:
		log.Error().Msgf("could not setup snmp v3, unknown authentication protocol: %s",
			strings.ToLower(community.AuthProtocol))
		return nil, errors.New("unknown authentication protocol")
	}

	if community.PrivacyProtocol != "" {
		switch strings.ToLower(community.PrivacyProtocol) {
		case aes:
			privProtocol = snmp.AES
			// NoPriv, DES implemented, AES planned
			log.Warn().Msg("AES privacy protocol may be unsupported yet")
		case des:
			privProtocol = snmp.DES
		default:
			log.Error().Msgf("could not setup snmp v3, unknown privacy protocol: %s",
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
		Timeout:       time.Second * 2,
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
		log.Error().Msg("validation failed: target required")
		valid = false
	}
	if secData == nil {
		log.Error().Msg("validation failed: security data required")
		valid = false
	} else {
		if secData.Name == "" {
			log.Error().Msg("validation failed: name required")
			valid = false
		}
		if secData.AuthProtocol != "" && secData.AuthPassword == "" {
			log.Error().Msg("validation failed: authentication password required")
			valid = false
		}
		if secData.PrivacyProtocol != "" && secData.PrivacyPassword == "" {
			log.Error().Msg("validation failed: privacy password required")
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

	log.Info().Msgf("-- start getting MIB: %s", mib)

	snmpMetric := AvailableMetrics[mib]
	if snmpMetric == nil {
		return nil, errors.New("unsupported metric " + mib)
	}

	if goSnmp == nil {
		return nil, errors.New("snmp client is not configured")
	}
	if goSnmp.Conn == nil {
		log.Error().Msg("SNMP connect failed")
		return nil, errors.New("SNMP connect failed")
	}

	var data SnmpMetricData
	data.SnmpMetric = *snmpMetric

	walkHandler := func(dataUnit snmp.SnmpPDU) error {
		log.Info().Msgf("-- walk Handler: data unit name: '%s', value: '%s'",
			dataUnit.Name, dataUnit.Value)
		var val SnmpValue
		val.Name = dataUnit.Name
		switch v := dataUnit.Value.(type) {
		case uint:
			val.Value = int64(v)
			log.Info().Msgf("*** parsed value for %s: %d", val.Name, val.Value)
		case uint64:
			val.Value = int64(v)
			log.Info().Msgf("*** parsed value for %s: %d", val.Name, val.Value)
		default:
			log.Warn().Msgf("value '%s' of unsupported type for %s", v, dataUnit.Name)
		}
		data.Values = append(data.Values, val)
		return nil
	}

	oid := data.SnmpMetric.Oid
	err := goSnmp.Walk(oid, walkHandler)
	if err != nil {
		return nil, err
	} else {
		log.Info().Msgf("-- end getting MIB: %s", mib)
	}

	return &data, nil
}
