package utils

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

const (
	colComm            = "comm"
	colName            = "name"
	colAuthProtocol    = "aprot"
	colAuthPassword    = "apass"
	colPrivacyProtocol = "pprot"
	colPrivacyPassword = "ppass"
)

var commColIdxMap = map[int]string{
	0: colComm,
	1: colName,
	2: colAuthProtocol,
	3: colAuthPassword,
	4: colPrivacyProtocol,
	5: colPrivacyPassword,
}

const commLinePattern = "comm(sec)?\\s+%s(\\z|\\s+)"

type SecurityData struct {
	Name            string
	AuthProtocol    string
	AuthPassword    string
	PrivacyProtocol string
	PrivacyPassword string
}

var confFilepath string

var doOnce sync.Once

func GetSecurityData(community string) (*SecurityData, error) {
	doOnce.Do(func() {
		retrieveConfFilepath()
	})
	log.Debug().Msgf("reading config from file: %snedi.conf", confFilepath)

	file, err := os.Open(confFilepath + "nedi.conf")
	if err != nil {
		log.Err(err).Msg("could not open nedi.conf")
		return nil, errors.New("failed to open nedi.conf")
	}
	defer file.Close()

	pattern := fmt.Sprintf(commLinePattern, community)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches, err := regexp.MatchString(pattern, line)
		if err != nil {
			log.Err(err).Msgf("could not match line '%s'", line)
			continue
		}
		if matches {
			var commDataMap = map[string]string{
				colComm:            "",
				colName:            "",
				colAuthProtocol:    "",
				colAuthPassword:    "",
				colPrivacyProtocol: "",
				colPrivacyPassword: "",
			}
			values := strings.Split(line, "\t")
			colIdx := 0
			for _, value := range values {
				v := strings.ReplaceAll(value, " ", "")
				if v == "" {
					continue
				}
				fieldName := commColIdxMap[colIdx]
				commDataMap[fieldName] = v
				colIdx++
			}
			aPass := commDataMap[colAuthPassword]
			pPass := commDataMap[colPrivacyPassword]
			if commDataMap[colComm] == "commsec" {
				if aPass != "" {
					aPass, err = Decrypt(aPass)
					if err != nil {
						log.Err(err).Msg("could not decrypt authentication password")
						return nil, errors.New("failed to decrypt authentication password")
					}
				}
				if pPass != "" {
					pPass, err = Decrypt(pPass)
					if err != nil {
						log.Err(err).Msg("could not decrypt privacy password")
						return nil, errors.New("failed to decrypt privacy password")
					}
				}
			}
			secData := SecurityData{
				Name:            commDataMap[colName],
				AuthProtocol:    commDataMap[colAuthProtocol],
				AuthPassword:    aPass,
				PrivacyProtocol: commDataMap[colPrivacyProtocol],
				PrivacyPassword: pPass,
			}
			return &secData, nil
		}
	}

	if err := scanner.Err(); err != nil {
		log.Err(err).Msg("could not read nedi.conf")
		return nil, errors.New("failed to read nedi.conf")
	}

	return nil, errors.New("no info found for community")
}

// Decrypt wraps XORpass
func Decrypt(s string) (string, error) {
	enc, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(XORpass(enc)), nil
}

// Encrypt wraps XORpass
func Encrypt(s string) string {
	return hex.EncodeToString(XORpass([]byte(s)))
}

// XORpass implements N XORpass from NeDi
func XORpass(s []byte) []byte {
	k, r := []byte("change for more security"), []byte{}
	if v, ok := os.LookupEnv("NEDI_ENCRYPT_KEY"); ok {
		k = []byte(v)
	}
	for _, ch := range s {
		i := k[len(k)-1]
		r = append(r, byte(int(ch)^int(i)))
		k = append([]byte{i}, k[:len(k)-1]...)
	}
	return r
}

func retrieveConfFilepath() {
	nediConfFilepath := os.Getenv("NEDI_CONF_PATH")
	if nediConfFilepath == "" {
		log.Warn().Msg("env variable 'NEDI_CONF_PATH' is not set")
		return
	}
	if !strings.HasSuffix(nediConfFilepath, "/") {
		nediConfFilepath = nediConfFilepath + "/"
	}
	confFilepath = nediConfFilepath
}
