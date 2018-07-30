package host

import (
	"testing"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

func TestFiltering(t *testing.T) {

	addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
	filt := AddressFilter(addr)

	list := filterList()
	list = addFilter(list, filt)

	report := &hci.AdvertisingReport{
		EventType: hci.ScanRsp,
		Address:   addr,
		Data:      nil,
		Rssi:      -80,
	}

	if !list.filter(report) {
		t.Errorf("Expected filter to match")
	}

	addr2, _ := hci.BtAddressFromString("22:33:44:55:66:77")
	report = &hci.AdvertisingReport{
		EventType: hci.ScanRsp,
		Address:   addr2,
		Data:      nil,
		Rssi:      -80,
	}

	if list.filter(report) {
		t.Errorf("Did not expect filter to match")
	}
}

func TestFilteringMultiple(t *testing.T) {

	addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
	addr2, _ := hci.BtAddressFromString("00:11:22:33:44:55")
	filt := AddressFilter(addr)
	filt2 := AddressFilter(addr2)

	list := filterList()
	list = addFilter(list, filt)
	list = addFilter(list, filt2)

	report := &hci.AdvertisingReport{
		EventType: hci.ScanRsp,
		Address:   addr2,
		Data:      nil,
		Rssi:      -80,
	}

	if !list.filter(report) {
		t.Errorf("Expected filter to match")
	}

	addr3, _ := hci.BtAddressFromString("aa:bb:cc:dd:ee:ff")

	report = &hci.AdvertisingReport{
		EventType: hci.ScanRsp,
		Address:   addr3,
		Data:      nil,
		Rssi:      -80,
	}

	if list.filter(report) {
		t.Errorf("Did not expect filter to match")
	}

}

func TestEmpyFilterList(t *testing.T) {
	list := filterList()
	addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
	report := &hci.AdvertisingReport{
		EventType: hci.ScanRsp,
		Address:   addr,
		Data:      nil,
		Rssi:      -80,
	}

	if !list.filter(report) {
		t.Errorf("Expected empty list to match all")
	}

}
