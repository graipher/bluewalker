package filter

import (
	"bytes"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

//AdFilter can be used to filter incoming Advertising Reports
//if Filter() returns true, matching ScanReport is created and
//passed to the channel returned by Host.StartScan()
type AdFilter interface {
	Filter(*hci.AdvertisingReport) bool
}

type addressFilter struct {
	addr hci.BtAddress
}

func (f *addressFilter) Filter(rep *hci.AdvertisingReport) bool {
	return rep.Address == f.addr
}

//ByAddress returns AdFilter which filters Advertising Reports by sender
//address. Filter passes report if it comes from given address
func ByAddress(address hci.BtAddress) AdFilter {
	return &addressFilter{addr: address}
}

type vendorFilter struct {
	preamble []byte
}

func (v *vendorFilter) Filter(report *hci.AdvertisingReport) bool {
	ret := false
	for _, data := range report.Data {
		if data.Typ == hci.AdManufacturerSpecific {
			if len(data.Data) < len(v.preamble) {
				continue
			}
			return bytes.Equal(data.Data[:len(v.preamble)], v.preamble)
		}
	}
	return ret
}

//ByVendor returns AdFilter which can be used to filter Advertising Reports
//based on the start of its vendor specific Advertising Data. Filter
//passes Advertising Reports which have vendor -specific advertising data and
//the first bytes of the vendor specific data match the given preamble.
func ByVendor(preamble []byte) AdFilter {
	return &vendorFilter{preamble: preamble}
}

type adTypeFilter struct {
	typ hci.AdType
}

func (f *adTypeFilter) Filter(report *hci.AdvertisingReport) bool {
	ret := false
	for _, data := range report.Data {
		if data.Typ == f.typ {
			ret = true
			break
		}
	}
	return ret
}

//ByAdType returns filter which can be used to filter Advertising Reports
//based on the Type field in the Ad Structures contained on the Advertising Reports
func ByAdType(typ hci.AdType) AdFilter {
	return &adTypeFilter{typ: typ}
}
