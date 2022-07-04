package main

import (
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/RIMEDO-Labs/xapp-demo/pkg/manager"
)

var log = logging.GetLogger("xapp-demo")

func main() {
	log.SetLevel(logging.DebugLevel)

	ready := make(chan bool)

	log.Info("Starting demo xAPP")

	config := manager.Config{
		AppID:              "xapp-demo",
		E2tAddress:         "onos-e2t",
		E2tPort:            5150,
		TopoAddress:        "onos-topo",
		TopoPort:           5150,
		SMName:             "oran-e2sm-mho",
		SMVersion:          "v2",
	}

	mgr := manager.NewManager(config)
	mgr.Run()

	<-ready
}
