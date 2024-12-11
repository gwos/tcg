package clients

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
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
	HostName string `yaml:"hostName"`
}

// DSClient implements DS API operations
type DSClient struct {
	DSConnection
}

// ValidateToken calls API
func (client *DSClient) ValidateToken(appName, apiToken string) error {
	if len(client.HostName) == 0 {
		sdklog.Logger.Info("DSClient is not configured")
		return nil
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}
	formValues := map[string]string{
		"gwos-app-name":  appName,
		"gwos-api-token": apiToken,
	}
	entrypoint := url.URL{
		Scheme: makeDalekServicesScheme(client.HostName),
		Host:   client.HostName,
		Path:   DSEntrypointValidateToken,
	}
	req := Req{
		URL:     entrypoint.String(),
		Method:  http.MethodPost,
		Headers: headers,
		Form:    formValues,
	}
	err := req.Send()

	if err == nil {
		if req.Status == 201 {
			if b, e := strconv.ParseBool(string(req.Response)); e == nil && b {
				sdklog.Logger.LogAttrs(context.Background(), slog.LevelDebug, "validate token", req.LogAttrs()...)
				return nil
			}
			eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, "invalid gwos-app-name or gwos-api-token")
			req.Err = eee
			sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not validate token", req.LogAttrs()...)
			return eee
		}
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not validate token", req.Details()...)
		return eee
	}
	sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not validate token", req.LogAttrs()...)
	return err
}

// Reload calls API
func (client *DSClient) Reload(agentID string) error {
	if len(client.HostName) == 0 {
		sdklog.Logger.Info("DSClient is not configured")
		return nil
	}
	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
	entrypoint := url.URL{
		Scheme: makeDalekServicesScheme(client.HostName),
		Host:   client.HostName,
		Path:   strings.ReplaceAll(DSEntrypointReload, ":agentID", agentID),
	}
	req := Req{
		URL:     entrypoint.String(),
		Method:  http.MethodPost,
		Headers: headers,
	}
	err := req.Send()

	if err == nil {
		if req.Status == 201 {
			sdklog.Logger.LogAttrs(context.Background(), slog.LevelDebug, "request for reload", req.LogAttrs()...)
			return nil
		}
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		if req.Status == 404 {
			req.Err = eee
			sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not request for reload: check AgentID", req.LogAttrs()...)
		}
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not request for reload", req.Details()...)
		return eee
	}
	sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not request for reload", req.LogAttrs()...)
	return err
}

// Create the scheme (http or https) based on hostName prefix
func makeDalekServicesScheme(hostName string) string {
	scheme := "https"
	if strings.HasPrefix(hostName, "dalekservices") {
		scheme = "http"
	}
	return scheme
}
