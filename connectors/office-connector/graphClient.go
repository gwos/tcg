package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

const (
	maxRetries = 5
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func login(tenantID, clientID, clientSecret, resource string) (str string, err error) {
	var (
		responseBody []byte
		body         io.Reader
		token        interface{}
		v            interface{}
		request      *http.Request
		response     *http.Response
	)

	endPoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/token", tenantID)

	auth := AuthRecord{
		GrantType:    "client_credentials",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Resource:     resource,
	}

	form := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_secret": []string{auth.ClientSecret},
		"client_id":     []string{auth.ClientID},
		"resource":      []string{auth.Resource},
	}

	body = bytes.NewBuffer([]byte(form.Encode()))
	if request, err = http.NewRequest(http.MethodPost, endPoint, body); err == nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if response, err = Do(request); err != nil {
			return
		}
	} else {
		return
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if responseBody, err = ioutil.ReadAll(response.Body); err == nil {
		_ = json.Unmarshal(responseBody, &v)
		if token, err = jsonpath.Get("$.access_token", v); err == nil {
			str = token.(string)
		}
	}

	return
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
	log.Info().Msgf("initialized MS Graph connection with  %s and %s", officeResource, graphResource)
	return nil
}

func ExecuteRequest(graphUri, token string) ([]byte, error) {
	request, _ := http.NewRequest("GET", graphUri, nil)
	request.Header.Set("accept", "application/json; odata.metadata=full")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		log.Info().Msg("Retrying Authentication...")
		_ = response.Body.Close()
		isOfficeToken := false
		if token == officeToken {
			isOfficeToken = true
		}
		officeToken = ""
		graphToken = ""
		_ = Initialize()
		newToken := graphToken
		if isOfficeToken {
			newToken = officeToken
		}
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", newToken))
		response, err = httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != 200 {
			b, _ := ioutil.ReadAll(response.Body)
			log.Debug().Msg(fmt.Sprintf("[url=%s][response=%s]", graphUri, string(b)))
			_ = response.Body.Close()
			return nil, errors.New(fmt.Sprintf("error to get data. [url: %s, status code: %d", graphUri, response.StatusCode))
		}
	}
	defer func() {
		_ = response.Body.Close()
	}()
	return ioutil.ReadAll(response.Body)
}

func Do(request *http.Request) (*http.Response, error) {
	response, err := httpClient.Do(request)
	var counter = 1
	for response.StatusCode != 200 && response.StatusCode != 401 {
		time.Sleep(time.Duration(counter) * time.Second)
		response, err = httpClient.Do(request)
		counter++
		if counter == maxRetries+1 {
			return response, err
		}
	}
	return response, err
}

func parseError(v interface{}) error {
	msg, err := jsonpath.Get("$.error.message", v)
	if err != nil {
		log.Err(err).Msg("could not parse")
		return err
	}
	if msg != nil {
		log.Error().Interface("error", msg).
			Msg("could not parse")
		return errors.New(msg.(string))
	}
	return nil
}

func createMetric(name, suffix string, value interface{}) *transit.TimeSeries {
	return createMetricWithThresholds(name, suffix, value, -1, -1)
}

func createMetricWithThresholds(name string, suffix string, value interface{}, warning float64, critical float64) *transit.TimeSeries {
	metricBuilder := connectors.MetricBuilder{
		Name:     fmt.Sprintf("%s%s", name, suffix),
		Value:    value,
		UnitType: transit.UnitCounter,
		Warning:  warning,
		Critical: critical,
		Graphed:  true, // TODO: get this value from configs
	}
	metric, err := connectors.BuildMetric(metricBuilder)
	if err != nil {
		log.Error().
			Str("metric", metricBuilder.Name).
			Msg("failed to build metric")
		return nil
	}
	return metric
}

func getCount(v interface{}) (c int, err error) {
	var (
		value interface{}
	)
	if v != nil {
		if value, err = jsonpath.Get("$.value[*]", v); err == nil {
			if value != nil {
				c = len(value.([]interface{}))
				if c == 0 && parseError(v) != nil {
					return
				}
			}
		}
	}
	return
}
