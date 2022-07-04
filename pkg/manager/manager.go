package manager

import (
	"context"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/RIMEDO-Labs/xapp-demo/pkg/demo"
	"github.com/RIMEDO-Labs/xapp-demo/pkg/southbound/e2"
)

var log = logging.GetLogger("xapp-demo", "manager")

type Config struct {
	AppID       string
	E2tAddress  string
	E2tPort     int
	TopoAddress string
	TopoPort    int
	SMName      string
	SMVersion   string
}

func NewManager(config Config) *Manager {

	ueStore := store.NewStore()
	cellStore := store.NewStore()

	indCh := make(chan *demo.E2NodeIndication)

	options := e2.Options{
		AppID:       config.AppID,
		E2tAddress:  config.E2tAddress,
		E2tPort:     config.E2tPort,
		TopoAddress: config.TopoAddress,
		TopoPort:    config.TopoPort,
		SMName:      config.SMName,
		SMVersion:   config.SMVersion,
	}

	e2Manager, err := e2.NewManager(options, indCh)
	if (err != nil) {
		log.Error(err)
	}

	demoController := demo.NewController(indCh, ueStore, cellStore)

	manager := &Manager{
		e2Manager:   e2Manager,
		demoCtrl: demoController,
	}
	return manager
}

type Manager struct {
	e2Manager   e2.Manager
	demoCtrl    *demo.Controller
}

func (m *Manager) Run() {
	if err := m.start(); err != nil {
		log.Fatal("Unable to run Manager", err)
	}
}

func (m *Manager) start() error {

	m.e2Manager.Start()

	go m.demoCtrl.Run(context.Background())

	return nil
}

