package hci

import "fmt"

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

//BtAddressType defines address type for bluetooth address
type BtAddressType byte

// BtAddress defines the Bluetooth address
type BtAddress struct {
	raw   [6]byte
	Atype BtAddressType
}

func (ba BtAddress) String() string {
	return fmt.Sprintf("%.2x:%.2x:%.2x:%.2x:%.2x:%.2x", ba.raw[5], ba.raw[4], ba.raw[3], ba.raw[2], ba.raw[1], ba.raw[0])
}

// Contstants for Bluetooth address type
const (
	LePublicAddress  BtAddressType = 0x00
	LePrivateAddress BtAddressType = 0x01
	BrEdrAddress     BtAddressType = 0x02
)

func (t BtAddressType) String() string {
	switch t {
	case LePublicAddress:
		return "LE Public"
	case LePrivateAddress:
		return "LE Private"
	case BrEdrAddress:
		return "BR/EDR"
	default:
		return "unknown"
	}
}

// ToBtAddress returns BtAddress with data from given slice
// the bytes are copied from the slice
func ToBtAddress(data []byte) BtAddress {

	var addr BtAddress
	for i := 0; i < 6; i++ {
		addr.raw[i] = data[i]
	}
	return addr
}

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

// AdType is the type for advertising data
// See Bluetooth 5.0, vol 3, part C, ch 11
type AdType byte

// AD type values
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
	AdManufacturerSpecific  AdType = 0xff
)

func (ad AdType) String() string {
	switch ad {
	case AdFlags:
		return "Flags"
	case AdMore16BitService:
		return "16 Bit Service UUID"
	case AdComplete16BitService:
		return "Complete 16 Service UUID"
	case AdMore32BitService:
		return "32 Bit Service UUID"
	case AdComplete32BitService:
		return "Complete 32 Service UUID"
	case AdMore128BitService:
		return "128 Bit Service UUID"
	case AdComplete128BitService:
		return "Complete 128 Service UUID"
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
func ParseAdData(buf []byte) ([]*AdStructure, error) {

	offset := 0
	structures := make([]*AdStructure, 0)
	for offset < len(buf) {
		ad, err := decodeAdStructure(buf[offset:])
		if ad == nil {
			// in theory, there could be another structure after
			// 0 -length block.
			offset++
			continue
		}
		if err != nil {
			return nil, err
		}
		structures = append(structures, ad)
		offset += (len(ad.Data) + 2)
	}
	return structures, nil
}
