package integration

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"net/http"
	"reflect"
	"sync"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
)

type APIClient struct {
	gwURI   string
	headers map[string]string
	once    sync.Once
}

func (c *APIClient) SendRequest(httpMethod string, requestURL string, headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {
	c.once.Do(func() {
		gwClient := new(clients.GWClient)
		gwClient.AppName = config.GetConfig().Connector.AppName
		gwClient.AppType = config.GetConfig().Connector.AppType
		gwClient.GWConnection = (*clients.GWConnection)(config.GetConfig().GWConnections[0])
		if err := gwClient.Connect(); err != nil {
			panic("aborting: " + err.Error())
		}
		token := reflect.ValueOf(gwClient).Elem().FieldByName("token").String()
		c.headers = map[string]string{
			"Accept":         "application/json",
			"GWOS-APP-NAME":  gwClient.AppName,
			"GWOS-API-TOKEN": token,
		}
		c.gwURI = gwClient.GWConnection.HostName
	})
	hh := make(map[string]string, len(c.headers)+len(headers))
	for k, v := range c.headers {
		hh[k] = v
	}
	for k, v := range headers {
		hh[k] = v
	}
	return clients.SendRequest(httpMethod, c.gwURI+requestURL, hh, formValues, byteBody)
}

func (c *APIClient) CheckHostExist(host string, mustExist bool, mustHasStatus string) error {
	statusCode, byteResponse, err := c.SendRequest(http.MethodGet, "/api/hosts/"+host, nil, nil, nil)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		stdlog.Print(" -> Host exists")
	} else {
		stdlog.Print(" -> Host doesn't exist")
	}

	if !mustExist && statusCode == 404 {
		return nil
	}
	if !(mustExist && statusCode == 200) {
		return fmt.Errorf("status code = %d (Details: %s), want = %d ", statusCode, string(byteResponse), 200)
	}

	response := new(struct {
		HostName      string `json:"hostName"`
		MonitorStatus string `json:"monitorStatus"`
	})
	if err := json.Unmarshal(byteResponse, response); err != nil {
		return err
	}

	if mustExist && (response.HostName != host || response.MonitorStatus != mustHasStatus) {
		return fmt.Errorf("host from database = (Name: %s, Status: %s), want = (Name: %s, Status: %s)",
			response.HostName, response.MonitorStatus, host, mustHasStatus)
	}

	return nil
}

func (c *APIClient) RemoveHost(hostname string) {
	code, bb, err := c.SendRequest(http.MethodDelete, "/api/hosts/"+hostname, nil, nil, nil)
	if err != nil || code != 200 {
		stdlog.Printf("could not remove host: %v [%v] %v", err, code, string(bb))
	}
}

func (c *APIClient) RemoveAgent(agentID string) {
	code, bb, err := c.SendRequest(http.MethodDelete, "/api/agents/"+agentID, nil, nil, nil)
	if err != nil || code != 200 {
		stdlog.Printf("could not remove agent: %v [%v] %v", err, code, string(bb))
	}
}
