package integration

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/clients"
	. "github.com/gwos/tng/config"
	"github.com/gwos/tng/log"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
)

const (
	HostGetAPI        = "http://localhost/api/hosts/"
	HostDeleteAPI     = "http://localhost/api/hosts/"
	HostStatusPending = "PENDING"
	HostStatusUp      = "UP"
	TestHostName      = "GW8_TNG_TEST_HOST"
)

type Response struct {
	HostName      string `json:"hostName"`
	MonitorStatus string `json:"monitorStatus"`
}

var headers map[string]string

func TestIntegration(t *testing.T) {
	var err error
	headers, err = config(t)
	defer clean(headers)
	assert.NoError(t, err)

	err = existenceCheck(false, "irrelevant")
	assert.NoError(t, err)

	err = installDependencies()
	assert.NoError(t, err)

	err = runJavaSynchronizeInventoryTest()
	assert.NoError(t, err)

	err = existenceCheck(true, HostStatusPending)
	assert.NoError(t, err)

	err = runJavaSendResourceWithMetricsTest()
	assert.NoError(t, err)

	err = existenceCheck(true, HostStatusUp)
	assert.NoError(t, err)
}

func config(t *testing.T) (map[string]string, error) {
	err := os.Setenv(ConfigEnv, path.Join("..", ConfigName))
	assert.NoError(t, err)

	gwClient := &clients.GWClient{GWConfig: GetConfig().GWConfigs[0]}
	err = gwClient.Connect()
	assert.NoError(t, err)

	token := reflect.ValueOf(gwClient).Elem().FieldByName("token").String()
	headers := map[string]string{
		"Accept":         "application/json",
		"GWOS-APP-NAME":  gwClient.GWConfig.AppName,
		"GWOS-API-TOKEN": token,
	}

	return headers, nil
}

func existenceCheck(mustExist bool, mustHasStatus string) error {
	statusCode, byteResponse, err := clients.SendRequest(http.MethodGet, HostGetAPI+TestHostName, headers, nil, nil)
	if err != nil {
		return err
	}
	if !mustExist && statusCode == 404 {
		return nil
	}
	if !(mustExist && statusCode == 200) {
		return fmt.Errorf("Status code = %d (Details: %s), want = %d ", statusCode, string(byteResponse), 404)
	}

	var response Response

	err = json.Unmarshal(byteResponse, &response)
	if err != nil {
		return err
	}

	if mustExist && (response.HostName != TestHostName || response.MonitorStatus != mustHasStatus) {
		return fmt.Errorf("Host from database = (Name: %s, Status: %s), want = (Name: %s, Status: %s)",
			response.HostName, response.MonitorStatus, TestHostName, mustHasStatus)
	}

	return nil
}

func installDependencies() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("mvn", "install:install-file", "-Dfile=lib/collagerest-common-8.0.0-SNAPSHOT.jar", "-DgroupId=org.groundwork", "-DartifactId=collagerest-common", "-Dversion=8.0.0-SNAPSHOT")
	cmd.Dir = path.Join(workDir, "../gw-transit")
	_, err = cmd.Output()
	if err != nil {
		return err
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
	_, _, err := clients.SendRequest(http.MethodDelete, HostDeleteAPI+TestHostName, headers, nil, nil)
	if err != nil {
		log.Error(err.Error())
	}

	cmd := exec.Command("rm", "-rf", "../gw-transit/src/main/resources/datastore")
	_, err = cmd.Output()
	if err != nil {
		log.Error(err.Error())
	}
}
