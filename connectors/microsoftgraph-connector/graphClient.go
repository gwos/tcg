package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func login(tenantID string, clientID string, clientSecret string, resource string) (string, error) {
	var request *http.Request
	var response *http.Response

	endPoint := "https://login.microsoftonline.com/" + tenantID + "/oauth2/token"
	auth := AuthRecord{
		GrantType:    "client_credentials",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Resource:     resource,
	}
	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	form.Add("client_secret", auth.ClientSecret)
	form.Add("client_id", auth.ClientID)
	form.Add("resource", auth.Resource)
	byteBody := []byte(form.Encode())
	var body io.Reader
	body = bytes.NewBuffer(byteBody)
	request, err := http.NewRequest(http.MethodPost, endPoint, body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err = Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	v := interface{}(nil)
	json.Unmarshal(responseBody, &v)
	token, err := jsonpath.Get("$.access_token", v)
	if err != nil {
		return "", err
	}
	return token.(string), nil
}

func Initialize() error {
	if officeToken != "" {
		return nil
	}
	token, err := login(tenantID, clientID, clientSecret, officeResource)
	if err != nil {
		return nil
	}
	officeToken = token
	token, err = login(tenantID, clientID, clientSecret, graphResource)
	if err != nil {
		return nil
	}
	graphToken = token
	log.Info(fmt.Sprintf("initialized MS Graph connection with  %s and %s", officeResource, graphResource))
	return nil
}

func ExecuteRequest(graphUri string, token string) ([]byte, error) {
	request, _ := http.NewRequest("GET", graphUri, nil)
	request.Header.Set(	"accept", "application/json; odata.metadata=full")
	request.Header.Set("Authorization", "Bearer " + token)
	response, error := httpClient.Do(request)
	if error != nil {
		return nil, error
	}
	if response.StatusCode != 200 {
		log.Info("[MSGraph Connector]:  Retrying Authentication...")
		response.Body.Close()
		isOfficeToken := false
		if token == officeToken {
			isOfficeToken = true
		}
		officeToken = ""
		graphToken = ""
		Initialize()
		newToken := graphToken
		if isOfficeToken {
			newToken = officeToken
		}
		request.Header.Set("Authorization", "Bearer " + newToken)
		response, error = httpClient.Do(request)
		if error != nil {
			return nil, error
		}
	}
	body, error:= ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if error != nil {
		return nil, error
	}
	return body, nil
}

func Do(request *http.Request) (*http.Response, error) {
	// TODO: retry logics
	return httpClient.Do(request)
}

func parseError(v interface{}) error {
	msg, err := jsonpath.Get("$.error.message", v)
	if err != nil {
		log.Error(err)
		return err
	}
	if msg != nil {
		log.Error(msg)
		return errors.New(msg.(string))
	}
	return nil
}


func createMetric(name string, suffix string, value interface{}) *transit.TimeSeries {
	return createMetricWithThresholds(name, suffix, value, -1, -1)
}

func createMetricWithThresholds(name string, suffix string, value interface{}, warning float64, critical float64) *transit.TimeSeries {
	metricBuilder := connectors.MetricBuilder{
		Name:       name + suffix,
		Value:      value,
		UnitType:   transit.UnitCounter,
		Warning:  warning,
		Critical: critical,
		Graphed: true, // TODO: get this value from configs
	}
	metric, err := connectors.BuildMetric(metricBuilder)
	if err != nil {
		log.Error("failed to build metric " + metricBuilder.Name)
		return nil
	}
	return metric
}

func getCount(v interface{}) (int, error) {
	var count int = 0
	if v != nil {
		value, err := jsonpath.Get("$.value[*]", v)
		if err != nil {
			return count, err
		}
		if value != nil {
			count = len(value.([]interface{}))
			if count == 0 {
				err := parseError(v)
				if err != nil {
					return 0, err
				}
			}
		}
	}
	return count, nil
}
