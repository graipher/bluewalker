package hci

import (
	"encoding/hex"
	"fmt"
	"strings"
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

// BtAddressFromString creates BtAddress struct from given string
// the string should contain Bluetooth address in canonical form
// the returned address has address type set to default (LE Public)
func BtAddressFromString(address string) (BtAddress, error) {

	parts := strings.Split(address, ":")
	if len(parts) != 6 {
		return BtAddress{}, fmt.Errorf("Invalid Bluetooth Address %s", address)
	}
	addr := BtAddress{}
	for i := 0; i < 6; i++ {
		if len(parts[i]) != 2 {
			return BtAddress{}, fmt.Errorf("Invalid Bluetooth Address %s", address)
		}
		b, err := hex.DecodeString(parts[i])
		if err != nil {
			return BtAddress{}, fmt.Errorf("Invalid Bluetooth Address %s", address)
		}
		addr.raw[5-i] = b[0]
	}
	return addr, nil
}
