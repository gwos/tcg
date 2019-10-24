package integration

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestIntegration(t *testing.T) {
	cmd1 := exec.Command("curl",
		"-H", "Accept: application/json", "-H", "Content-Type: application/json",
		"-H", "GWOS-APP-NAME: GW8", "-H", "GWOS-API-TOKEN: 0db2a4fd-a095-4392-8dbc-46bdd9eab02d",
		"--request", "POST", "--data", "{\"nameFilter\":\"MY_TESTq_HOS\",\"n\":5}",
		"http://localhost/api/statusViewer/search")
	out, _ := cmd1.Output()

	fmt.Println(string(out))
	//
	//workDir, err := os.Getwd()
	//if err != nil {
	//	log.Println(err)
	//}
	//
	//cmd2 := exec.Command("mvn", "-Dtest=AppTest#shouldSynchronizeInventory", "test")
	//cmd2.Dir = path.Join(workDir, "../gw-transit")
	//out, _ = cmd2.Output()
	//fmt.Println(string(out))
	//
	//cmd3 := exec.Command("curl",
	//	"-H", "Accept: application/json",
	//	"-H", "Content-Type: application/json",
	//	"-H", "GWOS-APP-NAME: GW8",
	//	"-H", "GWOS-API-TOKEN: 0db2a4fd-a095-4392-8dbc-46bdd9eab02d",
	//	"--request", "POST",
	//	"--data", "{\"nameFilter\":\"MY_TESTq_HOST\",\"n\":5}",
	//	"http://localhost/api/statusViewer/search")
	//out, _ = cmd3.Output()
	//fmt.Println(string(out))
}
