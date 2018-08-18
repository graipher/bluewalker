package filter

import (
	"encoding/hex"
	"testing"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

func buildAdvertisingReport(addrString string, data []*hci.AdStructure) *hci.AdvertisingReport {
	addr, _ := hci.BtAddressFromString(addrString)
	return &hci.AdvertisingReport{
		Address:   addr,
		EventType: hci.ScanRsp,
		Rssi:      -70,
		Data:      data,
	}
}

func TestAddressFilter(t *testing.T) {
	addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
	filt := ByAddress(addr)

	rep := buildAdvertisingReport("11:22:33:44:55:66", nil)
	if !filt.Filter(rep) {
		t.Errorf("Expected filter to match")
	}

	rep = buildAdvertisingReport("00:11:22:33:44:55", nil)
	if filt.Filter(rep) {
		t.Errorf("Filter should not have matched")
	}
}

func TestVendorData(t *testing.T) {

	buf, _ := hex.DecodeString("0a0b0c")
	filt := ByVendor(buf)

	vdata, _ := hex.DecodeString("0a0b0c0d0e0f")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdManufacturerSpecific
	structs[0].Data = vdata

	rep := buildAdvertisingReport("11:22:33:44:55", structs)

	if !filt.Filter(rep) {
		t.Errorf("Expected vendor specific filter to match")
	}
}
func TestVendorDataShort(t *testing.T) {

	buf, _ := hex.DecodeString("0a0b0c")
	filt := ByVendor(buf)

	vdata, _ := hex.DecodeString("0a0b")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdManufacturerSpecific
	structs[0].Data = vdata

	rep := buildAdvertisingReport("11:22:33:44:55", structs)

	if filt.Filter(rep) {
		t.Errorf("Did not expect the filter to match")
	}
}
func TestVendorDataNoVendor(t *testing.T) {

	buf, _ := hex.DecodeString("0a0b0c")
	filt := ByVendor(buf)

	vdata, _ := hex.DecodeString("0a0b0c0d0e0f")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdCompleteLocalName
	structs[0].Data = vdata

	rep := buildAdvertisingReport("11:22:33:44:55", structs)

	if filt.Filter(rep) {
		t.Errorf("Expected vendor specific filter to match")
	}
}
