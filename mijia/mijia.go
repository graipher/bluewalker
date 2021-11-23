package mijia

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Mijia data format as specified in
// https://github.com/pvvx/ATC_MiThermometer#bluetooth-advertising-formats
// Other model ids for search engines: LYWSD03MMC and MHO-C401
//
// Number of bytes of data to expect. Seems the data is missing the
// size and uid params from the beginning when we receive it. Thus len-2.
const (
	atcDataLength int = 14
	customDataLength int = 17
	flagsNotInUse uint16 = 0x20
)

//Data contains measurement information parsed from the vendor specific
//data sent by Mijia
type Data struct {
	Uuid          uint16  `json:"uuid"`
	Mac           [6]byte `json:"mac"`
	Temperature   float32 `json:"temperature"`
	Humidity      float32 `json:"humidity"`
	Voltage       float32 `json:"voltage"`
	Level         uint16  `json:"level"`
	Counter       uint16  `json:"counter"`
	// Custom Only, bits 0-4 in use, we use 5th to tell flags are not in use
	Flags         uint16  `json:"flags"`
}

func decodeData(data []byte) (*Data, error) {

	le := binary.LittleEndian
	var s16 int16
	var u16 uint16
	var u8 uint8

	ret := new(Data)
	rd := bytes.NewReader(data)

	binary.Read(rd, le, &u16)
	ret.Uuid = uint16(u16)
	for i := 0; i < 6; i++ {
		binary.Read(rd, le, &u8)
		ret.Mac[i] = uint8(u8)
	}

	binary.Read(rd, le, &s16)
	ret.Temperature = float32(s16) /100
	binary.Read(rd, le, &u16)
	ret.Humidity = float32(u16) / 100
	binary.Read(rd, le, &u16)
	ret.Voltage = float32(u16) / 1000
	binary.Read(rd, le, &u8)
	ret.Level = uint16(u8)
	binary.Read(rd, le, &u8)
	ret.Counter = uint16(u8)
	if len(data) == customDataLength {
		binary.Read(rd, le, &u8)
		ret.Flags = uint16(u8)
	} else {
		ret.Flags = flagsNotInUse
	}
	return ret, nil
}

//Decode decodes Mijia data from given advertising data. Returned Data
//structrure contains the decoded values.
func Decode(data []byte) (*Data, error) {

	// Check if ATC or Custom data format
	if len(data) != atcDataLength && len(data) != customDataLength {
		return nil, fmt.Errorf("not suitable size (%d) for Mijio data", len(data))
	}
	return decodeData(data)
}
