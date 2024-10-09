<p>
  <a href="http://www.gwos.com/" target="blank"><img src="../.github/img/readme_image.png" alt="GWOS Logo"/></a>
</p>

[![License](https://img.shields.io/github/license/gwos/tcg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/gwos/tcg/sdk?status.svg)](https://godoc.org/github.com/gwos/tcg/sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/gwos/tcg/sdk)](https://goreportcard.com/report/github.com/gwos/tcg/sdk)

# TCG SDK

SDK provides data structures, clients, and featured packages for integration with Groundwork Monitoring.

Known integrations:
- [TCG itself](https://github.com/gwos/tcg)
- [Telegraf output plugin](https://pkg.go.dev/github.com/influxdata/telegraf/plugins/outputs/groundwork)
- Diamanti Picasa collector


<a name="dependencies"></a>
## Dependencies

No 3rd party dependencies.
Only standard library imports allowed in this module to simplify integrations and prevent conflicts of dependencies with different versions.


<a name="envvar"></a>
## Environment variables

SDK HTTP clients processed environment variables:
- `TCG_TLS_CLIENT_INSECURE` [InsecureSkipVerify in tls.Config](https://pkg.go.dev/crypto/tls#Config), __false__ by default.
- `TCG_HTTP_CLIENT_TIMEOUT` [Timeout in http.Client](https://pkg.go.dev/net/http#Client), __5s__ by default.
- `TCG_HTTP_CLIENT_TIMEOUT_GW`, same as `TCG_HTTP_CLIENT_TIMEOUT` on communication with Groundwork Foundation server, __40s__ by default.


<a name="logging"></a>
## Logging

SDK module uses [log/slog](https://pkg.go.dev/log/slog) based logger. In oder to capture it [implement custom Handler](https://pkg.go.dev/log/slog#hdr-Writing_a_handler) and assign to module [Logger](https://pkg.go.dev/github.com/gwos/tcg/sdk/log#pkg-variables).

For example, see logger adapter in [Telegraf output plugin](https://pkg.go.dev/github.com/influxdata/telegraf/plugins/outputs/groundwork).
