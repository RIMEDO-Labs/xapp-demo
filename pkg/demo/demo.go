package demo

import (
	"time"
	"context"
	"sync"
	
	"strconv"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	e2sm_v2_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-v2-ies"
	"google.golang.org/protobuf/proto"
	"github.com/onosproject/onos-mho/pkg/store"
)

var log = logging.GetLogger("xapp-demo", "demo")

type E2NodeIndication struct {
	NodeID      string
	TriggerType e2sm_mho.MhoTriggerType
	IndMsg      e2api.Indication
}

func NewController(indChan chan *E2NodeIndication, ueStore store.Store, cellStore store.Store) *Controller {

	return &Controller{
		IndChan:         indChan,
		ueStore:         ueStore,
		cellStore:       cellStore,
		cells:           make(map[string]*CellData),
		mu:              sync.RWMutex{},
	}
}

type Controller struct {
	IndChan         chan *E2NodeIndication
	ueStore         store.Store
	cellStore       store.Store
	cells           map[string]*CellData
	mu              sync.RWMutex
}

func (c *Controller) Run(ctx context.Context) {
	go c.listenIndChan(ctx)

	go func() {
		for {
			chEntries := make(chan *store.Entry, 1024)
			err := c.ueStore.Entries(ctx, chEntries)
			if err != nil {
				log.Warn(err)
				
			} else {
				
				for entry := range chEntries {
					ueData := entry.Value.(UeData)
					log.Infof("UE: %v CGI: %v", ueData.UeID, ueData.CGIString)
						for cell, rsrp := range ueData.RsrpTable {
							log.Infof(" - %v: %v", cell, rsrp)
						}
					log.Info("  ")
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

func (c *Controller) listenIndChan(ctx context.Context) {
	var err error
	for indMsg := range c.IndChan {

		indHeaderByte := indMsg.IndMsg.Header
		indMessageByte := indMsg.IndMsg.Payload
		e2NodeID := indMsg.NodeID

		indHeader := e2sm_mho.E2SmMhoIndicationHeader{}
		if err = proto.Unmarshal(indHeaderByte, &indHeader); err == nil {
			indMessage := e2sm_mho.E2SmMhoIndicationMessage{}
			if err = proto.Unmarshal(indMessageByte, &indMessage); err == nil {
				switch x := indMessage.E2SmMhoIndicationMessage.(type) {
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat1:
					if indMsg.TriggerType == e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_UPON_RCV_MEAS_REPORT {
						go c.handleMeasReport(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
					} else if indMsg.TriggerType == e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_PERIODIC {
						go c.handlePeriodicReport(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
					}
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat2:
					//go c.handleRrcState(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat2(), e2NodeID)
				default:
					log.Warnf("Unknown MHO indication message format, indication message: %v", x)
				}
			}
		}
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *Controller) handlePeriodicReport(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ueID, err := GetUeID(message.GetUeId())
	if err != nil {
		log.Errorf("handlePeriodicReport() couldn't extract UeID: %v", err)
	}
	cgi := GetCGIFromIndicationHeader(header)
	cgi = c.ConvertCgiToTheRightForm(cgi)
	cgiObject := header.GetCgi()


	ueIdString := strconv.Itoa(int(ueID))
	n := (16 - len(ueIdString))
	for i := 0; i < n; i++ {
		//ueIdString = "0" + ueIdString
	}
	var ueData *UeData
	//newUe := false
	ueData = c.GetUe(ctx, ueIdString)
	if ueData == nil {
		ueData = c.CreateUe(ctx, ueIdString)
		c.AttachUe(ctx, ueData, cgi, cgiObject)
		
	} else if ueData.CGIString != cgi {
		c.AttachUe(ctx, ueData, cgi, cgiObject)
	
	}

	ueData.E2NodeID = e2NodeID

	rsrpServing, rsrpNeighbors, rsrpTable, cgiTable := c.GetRsrpFromMeasReport(ctx, GetNciFromCellGlobalID(header.GetCgi()), message.MeasReport)
	log.Infof("--- handlePeriodicReport UE: %v CGI: %v RSRP: %v", ueIdString, cgi, rsrpServing)

	plmnIdBytes := GetPlmnIDBytesFromCellGlobalID(cgiObject)
	plmnId := PlmnIDBytesToInt(plmnIdBytes)
	mcc, mnc := GetMccMncFromPlmnID(plmnId)

	log.Infof("mcc: %v; mnc: %v; nci: %v", mcc, mnc, GetNciFromCellGlobalID(header.GetCgi()))
	
	old5qi := ueData.FiveQi
	ueData.FiveQi = c.GetFiveQiFromMeasReport(ctx, GetNciFromCellGlobalID(header.GetCgi()), message.MeasReport)

	if (old5qi != ueData.FiveQi) {
		// log.Infof("\t\tQUALITY MESSAGE: 5QI for UE [ID:%v] changed [5QI:%v]\n", ueData.UeID, ueData.FiveQi)
	}

	ueData.RsrpServing, ueData.RsrpNeighbors, ueData.RsrpTable, ueData.CgiTable = rsrpServing, rsrpNeighbors, rsrpTable, cgiTable
	c.SetUe(ctx, ueData)

}

func (c *Controller) handleMeasReport(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ueID, err := GetUeID(message.GetUeId())
	if err != nil {
		log.Errorf("handleMeasReport() couldn't extract UeID: %v", err)
	}
	cgi := GetCGIFromIndicationHeader(header)
	cgi = c.ConvertCgiToTheRightForm(cgi)
	cgiObject := header.GetCgi()

	ueIdString := strconv.Itoa(int(ueID))
	n := (16 - len(ueIdString))
	for i := 0; i < n; i++ {
		//ueIdString = "0" + ueIdString
	}
	var ueData *UeData
	ueData = c.GetUe(ctx, ueIdString)
	if ueData == nil {
		ueData = c.CreateUe(ctx, ueIdString)
		c.AttachUe(ctx, ueData, cgi, cgiObject)
	} else if ueData.CGIString != cgi {
		c.AttachUe(ctx, ueData, cgi, cgiObject)
	}

	ueData.E2NodeID = e2NodeID

	ueData.RsrpServing, ueData.RsrpNeighbors, ueData.RsrpTable, ueData.CgiTable = c.GetRsrpFromMeasReport(ctx, GetNciFromCellGlobalID(header.GetCgi()), message.MeasReport)
	log.Infof("--- handleMeasReport UE: %v CGI: %v RSRP: %v", ueIdString, cgi, ueData.RsrpServing)

	old5qi := ueData.FiveQi
	ueData.FiveQi = c.GetFiveQiFromMeasReport(ctx, GetNciFromCellGlobalID(header.GetCgi()), message.MeasReport)
	if (old5qi != ueData.FiveQi) {
		//log.Infof("\t\tQUALITY MESSAGE: 5QI for UE [ID:%v] changed [5QI:%v]\n", ueData.UeID, ueData.FiveQi)
	}

	c.SetUe(ctx, ueData)

}


func (c *Controller) CreateUe(ctx context.Context, ueID string) *UeData {
	if len(ueID) == 0 {
		panic("bad data")
	}
	ueData := &UeData{
		UeID:          ueID,
		CGIString:     "",
		RrcState:      e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED)],
		RsrpNeighbors: make(map[string]int32),
		Idle:          false,
	}
	_, err := c.ueStore.Put(ctx, ueID, *ueData)
	if err != nil {
		log.Warn(err)
	}

	return ueData
}

func (c *Controller) GetUe(ctx context.Context, ueID string) *UeData {
	var ueData *UeData
	u, err := c.ueStore.Get(ctx, ueID)
	if err != nil || u == nil {
		return nil
	}
	t := u.Value.(UeData)
	ueData = &t
	if ueData.UeID != ueID {
		panic("bad data")
	}

	return ueData
}

func (c *Controller) SetUe(ctx context.Context, ueData *UeData) {
	_, err := c.ueStore.Put(ctx, ueData.UeID, *ueData)
	if err != nil {
		panic("bad data")
	}
}

func (c *Controller) AttachUe(ctx context.Context, ueData *UeData, cgi string, cgiObject *e2sm_v2_ies.Cgi) {

	c.DetachUe(ctx, ueData)

	ueData.CGIString = cgi
	ueData.CGI = cgiObject
	c.SetUe(ctx, ueData)
	cell := c.GetCell(ctx, cgi)
	if cell == nil {
		cell = c.CreateCell(ctx, cgi, cgiObject)
	}
	cell.Ues[ueData.UeID] = ueData
	c.SetCell(ctx, cell)
}

func (c *Controller) DetachUe(ctx context.Context, ueData *UeData) {
	for _, cell := range c.cells {
		delete(cell.Ues, ueData.UeID)
	}
}

func (c *Controller) CreateCell(ctx context.Context, cgi string, cgiObject *e2sm_v2_ies.Cgi) *CellData {
	if len(cgi) == 0 {
		panic("bad data")
	}
	cellData := &CellData{
		CGI:       cgiObject,
		CGIString: cgi,
		Ues:       make(map[string]*UeData),
	}
	_, err := c.cellStore.Put(ctx, cgi, *cellData)
	if err != nil {
		panic("bad data")
	}
	c.cells[cellData.CGIString] = cellData
	return cellData
}

func (c *Controller) GetCell(ctx context.Context, cgi string) *CellData {
	var cellData *CellData
	cell, err := c.cellStore.Get(ctx, cgi)
	if err != nil || cell == nil {
		return nil
	}
	t := cell.Value.(CellData)
	if t.CGIString != cgi {
		panic("bad data")
	}
	cellData = &t
	return cellData
}

func (c *Controller) SetCell(ctx context.Context, cellData *CellData) {
	if len(cellData.CGIString) == 0 {
		panic("bad data")
	}
	_, err := c.cellStore.Put(ctx, cellData.CGIString, *cellData)
	if err != nil {
		panic("bad data")
	}
	c.cells[cellData.CGIString] = cellData
}

func (c *Controller) GetFiveQiFromMeasReport(ctx context.Context, servingNci uint64, measReport []*e2sm_mho.E2SmMhoMeasurementReportItem) int64 {
	var fiveQiServing int64

	for _, measReportItem := range measReport {

		if GetNciFromCellGlobalID(measReportItem.GetCgi()) == servingNci {
			fiveQi := measReportItem.GetFiveQi()
			if fiveQi != nil {
				fiveQiServing = int64(fiveQi.GetValue())
				if fiveQiServing > 127 {
					fiveQiServing = 2
				} else {
					fiveQiServing = 1
				}
			} else {
				fiveQiServing = -1
			}
		}
	}

	return fiveQiServing
}

func (c *Controller) GetRsrpFromMeasReport(ctx context.Context, servingNci uint64, measReport []*e2sm_mho.E2SmMhoMeasurementReportItem) (int32, map[string]int32, map[string]int32, map[string]*e2sm_v2_ies.Cgi) {
	var rsrpServing int32
	rsrpNeighbors := make(map[string]int32)
	rsrpTable := make(map[string]int32)
	cgiTable := make(map[string]*e2sm_v2_ies.Cgi)

	for _, measReportItem := range measReport {

		if GetNciFromCellGlobalID(measReportItem.GetCgi()) == servingNci {
			CGIString := GetCGIFromMeasReportItem(measReportItem)
			CGIString = c.ConvertCgiToTheRightForm(CGIString)
			rsrpServing = measReportItem.GetRsrp().GetValue()
			rsrpTable[CGIString] = measReportItem.GetRsrp().GetValue()
			cgiTable[CGIString] = measReportItem.GetCgi()
		} else {
			CGIString := GetCGIFromMeasReportItem(measReportItem)
			CGIString = c.ConvertCgiToTheRightForm(CGIString)
			rsrpNeighbors[CGIString] = measReportItem.GetRsrp().GetValue()
			rsrpTable[CGIString] = measReportItem.GetRsrp().GetValue()
			cgiTable[CGIString] = measReportItem.GetCgi()
			cell := c.GetCell(ctx, CGIString)
			if cell == nil {
				_ = c.CreateCell(ctx, CGIString, measReportItem.GetCgi())
			}
		}
	}

	return rsrpServing, rsrpNeighbors, rsrpTable, cgiTable
}

func (c *Controller) ConvertCgiToTheRightForm(cgi string) string {
	
	return cgi[0:6] + cgi[14:15] + cgi[12:14] + cgi[10:12] + cgi[8:10] + cgi[6:8]
	
	return cgi
}