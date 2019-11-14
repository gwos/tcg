package integration

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/services"
	"log"
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
	defer clean(headers)
	if err != nil {
		t.Error(err)
	}

	err = existenceCheck(false, "irrelevant")
	if err != nil {
		t.Error(err)
		return
	}

	err = installDependencies()
	if err != nil {
		t.Error(err)
		return
	}

	err = runJavaSynchronizeInventoryTest()
	if err != nil {
		t.Error(err)
		return
	}

	err = existenceCheck(true, HostStatusPending)

	err = runJavaSendResourceWithMetricsTest()
	if err != nil {
		t.Error(err)
		return
	}

	err = existenceCheck(true, HostStatusUp)
	if err != nil {
		t.Error(err)
		return
	}
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

	token := reflect.ValueOf(service).Elem().FieldByName("token").String()
	headers := map[string]string{
		"Accept":         "application/json",
		"GWOS-APP-NAME":  service.GroundworkConfig.AppName,
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

	cmd1 := exec.Command("mvn", "-Dtest=AppTest#shouldSynchronizeInventory", "test")
	cmd1.Dir = path.Join(workDir, "../gw-transit")
	_, err = cmd1.Output()
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

	cmd1 := exec.Command("mvn", "-Dtest=AppTest#shouldSendResourceAndMetrics", "test")
	cmd1.Dir = path.Join(workDir, "../gw-transit")
	out, err := cmd1.Output()
	log.Println(string(out))
	if err != nil {
		return err
	}

	return nil
}

func clean(headers map[string]string) {
	_, _, err := clients.SendRequest(http.MethodDelete, HostDeleteAPI+TestHostName, headers, nil, nil)
	if err != nil {
		log.Println(err)
	}

	cmd := exec.Command("rm", "-rf", "../gw-transit/src/main/resources/datastore")
	_, err = cmd.Output()
	if err != nil {
		log.Println(err)
	}
}
