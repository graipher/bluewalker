package hci

import (
	"bytes"
	"fmt"
	"log"
)

// Transport allows sending and receiving raw HCI packets
// Use hci.Raw() to create transport
type Transport interface {
	// Close closes the transport
	Close()
	Read() ([]byte, error)
	Write(buffer []byte) error
}

const (
	hciCommandPacket byte = 0x01
	hciACLPacket     byte = 0x02
	// HciEventPacket indicates that data from transport contains HCI Event
	HciEventPacket byte = 0x04
)

// AdvType defines the Advertising Event Type
// See Bluetooth 5.0, vol 2, part E, ch 7.7.65.2
type AdvType byte

// Advertising type values
const (
	AdvInd        AdvType = 0x00
	AdvDirectInd  AdvType = 0x01
	AdvScanInd    AdvType = 0x02
	AdvNonconnInd AdvType = 0x03
	ScanRsp       AdvType = 0x04
)

func (adv AdvType) String() string {
	switch adv {
	case AdvInd:
		return "Connectable undirected"
	case AdvDirectInd:
		return "Connectable directed"
	case AdvScanInd:
		return "Scannable undirected"
	case AdvNonconnInd:
		return "Non connectable undirected"
	case ScanRsp:
		return "Scan Response"
	default:
		return fmt.Sprintf("Unknown (%.2x)", int(adv))
	}
}

//AdvertisingReport represents data parsed from LE Advertising Report
//Event received from controller
// See Bluetooth 5.0, vol 2, part E, ch 7.7.65.2
type AdvertisingReport struct {
	EventType AdvType
	Address   BtAddress
	Data      []*AdStructure
	Rssi      int8
}

func (r *AdvertisingReport) String() string {
	return fmt.Sprintf("%s from %s (%s) with %d bytes of data, RSSI %d", r.EventType.String(), r.Address.String(), r.Address.Atype.String(), len(r.Data), r.Rssi)
}

// AdType is the type for advertising data
// See Bluetooth 5.0, vol 3, part C, ch 11
type AdType byte

// AD type values
// See https://www.bluetooth.com/specifications/assigned-numbers/generic-access-profile
const (
	AdFlags                 AdType = 0x01
	AdMore16BitService      AdType = 0x02
	AdComplete16BitService  AdType = 0x03
	AdMore32BitService      AdType = 0x04
	AdComplete32BitService  AdType = 0x05
	AdMore128BitService     AdType = 0x06
	AdComplete128BitService AdType = 0x07
	AdShortenedLocalName    AdType = 0x08
	AdCompleteLocalName     AdType = 0x09
	AdTxPower               AdType = 0x0a
	AdClassOfdevice         AdType = 0x0d
	AdDeviceAddress         AdType = 0x1b
	AdAppearance            AdType = 0x19
	AdManufacturerSpecific  AdType = 0xff
)

func (ad AdType) String() string {
	switch ad {
	case AdFlags:
		return "Flags"
	case AdMore16BitService:
		return "16 Bit Service Class UUID"
	case AdComplete16BitService:
		return "Complete 16 Bit Service Class UUID"
	case AdMore32BitService:
		return "32 Bit Service Class UUID"
	case AdComplete32BitService:
		return "Complete 32 Bit Service Class UUID"
	case AdMore128BitService:
		return "128 Bit Service Class UUID"
	case AdComplete128BitService:
		return "Complete 128 Bit Service Class UUID"
	case AdShortenedLocalName:
		return "Local name"
	case AdCompleteLocalName:
		return "Complete local name"
	case AdTxPower:
		return "Tx Power"
	case AdClassOfdevice:
		return "Class of device"
	case AdManufacturerSpecific:
		return "Manufacturer Specific"
	case AdDeviceAddress:
		return "LE Bluetooth Device Address"
	case AdAppearance:
		return "Appearance"
	default:
		return fmt.Sprintf("Unknown (%.2x)", int(ad))
	}
}

// AdStructure defines advertising data
// See Bluetooth 5.0, vol 3, part C, ch 11
type AdStructure struct {
	Typ  AdType
	Data []byte
}

func (ad *AdStructure) String() string {
	return fmt.Sprintf("%s : %x", ad.Typ.String(), ad.Data)
}

func decodeAdStructure(buf []byte) (*AdStructure, error) {
	length := int(buf[0])
	// sometimes zero -length AD structres are used for padding
	if length == 0 {
		return nil, nil
	}
	if length+1 > len(buf) {
		return nil, fmt.Errorf("Invalid length for AD Structure")
	}
	t := AdType(buf[1])
	dat := buf[2 : 2+length-1]
	return &AdStructure{Typ: t, Data: dat}, nil
}

// ParseAdData parses the advertising data to ad structres
func parseAdData(buf []byte) ([]*AdStructure, error) {

	offset := 0
	structures := make([]*AdStructure, 0)
	for offset < len(buf) {
		ad, err := decodeAdStructure(buf[offset:])
		if err != nil {
			return nil, err
		}
		if ad == nil {
			// in theory, there could be another structure after
			// 0 -length block.
			offset++
			continue
		}
		structures = append(structures, ad)
		offset += (len(ad.Data) + 2)
	}
	return structures, nil
}

//DecodeAdvertisingReport can be used to decode data received in
//Advertising Report Event. Returns all reports contained in event.
func DecodeAdvertisingReport(buf []byte) ([]*AdvertisingReport, error) {

	eMalformed := fmt.Errorf("Malformed data for advertising report")
	rd := bytes.NewReader(buf)
	b, err := rd.ReadByte()
	if err != nil {
		return nil, eMalformed
	}
	numReports := int(b)

	ret := make([]*AdvertisingReport, numReports)
	for i := 0; i < numReports; i++ {
		ret[i] = new(AdvertisingReport)
		b, err := rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		ret[i].EventType = AdvType(b)
		// Read the address, first byte is address type
		b, err = rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		addrBytes := make([]byte, 6)
		n, err := rd.Read(addrBytes)
		if n != len(addrBytes) || err != nil {
			return nil, eMalformed
		}
		ret[i].Address = ToBtAddress(addrBytes)
		if b == 0 {
			ret[i].Address.Atype = LePublicAddress
		} else {
			ret[i].Address.Atype = LePrivateAddress
		}
		// Read the AD Structure data, length first
		b, err = rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		if b > 0 {
			advData := make([]byte, int(b))
			n, err = rd.Read(advData)
			if n != len(advData) || err != nil {
				return nil, eMalformed
			}
			ret[i].Data, err = parseAdData(advData)
			if err != nil {
				return nil, fmt.Errorf("Malformed data in AD Structures: %s", err.Error())
			}
		}
		b, err = rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		ret[i].Rssi = int8(b)
		log.Printf("Advertising report: %s", ret[i].String())
	}
	return ret, nil
}
