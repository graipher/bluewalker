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
	AdFlags                  AdType = 0x01
	AdMore16BitService       AdType = 0x02
	AdComplete16BitService   AdType = 0x03
	AdMore32BitService       AdType = 0x04
	AdComplete32BitService   AdType = 0x05
	AdMore128BitService      AdType = 0x06
	AdComplete128BitService  AdType = 0x07
	AdShortenedLocalName     AdType = 0x08
	AdCompleteLocalName      AdType = 0x09
	AdTxPower                AdType = 0x0a
	AdClassOfdevice          AdType = 0x0d
	AdPairingHash            AdType = 0x0e
	AdPairingRandomizer      AdType = 0x0f
	AdSmTk                   AdType = 0x10
	AdSmOobFlags             AdType = 0x11
	AdSlaveConnInterval      AdType = 0x12
	Ad16bitServiceSol        AdType = 0x14
	Ad128bitServiceSol       AdType = 0x15
	AdServiceData            AdType = 0x16
	AdPublicTargetAddr       AdType = 0x17
	AdRandomTargetAddr       AdType = 0x18
	AdAppearance             AdType = 0x19
	AdAdvInterval            AdType = 0x1a
	AdDeviceAddress          AdType = 0x1b
	AdLeRole                 AdType = 0x1c
	AdPairingHash256         AdType = 0x1d
	AdPairingRandomizer256   AdType = 0x1e
	Ad32BitServiceSol        AdType = 0x1f
	AdServiceData32          AdType = 0x20
	AdServiceData128         AdType = 0x21
	AdSecureConnConfirm      AdType = 0x22
	AdSecureConnRandom       AdType = 0x23
	AdURI                    AdType = 0x24
	AdIndoorPosit            AdType = 0x25
	AdTransportDiscoveryData AdType = 0x26
	AdLeSupportedFeatures    AdType = 0x27
	AdChannelMapUpdate       AdType = 0x28
	AdMeshPbAdv              AdType = 0x29
	AdMeshMessage            AdType = 0x2a
	AdMeshBeacon             AdType = 0x2b
	Ad3dData                 AdType = 0x3d
	AdManufacturerSpecific   AdType = 0xff
)

// AD Flags bitmap values.
// Specified in Supplement to the Bluetooth Core Specification (CSS version 7), Part A, ch 1.3.1
const (
	// LE Limited Discoverable Mode
	AdFlagLimitedDisc = 0x01
	// LE General Discoverable Mode
	AdFlagGeneralDisc = (0x01 << 1)
	// BR/EDR Not Supported.
	AdFlagNoBrEdr = (0x01 << 2)
	// Simultaneous LE and BR/EDR to Same Device Capable (Controller)
	AdFlagLeBrEdrController = (0x01 << 3)
	// Simultaneous LE and BR/EDR to Same Device Capable (Host)
	AdFlagLeBrEdrHost = (0x01 << 4)
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
		return "Shortened Local name"
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
	case AdPairingHash:
		return "Simple Pairing Hash"
	case AdPairingRandomizer:
		return "Simple Pairing Randomizer"
	case AdSmTk:
		return "Security Manager TK Value"
	case AdSmOobFlags:
		return "Security Manager OOB Flags"
	case AdSlaveConnInterval:
		return "Slave Connection Interval Range"
	case Ad16bitServiceSol:
		return "List of 16-bit Service Solicitation UUIDs"
	case Ad128bitServiceSol:
		return "List of 128-bit Service Solicitation UUIDs"
	case AdServiceData:
		return "Service Data"
	case AdPublicTargetAddr:
		return "Public Target Address"
	case AdRandomTargetAddr:
		return "Random Target Address"
	case AdAdvInterval:
		return "Advertising interval"
	case AdLeRole:
		return "LE Role"
	case AdPairingHash256:
		return "Simple Pairing Hash C-256"
	case AdPairingRandomizer256:
		return "Simple Pairing Randomizer R-256"
	case Ad32BitServiceSol:
		return "List of 32-bit Service Solicitation UUIDs"
	case AdServiceData32:
		return "Service Data - 32-bit UUID"
	case AdServiceData128:
		return "Service Data - 128-bit UUID"
	case AdSecureConnConfirm:
		return "LE Secure Connections Confirmation Value"
	case AdSecureConnRandom:
		return "LE Secure Connections Random Value"
	case AdURI:
		return "URI"
	case AdIndoorPosit:
		return "Indoor Positioning"
	case AdTransportDiscoveryData:
		return "Transport Discovery Data"
	case AdLeSupportedFeatures:
		return "LE Supported Features"
	case AdChannelMapUpdate:
		return "Channel Map Update Indication"
	case AdMeshPbAdv:
		return "PB-ADV"
	case AdMeshMessage:
		return "Mesh Message"
	case AdMeshBeacon:
		return "Mesh Beacon"
	case Ad3dData:
		return "3D Data"
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
	return fmt.Sprintf("%s : 0x%x", ad.Typ.String(), ad.Data)
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
			ret[i].Address.Atype = LeRandomAddress
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
