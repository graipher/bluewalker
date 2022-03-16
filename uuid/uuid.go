package uuid

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	UUID_LENGTH    int = 16
	UUID_STR_LEN   int = 36
	UUID16_LENGTH  int = 2
	UUID16_STR_LEN int = UUID16_LENGTH * 2
	UUID32_LENGTH  int = 4
	UUID32_STR_LEN int = UUID32_LENGTH * 2
)

// error returned if UUID can not be parsed from string
var ErrString = errors.New("can not parse UUID from input")

// error returned if UUID can not be parsed from given data
var ErrData = errors.New("invalid data length for UUID")

// Empty UUID value
var NoneUuid Uuid

var uuidStrPartLengths = []int{4, 2, 2, 2, 6}

// Bluetooth Core 5.2 Vol 3, part B, ch 2.5.1
type Uuid [16]byte
type Uuid16 uint16
type Uuid32 uint32

// Parse UUID from given string
func FromString(value string) (Uuid, error) {
	var ret Uuid
	if len(value) != UUID_STR_LEN {
		return NoneUuid, ErrString
	}
	parts := strings.Split(value, "-")
	if len(parts) != len(uuidStrPartLengths) {
		return NoneUuid, ErrString
	}

	offset := 0
	for i, p := range uuidStrPartLengths {
		if len(parts[i]) != p*2 {
			return NoneUuid, ErrString
		}
		if val, err := hex.DecodeString(parts[i]); err == nil {
			copy(ret[offset:offset+p], val)
			offset += p
		} else {
			return NoneUuid, ErrString
		}
	}
	return ret, nil
}

// Parse UUID from given byte slice. Error is returned
// if the slice does not contain enough data
func FromBytes(value []byte) (Uuid, error) {
	if len(value) != UUID_LENGTH {
		return NoneUuid, ErrData
	}

	var ret Uuid
	for i := 0; i < UUID_LENGTH; i++ {
		ret[i] = value[UUID_LENGTH-1-i]
	}
	return ret, nil
}

func (u Uuid) String() string {

	bld := strings.Builder{}
	offset := 0
	for i, p := range uuidStrPartLengths {
		bld.Write([]byte(hex.EncodeToString(u[offset : offset+p])))
		offset += p
		if i != len(uuidStrPartLengths)-1 {
			bld.WriteRune('-')
		}
	}
	return bld.String()
}

func (u Uuid16) String() string {
	return fmt.Sprintf("%.4x", uint16(u))
}

func (u Uuid32) String() string {
	return fmt.Sprintf("%.8x", uint32(u))
}

// Parse UUID16 from string. Error is returned if string is malformed
func Uuid16FromString(value string) (Uuid16, error) {
	value = strings.TrimPrefix(value, "0x")
	if len(value) != UUID16_STR_LEN {
		return 0x00, ErrString
	}
	if data, err := hex.DecodeString(value); err == nil {
		return Uuid16(binary.BigEndian.Uint16(data)), nil
	} else {
		return 0x00, ErrString
	}
}

// Parse UUID16 from byte slice. Error is returned if slice does not
// contain enough data
func Uuid16FromBytes(data []byte) (Uuid16, error) {
	if len(data) < UUID16_LENGTH {
		return 0x00, ErrData
	}
	return Uuid16(binary.LittleEndian.Uint16(data)), nil
}

// Parse UUID32 from byte slice. Error is returned if slice does not
// contain enough data
func Uuid32FromBytes(data []byte) (Uuid32, error) {
	if len(data) < UUID32_LENGTH {
		return 0x00, ErrData
	}
	return Uuid32(binary.LittleEndian.Uint32(data)), nil
}

// Parse UUID32 from string. Error is returned if string is malformed
func Uuid32FromString(value string) (Uuid32, error) {
	value = strings.TrimPrefix(value, "0x")
	if len(value) != UUID32_STR_LEN {
		return 0x00, ErrString
	}
	if data, err := hex.DecodeString(value); err == nil {
		return Uuid32(binary.BigEndian.Uint32(data)), nil
	} else {
		return 0x00, ErrString
	}
}
