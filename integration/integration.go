package integration

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gwos/tcg/config"
	sdklog "github.com/gwos/tcg/sdk/log"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

const (
	dynInventoryFalse = false
	dynInventoryTrue  = true
	natsAckWait5s     = time.Second * 5
	natsAckWait30s    = time.Second * 30

	testName = "test.tcg.gw8"
)

var TestConfigDefaults = map[string]string{
	"TCG_CONNECTOR_AGENTID":          "INTEGRATION-TEST",
	"TCG_CONNECTOR_APPNAME":          "INTEGRATION-TEST",
	"TCG_CONNECTOR_APPTYPE":          "VEMA",
	"TCG_CONNECTOR_ENABLED":          "true",
	"TCG_CONNECTOR_NATSFILESTOREDIR": "natsstore.test",
	"TCG_GWCONNECTIONS_0_ENABLED":    "true",
	"TCG_GWCONNECTIONS_0_HOSTNAME":   "https://localhost",
	"TCG_GWCONNECTIONS_0_PASSWORD":   "",
	"TCG_GWCONNECTIONS_0_USERNAME":   "",
}

var (
	TestFlagClient     = false
	TestFlagLogger     = false
	TestLoopMetrics    = 4
	TestMessagesCount  = 4
	TestResourcesCount = 20
	TestServicesCount  = 50

	_ = lookupEnv("TEST_FLAG_CLIENT", &TestFlagClient)
	_ = lookupEnv("TEST_FLAG_LOGGER", &TestFlagLogger)
	_ = lookupEnv("TEST_LOOP_METRICS", &TestLoopMetrics)
	_ = lookupEnv("TEST_MESSAGES_COUNT", &TestMessagesCount)
	_ = lookupEnv("TEST_RESOURCES_COUNT", &TestResourcesCount)
	_ = lookupEnv("TEST_SERVICES_COUNT", &TestServicesCount)
)

func lookupEnv(env string, arg any) bool {
	if s, ok := os.LookupEnv(env); ok {
		switch arg := arg.(type) {
		case *bool:
			v, err := strconv.ParseBool(s)
			*arg = err == nil && v
			return true
		case *int:
			if v, err := strconv.Atoi(s); err == nil {
				*arg = v
				return true
			}
		case *string:
			*arg = s
			return true
		}
	}
	return false
}

var apiClient = new(APIClient)

func setupIntegration(t testing.TB, natsAckWait time.Duration, isDynamicInventory bool) {
	for k, v := range TestConfigDefaults {
		if _, ok := os.LookupEnv(k); !ok {
			t.Setenv(k, v)
		}
	}
	if len(os.Getenv("TCG_GWCONNECTIONS_0_USERNAME")) == 0 ||
		len(os.Getenv("TCG_GWCONNECTIONS_0_PASSWORD")) == 0 {
		t.Errorf("[setupIntegration]: Provide environment variables for Groundwork Connection: %s and %s",
			"TCG_GWCONNECTIONS_0_USERNAME", "TCG_GWCONNECTIONS_0_PASSWORD")
		t.SkipNow()
	}

	cfg := config.GetConfig()
	cfg.Connector.NatsAckWait = natsAckWait
	cfg.GWConnections[0].IsDynamicInventory = isDynamicInventory

	if TestFlagLogger {
		// test for memory usage without zerolog integration leyer in clients
		sdklog.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}).WithGroup("tcg.sdk"))
	}

	service := services.GetTransitService()
	assert.NoError(t, service.StopNats())
	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())
	t.Log("[setupIntegration]: ", service.Status())
	t.Logf("cfg.Connector: %+v", cfg.Connector)
	t.Logf("cfg.GWConnections[0]: %+v", cfg.GWConnections[0])
}

func cleanNats(t testing.TB) {
	service := services.GetTransitService()
	assert.NoError(t, service.StopNats())
	assert.NoError(t, service.ResetNats())
	_ = os.Remove(filepath.Join(service.NatsStoreDir, "inventory.json"))
	_ = os.Remove(filepath.Join(service.NatsStoreDir, "inventory1.json"))
	assert.NoError(t, os.Remove(service.NatsStoreDir))
	t.Log("[cleanNats]: ", service.Status())
}

func makeResource(rsIdx, svcCount int) transit.MonitoredResource {
	rs := new(transit.MonitoredResource)
	rs.Status = transit.HostUp
	rs.Type = transit.ResourceTypeHost
	rs.LastCheckTime = transit.NewTimestamp()
	rs.NextCheckTime = transit.NewTimestamp()
	*rs.NextCheckTime = rs.NextCheckTime.Add(time.Minute * 60)
	rs.Name = fmt.Sprintf("host%v.%v", rsIdx, testName)
	rs.Device = rs.Name
	rs.Description = strings.Join(
		append([]string{strings.ToUpper(rs.Name)}, randStrs(rsIdx)...), " ")

	for i := 0; i < svcCount; i++ {
		svc := new(transit.MonitoredService)
		svc.Name = fmt.Sprintf("service%v.%v", i, rs.Name)
		svc.Owner = rs.Name
		svc.Status = transit.ServiceOk
		svc.Type = transit.ResourceTypeService
		svc.LastCheckTime = transit.NewTimestamp()
		svc.NextCheckTime = transit.NewTimestamp()
		*svc.NextCheckTime = svc.NextCheckTime.Add(time.Minute * 60)

		svc.LastPluginOutput = strings.Join(randStrs(i), " ")
		svc.Description = strings.Join(
			append([]string{strings.ToUpper(svc.Name)}, randStrs(i+rsIdx)...), " ")

		m := new(transit.TimeSeries)
		m.Interval = new(transit.TimeInterval)
		m.Interval.StartTime = transit.NewTimestamp()
		m.Interval.EndTime = transit.NewTimestamp()
		m.MetricName = "test_metric"
		m.SampleType = transit.Value
		m.Value = transit.NewTypedValue(i)
		m.Unit = transit.MB

		svc.Metrics = append(svc.Metrics, *m)
		rs.Services = append(rs.Services, *svc)
	}

	return *rs
}

func randStrs(x ...int) []string {
	dict := []string{
		string(transit.ResourceTypeHost),
		string(transit.ResourceTypeService),
		string(transit.ResourceTypeHypervisor),
		string(transit.ResourceTypeInstance),
		string(transit.ResourceTypeVirtualMachine),
		string(transit.ResourceTypeCloudApp),
		string(transit.ResourceTypeCloudFunction),
		string(transit.ResourceTypeLoadBalancer),
		string(transit.ResourceTypeContainer),
		string(transit.ResourceTypeStorage),
		string(transit.ResourceTypeNetwork),
		string(transit.ResourceTypeNetworkSwitch),
		string(transit.ResourceTypeNetworkDevice),
	}
	x = append(x, rand.Intn(len(dict)), rand.Intn(len(dict)))
	n, m := x[0], x[1]
	i := n % len(dict)
	dict = append(dict, dict...)
	return dict[i : i+m]
}
