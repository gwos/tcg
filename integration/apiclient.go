package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gwos/tcg/sdk/clients"
	tcgerr "github.com/gwos/tcg/sdk/errors"
	sdklog "github.com/gwos/tcg/sdk/log"
)

type APIClient struct {
	clients.GWClient
}

func (c *APIClient) CheckHostExist(host string, mustExist bool, mustHasStatus string) error {
	bb, err := c.SendRequest(context.TODO(), http.MethodGet, clients.GWEntrypoint("/api/hosts/"+host), "", nil)
	if errors.Is(err, tcgerr.ErrNotFound) {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelError, " -> Host doesn't exist")
		return nil
	}
	if err != nil {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelError, " -> could not check Host")
		return err
	}
	sdklog.Logger.LogAttrs(context.TODO(), slog.LevelWarn, " -> Host exists")

	var response struct {
		HostName      string `json:"hostName"`
		MonitorStatus string `json:"monitorStatus"`
	}
	if err := json.Unmarshal(bb, &response); err != nil {
		return err
	}

	if mustExist && (response.HostName != host || response.MonitorStatus != mustHasStatus) {
		return fmt.Errorf("host from database = (Name: %s, Status: %s), want = (Name: %s, Status: %s)",
			response.HostName, response.MonitorStatus, host, mustHasStatus)
	}

	return nil
}

func (c *APIClient) RemoveHost(hostname string) {
	_, err := c.SendRequest(context.TODO(), http.MethodDelete, clients.GWEntrypoint("/api/hosts/"+hostname), "", nil)
	if err != nil {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelError, "could not remove host", slog.String("hostname", hostname))
	}
}

func (c *APIClient) RemoveAgent(agentID string) {
	if TestKeepInventory {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelWarn, "skip removing agent due to TestKeepInventory flag")
		return
	}
	_, err := c.SendRequest(context.TODO(), http.MethodDelete, clients.GWEntrypoint("/api/agents/"+agentID), "", nil)
	if err != nil {
		sdklog.Logger.LogAttrs(context.TODO(), slog.LevelError, "could not remove agent", slog.String("agentID", agentID))
	}
}
