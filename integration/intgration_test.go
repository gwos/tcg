package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"testing"
)

const (
	HostGetApi        = "http://localhost/api/hosts/"
	HostDeleteApi     = "http://localhost/api/hosts/"
	HostStatusPending = "PENDING"
	HostStatusUp      = "UP"
	TestHostName      = "GW8_TNG_TEST_HOST"
	ConfigEnv         = "TNG_CONFIG"
	ConfigName        = "config.yml"
)

type Response struct {
	HostName      string `json:"hostName"`
	MonitorStatus string `json:"monitorStatus"`
}

var headers map[string]string

func TestIntegration(t *testing.T) {
	var err error
	headers, err = config()
	if err != nil {
		t.Error(err)
	}

	err = existenceCheck(false, "irrelevant")
	if err != nil {
		t.Error(err)
	}

	err = runJavaSynchronizeInventoryTest()
	if err != nil {
		t.Error(err)
	}

	defer clean(headers)

	err = existenceCheck(true, HostStatusPending)

	err = runJavaSendResourceWithMetricsTest()
	if err != nil {
		t.Error(err)
	}

	err = existenceCheck(true, HostStatusUp)
}

func config() (map[string]string, error) {
	err := os.Setenv(ConfigEnv, path.Join("..", ConfigName))
	if err != nil {
		return nil, err
	}

	service := services.GetTransitService()

	err = service.Connect()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept":         "application/json",
		"GWOS-APP-NAME":  "gw8",
		"GWOS-API-TOKEN": service.GroundworkConfig.Token,
	}

	return headers, nil
}

func existenceCheck(mustExist bool, mustHasStatus string) error {
	statusCode, byteResponse, err := transit.SendRequest(http.MethodGet, HostGetApi+TestHostName, headers, nil, nil)
	if err != nil {
		return err
	}
	if !mustExist && statusCode == 404 {
		return nil
	}
	if !(!mustExist && statusCode != 404) {
		return errors.New(fmt.Sprintf("Status code = %d(Details: %s), want = %d", statusCode, string(byteResponse), 404))
	}

	var response Response

	err = json.Unmarshal(byteResponse, &response)
	if err != nil {
		return err
	}

	if mustExist && (response.HostName != TestHostName || response.MonitorStatus != mustHasStatus) {
		return errors.New(fmt.Sprintf("Host from database = (Name: %s, Status: %s), want = (Name: %s, Status: %s)",
			response.HostName, response.MonitorStatus, TestHostName, mustHasStatus))
	}

	return nil
}

func runJavaSynchronizeInventoryTest() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("mvn", "-Dtest=AppTest#shouldSynchronizeInventory", "test")
	cmd.Dir = path.Join(workDir, "../gw-transit")
	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func runJavaSendResourceWithMetricsTest() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("mvn", "-Dtest=AppTest#shouldSendResourceAndMetrics", "test")
	cmd.Dir = path.Join(workDir, "../gw-transit")
	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func clean(headers map[string]string) {
	_, _, err := transit.SendRequest(http.MethodDelete, HostDeleteApi+TestHostName, headers, nil, nil)
	if err != nil {
		log.Println(err)
	}

	cmd := exec.Command("rm", "-rf", "../gw-transit/src/main/resources/datastore")
	_, err = cmd.Output()
	if err != nil {
		log.Println(err)
	}
}
