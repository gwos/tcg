package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"sync"

	"maps"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
	sdklog "github.com/gwos/tcg/sdk/log"
)

type APIClient struct {
	gwURI   string
	headers map[string]string
	once    sync.Once
}

func (c *APIClient) SendRequest(httpMethod string, requestURL string, headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {
	c.once.Do(func() {
		gwClient := clients.GWClient{
			AppName:      config.GetConfig().Connector.AppName,
			AppType:      config.GetConfig().Connector.AppType,
			GWConnection: config.GetConfig().GWConnections[0].AsClient(),
		}
		if err := gwClient.Connect(); err != nil {
			panic("aborting: " + err.Error())
		}
		token := reflect.ValueOf(&gwClient).Elem().FieldByName("token").String()
		c.headers = map[string]string{
			"Accept":         "application/json",
			"GWOS-APP-NAME":  gwClient.AppName,
			"GWOS-API-TOKEN": token,
		}
		c.gwURI = gwClient.GWConnection.HostName
	})
	hh := make(map[string]string, len(c.headers)+len(headers))
	maps.Copy(hh, c.headers)
	maps.Copy(hh, headers)
	return clients.SendRequest(httpMethod, c.gwURI+requestURL, hh, formValues, byteBody)
}

func (c *APIClient) CheckHostExist(host string, mustExist bool, mustHasStatus string) error {
	statusCode, byteResponse, err := c.SendRequest(http.MethodGet, "/api/hosts/"+host, nil, nil, nil)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelWarn, " -> Host exists")
	} else {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelWarn, " -> Host doesn't exist")
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
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelError, "could not remove host",
			slog.Any("error", err), slog.Int("code", code), slog.String("response", string(bb)))
	}
}

func (c *APIClient) RemoveAgent(agentID string) {
	if TestKeepInventory {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelWarn, "skip removing agent due to TestKeepInventory flag")
		return
	}
	code, bb, err := c.SendRequest(http.MethodDelete, "/api/agents/"+agentID, nil, nil, nil)
	if err != nil || code != 200 {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelError, "could not remove agent",
			slog.Any("error", err), slog.Int("code", code), slog.String("response", string(bb)))
	}
}
