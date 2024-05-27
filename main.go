package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors/apm"
	"github.com/gwos/tcg/connectors/azure"
	"github.com/gwos/tcg/connectors/checker"
	"github.com/gwos/tcg/connectors/elastic"
	"github.com/gwos/tcg/connectors/events"
	"github.com/gwos/tcg/connectors/k8s"
	"github.com/gwos/tcg/connectors/nsca"
	"github.com/gwos/tcg/connectors/office"
	"github.com/gwos/tcg/connectors/server"
	"github.com/gwos/tcg/connectors/snmp"
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
	CmdAzure   = "azure"
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
	case matchCmd(CmdAPM, appName) || matchCmd(CmdAPM, args0bs):
		apm.Run()
	case matchCmd(CmdChecker, appName) || matchCmd(CmdChecker, args0bs):
		checker.Run()
	case matchCmd(CmdElastic, appName) || matchCmd(CmdElastic, args0bs):
		elastic.Run()
	case matchCmd(CmdK8S, appName) || matchCmd(CmdK8S, args0bs) ||
		matchCmd(CmdK8Sa, appName) || matchCmd(CmdK8Sa, args0bs):
		k8s.Run()
	case matchCmd(CmdNSCA, appName) || matchCmd(CmdNSCA, args0bs):
		nsca.Run()
	case matchCmd(CmdOffice, appName) || matchCmd(CmdOffice, args0bs):
		office.Run()
	case matchCmd(CmdServer, appName) || matchCmd(CmdServer, args0bs):
		server.Run()
	case matchCmd(CmdSNMP, appName) || matchCmd(CmdSNMP, args0bs):
		snmp.Run()
	case matchCmd(CmdEvents, appName) || matchCmd(CmdEvents, args0bs):
		events.Run()
	case matchCmd(CmdAzure, appName) || matchCmd(CmdAzure, args0bs):
		azure.Run()
	default:
		panic("main: unknown command:" +
			" args0bs=" + args0bs + " appName=" + appName)
	}
}
