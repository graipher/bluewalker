package host

import (
	"github.com/graipher/bluewalker/filter"
	"github.com/graipher/bluewalker/hci"
	"github.com/graipher/bluewalker/logging"
)

// Parse Advertising Report Data.
// The buffer should contain data from LE Advertising Report LE Meta HCI event
func handleAdvertisingReport(ch chan *ScanReport, filter filter.AdFilter, data []byte) error {

	reports, err := hci.DecodeAdvertisingReport(data)
	if err != nil {
		return err
	}
	for _, rep := range reports {
		if filter != nil && !filter.Filter(rep) {
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
			logging.Warning.Printf("Dropping AD report due channel being full!")
		}
	}
	return nil
}
