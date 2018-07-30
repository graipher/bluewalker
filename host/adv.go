package host

import (
	"log"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

//AdFilter can be used to filter incoming Advertising Reports
//if Filter() returns true, matching ScanReport is created and
//passed to the channel returned by Host.StartScan()
type AdFilter interface {
	Filter(*hci.AdvertisingReport) bool
}

type adfilters []AdFilter

func filterList() adfilters {
	return adfilters(make([]AdFilter, 0))
}

func (f adfilters) filter(report *hci.AdvertisingReport) bool {
	pass := true
	if len(f) != 0 {
		pass = false
		for _, filt := range f {
			if filt.Filter(report) {
				pass = true
				break
			}
		}
	}
	return pass
}

func addFilter(filters adfilters, filt AdFilter) adfilters {
	return append(filters, filt)
}

type addressFilter struct {
	addr hci.BtAddress
}

func (f *addressFilter) Filter(rep *hci.AdvertisingReport) bool {
	return rep.Address == f.addr
}

//AddressFilter returns AdFilter which filters Advertising Reports on sender
//address. Filter passes report if it comes from given address
func AddressFilter(address hci.BtAddress) AdFilter {
	return &addressFilter{addr: address}
}

// Parse Advertising Report Data.
// The buffer should contain data from LE Advertising Report LE Meta HCI event
func handleAdvertisingReport(ch chan *ScanReport, filters adfilters, data []byte) error {

	reports, err := hci.DecodeAdvertisingReport(data)
	if err != nil {
		return err
	}
	for _, rep := range reports {
		if !filters.filter(rep) {
			return nil
		}

		scanReport := new(ScanReport)
		scanReport.Address = rep.Address
		scanReport.Data = rep.Data
		scanReport.Rssi = rep.Rssi

		// we can't block here as we are running on event loop goroutine.
		// hence check if the channel is writable.
		select {
		case ch <- scanReport:
		default:
			log.Printf("Dropping AD report due channel being full!")

		}
	}
	return nil
}
