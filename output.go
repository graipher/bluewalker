package main

import (
	"encoding/binary"
	"fmt"
	"strings"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

func formatAddress(addr hci.BtAddress) string {
	addrstr := addr.String()
	if addr.Atype == hci.LeRandomAddress {
		addrstr += ",random ("
		if addr.IsNonResolvable() {
			addrstr += "non-resolvable private"
		} else if addr.IsResolvable() {
			addrstr += "resolvable private"
		} else if addr.IsStatic() {
			addrstr += "static"
		} else {
			addrstr += "??"
		}
		addrstr += ")"

	}
	return addrstr
}

func checkFlag(flags byte, flag int) bool {
	return (int(flags) & flag) == flag
}

//Description for each flag in AD Flags bitmask
var flagNames = []struct {
	flag int
	name string
}{
	{hci.AdFlagLimitedDisc, "LE Limited Discoverable"},
	{hci.AdFlagGeneralDisc, "LE General Discoverable"},
	{hci.AdFlagNoBrEdr, "BR/EDR not supported"},
	{hci.AdFlagLeBrEdrController, "LE & BR/EDR (controller)"},
	{hci.AdFlagLeBrEdrHost, "LE & BR/EDR (host)"},
}

func decodeAdFlags(flags []byte) string {
	if len(flags) != 1 {
		return "(Invalid)"
	}
	str := strings.Builder{}
	str.WriteString("[")
	for i := 7; i >= 0; i-- {
		if checkFlag(flags[0], (0x01 << uint8(i))) {
			str.WriteString("1")
		} else {
			str.WriteString("0")
		}
	}
	str.WriteString("]")
	if flags[0] == 0 {
		return str.String()
	}
	str.WriteString("(")
	hasFlag := false
	for _, fl := range flagNames {
		if checkFlag(flags[0], fl.flag) {
			if hasFlag {
				str.WriteString(",")
			}
			str.WriteString(fl.name)
			hasFlag = true
		}
	}
	str.WriteString(")")
	return str.String()
}

func decodeDeviceAddress(data []byte) string {
	if len(data) != 7 {
		return "(invalid)"
	}
	addr := hci.ToBtAddress(data[1:])
	if data[0]&0x01 == 0x01 {
		addr.Atype = hci.LeRandomAddress
	}
	return formatAddress(addr)
}

func decodeServiceData(data []byte) string {
	// Service Data starts with 16-bit UUID followed by service data
	// Supplement to Bluetooth Core Specification ch 1.11
	if len(data) < 2 {
		return fmt.Sprintf("0x%x", data)
	}
	sb := strings.Builder{}
	uuid := binary.LittleEndian.Uint16(data[0:2])
	sb.WriteString(fmt.Sprintf("UUID: 0x%.4x", uuid))
	switch uuid {
	case 0xfd6f:
		// Google & Apple Exposure Notification for COVID-19
		// https://covid19-static.cdn-apple.com/applications/covid19/current/static/contact-tracing/pdf/ExposureNotification-BluetoothSpecificationv1.2.pdf?1
		sb.WriteString(", Exposure Notification")
		if len(data) < 22 {
			sb.WriteString(fmt.Sprintf("\n\t\t(invalid data) 0x%x", data[2:]))
		} else {
			sb.WriteString(fmt.Sprintf("\n\t\tProximity Identifier: 0x%x, Encrypted Metadata: 0x%x", data[2:18], data[18:]))
		}
	default:
		if len(data) > 2 {
			sb.WriteString(fmt.Sprintf(", Data: 0x%x", data[2:]))
		}
	}
	return sb.String()
}

func decodeVendorSpecificData(data []byte) string {
	if len(data) < 2 {
		return fmt.Sprintf("0x%x", data)
	}
	sb := strings.Builder{}
	companyID := binary.LittleEndian.Uint16(data[0:2])
	if name, found := btCompanyIdentifiers[int(companyID)]; found {
		sb.WriteString(fmt.Sprintf("%s (0x%.4x)", name, companyID))
	} else {
		sb.WriteString(fmt.Sprintf("Unknown company ID 0x%.4x", companyID))
	}
	if len(data) > 2 {
		sb.WriteString(fmt.Sprintf(", Data: 0x%x", data[2:]))
	}
	return sb.String()
}

func decodeDeviceName(data []byte) string {
	return fmt.Sprintf("Name: \"%s\"", string(data))
}

func defaultDecode(data []byte) string {
	return fmt.Sprintf("Data: 0x%x", data)
}

type adDataDecodingFunc func([]byte) string

var adDataDecoders = map[hci.AdType]adDataDecodingFunc{
	hci.AdFlags:                decodeAdFlags,
	hci.AdDeviceAddress:        decodeDeviceAddress,
	hci.AdServiceData:          decodeServiceData,
	hci.AdManufacturerSpecific: decodeVendorSpecificData,
	hci.AdCompleteLocalName:    decodeDeviceName,
	hci.AdShortenedLocalName:   decodeDeviceName,
}

func decodeAdStructure(ad *hci.AdStructure) string {

	decFunc, ok := adDataDecoders[ad.Typ]
	if !ok {
		decFunc = defaultDecode
	}
	decoded := decFunc(ad.Data)
	return fmt.Sprintf("%s: %s", ad.Typ, decoded)
}
