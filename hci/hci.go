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
