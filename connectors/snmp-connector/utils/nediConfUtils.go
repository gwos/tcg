package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/gwos/tcg/log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
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

const scriptFilepath = "utils/xorp.pl"

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
	log.Debug("|nediConfUtils.go| : [GetSecurityData]: Reading config from file: ", confFilepath, "nedi.conf")

	file, err := os.Open(confFilepath + "nedi.conf")
	if err != nil {
		log.Error("|nediConfUtils.go| : [GetSecurityData]: Failed to open nedi.conf: ", err)
		return nil, errors.New("failed to open nedi.conf")
	}
	defer file.Close()

	pattern := fmt.Sprintf(commLinePattern, community)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches, err := regexp.MatchString(pattern, line)
		if err != nil {
			log.Error("|nediConfUtils.go| : [GetSecurityData]: Failed to match line '", line, "': ", err)
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
					aPass, err = decrypt(aPass)
					if err != nil {
						log.Error("|nediConfUtils.go| : [GetSecurityData]: Failed to decrypt authentication password: ", err)
						return nil, errors.New("failed to decrypt authentication password")
					}
				}
				if pPass != "" {
					pPass, err = decrypt(pPass)
					if err != nil {
						log.Error("|nediConfUtils.go| : [GetSecurityData]: Failed to decrypt privacy password: ", err)
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
		log.Error("|nediConfUtils.go| : [GetSecurityData]: Failed to read nedi.conf: ", err)
		return nil, errors.New("failed to read nedi.conf")
	}

	return nil, errors.New("no info found for community")
}

func decrypt(encrypted string) (string, error) {
	// TODO: rewrite it without invoking perl
	cmd := exec.Command("perl", scriptFilepath, encrypted)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	if err != nil {
		log.Error("|nediConfUtils.go| : [decrypt]: Failed to run command '", cmd, "': ", err)
		return "", errors.New("failed to run script to decrypt")
	}
	if len(errOut.Bytes()) > 0 {
		log.Error("|nediConfUtils.go| : [decrypt]: Error when running script to decrypt: ", string(errOut.Bytes()))
		return "", errors.New("error when running script to decrypt")
	}
	return out.String(), nil
}

func retrieveConfFilepath() {
	nediConfFilepath := os.Getenv("NEDI_CONF_PATH")
	if nediConfFilepath == "" {
		log.Warn("|nediConfUtils.go| : [retrieveConfFilepath]: Env variable 'NEDI_CONF_PATH' is not set")
		return
	}
	if !strings.HasSuffix(nediConfFilepath, "/") {
		nediConfFilepath = nediConfFilepath + "/"
	}
	confFilepath = nediConfFilepath
}
