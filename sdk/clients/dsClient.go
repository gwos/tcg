package clients

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	tcgerr "github.com/gwos/tcg/sdk/errors"
	sdklog "github.com/gwos/tcg/sdk/log"
)

// Define entrypoints for DSOperations
const (
	DSEntrypointReload        = "/dalekservices/connectors/reload/:agentID"
	DSEntrypointValidateToken = "/dalekservices/validate-token"
)

// DSConnection defines DalekServices Connection configuration
type DSConnection struct {
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName string `env:"HOSTNAME" yaml:"hostName"`
}

// DSClient implements DS API operations
type DSClient struct {
	DSConnection
}

// url makes the DalekServices URL for path
func (client *DSClient) url(path string) string {
	u := url.URL{
		Scheme: makeDalekServicesScheme(client.HostName),
		Host:   client.HostName,
		Path:   path,
	}
	return u.String()
}

// ValidateToken calls API
func (client *DSClient) ValidateToken(appName, apiToken string) error {
	if len(client.HostName) == 0 {
		sdklog.Logger.Info("DSClient is not configured")
		return nil
	}

	ctx := context.Background()
	req := Req{
		URL:     client.url(DSEntrypointValidateToken),
		Method:  http.MethodPost,
		Headers: formHeaders(),
		Form: map[string]string{
			"gwos-app-name":  appName,
			"gwos-api-token": apiToken,
		},
	}
	if err := req.Send(); err != nil {
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not validate token", req.LogAttrs()...)
		return err
	}
	return checkTokenResponse(ctx, &req, http.StatusCreated, "validate token")
}

// Reload calls API
func (client *DSClient) Reload(agentID string) error {
	if len(client.HostName) == 0 {
		sdklog.Logger.Info("DSClient is not configured")
		return nil
	}

	ctx := context.Background()
	req := Req{
		URL:     client.url(strings.ReplaceAll(DSEntrypointReload, ":agentID", agentID)),
		Method:  http.MethodPost,
		Headers: jsonHeaders(),
	}
	if err := req.Send(); err != nil {
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not request for reload", req.LogAttrs()...)
		return err
	}
	switch req.Status {
	case http.StatusCreated:
		sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "request for reload", req.LogAttrs()...)
		return nil
	case http.StatusNotFound:
		req.Err = fmt.Errorf("%w: %v", tcgerr.ErrNotFound, string(req.Response))
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not request for reload: check AgentID", req.LogAttrs()...)
		return req.Err
	default:
		req.Err = fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not request for reload", req.Details()...)
		return req.Err
	}
}

// Create the scheme (http or https) based on hostName prefix
func makeDalekServicesScheme(hostName string) string {
	if strings.HasPrefix(hostName, "dalekservices") {
		return "http"
	}
	return "https"
}
