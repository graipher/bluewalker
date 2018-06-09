package host

import (
	"fmt"
	"log"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

type advertisingReport struct {
	typ  hci.AdvType
	from hci.BtAddress
	rssi int
	data []byte
}

func (a *advertisingReport) String() string {

	return fmt.Sprintf("%s from %s (%s) with %d bytes of data", a.typ.String(), a.from.String(), a.from.Atype.String(), len(a.data))

}

// Parse Advertising Report Data.
// The buffer should contain data from LE Advertising Report LE Meta HCI event
// see Bluetooth v5.0, vol 2, part E, ch 7.7.65.2
func parseAdvertisingReport(data []byte) error {

	records := data[0]
	offset := 1
	reports := make([]*advertisingReport, records)
	for i := 0; i < int(records); i++ {
		if offset+10 > len(data) {
			return fmt.Errorf("Malformed Advertising report")
		}
		reports[i] = new(advertisingReport)
		reports[i].typ = hci.AdvType(data[offset])
		offset++
		addrType := hci.LePublicAddress
		if data[offset] != 0x00 {
			addrType = hci.LePrivateAddress // FIXME: Not really
		}
		offset++
		reports[i].from = hci.ToBtAddress(data[offset : offset+6])
		reports[i].from.Atype = addrType
		offset += 6
		dataLen := int(data[offset])
		offset++
		if dataLen > 0 {
			if offset+dataLen > len(data) {
				return fmt.Errorf("Invalid data length for Advertising report")
			}
			reports[i].data = data[offset : offset+dataLen]
		}
		offset += dataLen
		reports[i].rssi = int(data[offset])
		offset++
	}

	for _, rep := range reports {
		log.Printf("Report: %s", rep.String())
		ads, err := hci.ParseAdData(rep.data)
		if err != nil {
			log.Printf("Invalid AD Data: %s", err.Error())
		} else {
			log.Printf("AD Data:")
			for _, ad := range ads {
				log.Printf("|%s", ad.String())
			}
		}
	}
	return nil
}
