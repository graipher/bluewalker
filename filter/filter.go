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

type irkFilter struct {
	irk []byte
	// if non-nil, latest resolved address, match first
	resolved *hci.BtAddress
}

func (f *irkFilter) Filter(report *hci.AdvertisingReport) bool {
	// Resolve will check if the address is of right type
	addr := report.Address
	ret := false
	if f.resolved != nil && *f.resolved == addr {
		ret = true
	} else {
		if addr.Resolve(f.irk) {
			f.resolved = &addr
			ret = true
		}
	}
	return ret
}

//ByIrk returns filter which returns true if advertiser address is resolvable using given irk
func ByIrk(irk []byte) AdFilter {
	return &irkFilter{irk: irk}
}
