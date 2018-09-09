package host

import (
	"log"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
)

type adfilters []filter.AdFilter

func filterList() adfilters {
	return adfilters(make([]filter.AdFilter, 0))
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

func addFilter(filters adfilters, filt filter.AdFilter) adfilters {
	return append(filters, filt)
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
		scanReport.Type = rep.EventType

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
