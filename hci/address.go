package hci

import (
	"encoding/hex"
	"encoding/json"
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

//MarshalJSON marshals BtAddress into JSON
func (ba BtAddress) MarshalJSON() ([]byte, error) {

	return json.Marshal(struct {
		Address string `json:"address"`
		Type    string `json:"type"`
	}{ba.String(), ba.Atype.String()})
}

//IsStatic returns true if address is static random address
// See Bluetooth 5.0, vol6, part B, ch 1.3.2.1
func (ba BtAddress) IsStatic() bool {
	if ba.Atype != LeRandomAddress {
		return false
	}
	return ba.raw[5]&0xc0 == 0xc0
}

//IsNonResolvable returns true if address is non-resolvable private random address
// See Bluetooth 5.0, vol6, part B, ch 1.3.2.2
func (ba BtAddress) IsNonResolvable() bool {
	if ba.Atype != LeRandomAddress {
		return false
	}
	return ba.raw[5]&0xc0 == 0x00
}

//IsResolvable returns true if address is resolvable private random address
// See Bluetooth 5.0, vol6, part B, ch 1.3.2.2
func (ba BtAddress) IsResolvable() bool {
	if ba.Atype != LeRandomAddress {
		return false
	}
	return ba.raw[5]&0xc0 == 0x40
}

//UnmarshalJSON parses the JSON encoded Bluetooth Address (as returned by
//ba.MarshalJSON()) and sets values of this address as those parsed
func (ba *BtAddress) UnmarshalJSON(b []byte) error {

	s := struct {
		Address string `json:"address"`
		Type    string `json:"type"`
	}{}

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	tmp, err := BtAddressFromString(s.Address)
	if err != nil {
		return err
	}
	ba.raw = tmp.raw
	switch s.Type {
	case "LE Public":
		ba.Atype = LePublicAddress
	case "LE Random":
		ba.Atype = LeRandomAddress
	case "BR/EDR":
		ba.Atype = BrEdrAddress
	default:
		return fmt.Errorf("Invalid bluetooth address type")
	}

	return nil
}

// Contstants for Bluetooth address type
const (
	LePublicAddress BtAddressType = 0x00
	LeRandomAddress BtAddressType = 0x01
	BrEdrAddress    BtAddressType = 0x02
)

func (t BtAddressType) String() string {
	switch t {
	case LePublicAddress:
		return "LE Public"
	case LeRandomAddress:
		return "LE Random"
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
