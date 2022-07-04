package demo

import (
	"fmt"
	"strconv"

	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	e2sm_v2_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-v2-ies"
)

type UeData struct {
	UeID          string
	E2NodeID      string
	CGI           *e2sm_v2_ies.Cgi
	CGIString     string
	RrcState      string
	FiveQi        int64
	RsrpServing   int32
	RsrpNeighbors map[string]int32
	RsrpTable     map[string]int32
	CgiTable      map[string]*e2sm_v2_ies.Cgi
	Idle          bool
}

type CellData struct {
	CGI                    *e2sm_v2_ies.Cgi
	CGIString              string
	CumulativeHandoversIn  int
	CumulativeHandoversOut int
	Ues                    map[string]*UeData
}


func PlmnIDBytesToInt(b []byte) uint64 {
	return uint64(b[2])<<16 | uint64(b[1])<<8 | uint64(b[0])
}

func PlmnIDNciToCGI(plmnID uint64, nci uint64) string {
	cgi := strconv.FormatInt(int64(plmnID<<36|(nci&0xfffffffff)), 16)
	return cgi
}

func GetNciFromCellGlobalID(cellGlobalID *e2sm_v2_ies.Cgi) uint64 {
	return BitStringToUint64(cellGlobalID.GetNRCgi().GetNRcellIdentity().GetValue().GetValue(), int(cellGlobalID.GetNRCgi().GetNRcellIdentity().GetValue().GetLen()))
}

func GetPlmnIDBytesFromCellGlobalID(cellGlobalID *e2sm_v2_ies.Cgi) []byte {
	return cellGlobalID.GetNRCgi().GetPLmnidentity().GetValue()
}

func GetMccMncFromPlmnID(plmnId uint64) (string, string) {
	plmnIdString := strconv.FormatUint(plmnId, 16)
	return plmnIdString[0:3], plmnIdString[3:]
}

func GetPlmnIdFromMccMnc(mcc string, mnc string) (uint64, error) {
	combined := mcc + mnc
	plmnId, err := strconv.ParseUint(combined, 16, 64)
	if err != nil {
		log.Warn("Cannot convert PLMN ID string into uint64 type!")
	}
	return plmnId, err
}

func GetCGIFromIndicationHeader(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1) string {
	nci := GetNciFromCellGlobalID(header.GetCgi())
	plmnIDBytes := GetPlmnIDBytesFromCellGlobalID(header.GetCgi())
	plmnID := PlmnIDBytesToInt(plmnIDBytes)
	return PlmnIDNciToCGI(plmnID, nci)
}

func GetCGIFromMeasReportItem(measReport *e2sm_mho.E2SmMhoMeasurementReportItem) string {
	nci := GetNciFromCellGlobalID(measReport.GetCgi())
	plmnIDBytes := GetPlmnIDBytesFromCellGlobalID(measReport.GetCgi())
	plmnID := PlmnIDBytesToInt(plmnIDBytes)
	return PlmnIDNciToCGI(plmnID, nci)
}

func BitStringToUint64(bitString []byte, bitCount int) uint64 {
	var result uint64
	for i, b := range bitString {
		result += uint64(b) << ((len(bitString) - i - 1) * 8)
	}
	if bitCount%8 != 0 {
		return result >> (8 - bitCount%8)
	}
	return result
}

func GetUeID(ueID *e2sm_v2_ies.Ueid) (int64, error) {

	switch ue := ueID.Ueid.(type) {
	case *e2sm_v2_ies.Ueid_GNbUeid:
		return ue.GNbUeid.GetAmfUeNgapId().GetValue(), nil
	case *e2sm_v2_ies.Ueid_ENbUeid:
		return ue.ENbUeid.GetMMeUeS1ApId().GetValue(), nil
	case *e2sm_v2_ies.Ueid_EnGNbUeid:
		return int64(ue.EnGNbUeid.GetMENbUeX2ApId().GetValue()), nil
	case *e2sm_v2_ies.Ueid_NgENbUeid:
		return ue.NgENbUeid.GetAmfUeNgapId().GetValue(), nil
	default:
		return -1, fmt.Errorf("GetUeID() couldn't extract UeID - obtained unexpected type %v", ue)
	}
}

