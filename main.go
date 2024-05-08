package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors/apm"
)

const (
	CmdAPM     = "apm"
	CmdChecker = "checker"
	CmdElastic = "elastic"
	CmdEvents  = "events"
	CmdK8S     = "k8s"
	CmdK8Sa    = "kubernetes"
	CmdNSCA    = "nsca"
	CmdOffice  = "office"
	CmdServer  = "server"
	CmdSNMP    = "snmp"
)

var cmdRe = regexp.MustCompile(`^(tcg[_-])?(?P<cmdName>.+?)([_-]connector)?$`)

func matchCmd(cmd, str string) bool {
	if match := cmdRe.FindStringSubmatch(str); match != nil {
		cmdName := match[cmdRe.SubexpIndex("cmdName")]
		return strings.EqualFold(cmd, cmdName)
	}
	return false
}

func main() {
	args0bs := filepath.Base(os.Args[0])
	appName := config.GetConfig().Connector.AppName

	switch {
	case matchCmd(CmdAPM, appName) ||
		matchCmd(CmdAPM, args0bs):
		apm.Main()

	default:
		panic("main: unknown command:" +
			" args0bs=" + args0bs + " appName=" + appName)
	}
}
