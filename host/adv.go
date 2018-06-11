package host

import (
	"log"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

// Parse Advertising Report Data.
// The buffer should contain data from LE Advertising Report LE Meta HCI event
func handleAdvertisingReport(ch chan *ScanReport, data []byte) error {

	reports, err := hci.DecodeAdvertisingReport(data)
	if err != nil {
		return err
	}
	for _, rep := range reports {
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
