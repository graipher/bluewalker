package ruuvi

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Ruuvi Tag data format as specified in
// https://github.com/ruuvi/ruuvi-sensor-protocols

// Offsets for the ruuvi measurement data
// These are for format version 3
const (
	formatOffset          int = 0
	v3HumidityOffset      int = 1
	v3TempOffset          int = 2
	v3TempFractionOffset  int = 3
	v3PressureOffset      int = 4
	v3AccelerationXOffset int = 6
	v3AccelerationYOffet  int = 8
	v3AccelerationZOffset int = 10
	v3BatteryVoltageOffet int = 12
)

// number of bytes of data to expect
const (
	v3DataLength int = 14
	v5DataLength int = 24
)

const (
	formatV3 int = 3
	formatV5 int = 5
)

//"Not available" values for data fields available only in Ruuvi tag data format 5
//If Data is parsed from data format v3 the relevant fields are set to these
//values
const (
	TxPowerNA   int = 31
	MoveCountNA int = 255
	SeqnoNA     int = 0xffff
)

//Data contains measurement information parsed from the vendor specific
//data sent by Ruuvi tag
//If data is decoded from Ruuvi tag data version 3, the TxPower, MoveCount
//and Seqno fields are set to TxPowerNA, MoveCountNA and SeqnoNA, respectively
//as they are only available in Ruuvi Data format 5.
//See https://github.com/ruuvi/ruuvi-sensor-protocols for details of Ruuvi
//data protocols
type Data struct {
	Humidity      float32 `json:"humidity"`
	Temperature   float32 `json:"temperature"`
	Pressure      int     `json:"pressure"`
	AccelerationX float32 `json:"accelerationX"`
	AccelerationY float32 `json:"accelerationY"`
	AccelerationZ float32 `json:"accelerationZ"`
	Voltage       int     `json:"voltage"`
	// V5 Only
	TxPower   int `json:"txpower"`
	MoveCount int `json:"movementCount"`
	Seqno     int `json:"sequence"`
}

func readAccl(rd *bytes.Reader) float32 {
	var accl int16
	binary.Read(rd, binary.BigEndian, &accl)
	return float32(accl) / 1000
}

func decodeV3Data(data []byte) (*Data, error) {
	ret := new(Data)
	ret.Humidity = float32(data[v3HumidityOffset]) / 2

	//MSB is sign, next 7 bits are decimal value
	temp := int(data[v3TempOffset] & 0x7f)
	if data[v3TempOffset]&0x80 == 0x80 {
		temp = -temp
		ret.Temperature = float32(temp) - float32(data[v3TempFractionOffset])/100
	} else {
		ret.Temperature = float32(temp) + float32(data[v3TempFractionOffset])/100
	}

	ret.Pressure = int(binary.BigEndian.Uint16(data[v3PressureOffset:v3PressureOffset+2])) + 50000
	ret.Voltage = int(binary.BigEndian.Uint16(data[v3BatteryVoltageOffet:]))

	rd := bytes.NewReader(data[v3AccelerationXOffset:v3BatteryVoltageOffet])
	ret.AccelerationX = readAccl(rd)
	ret.AccelerationY = readAccl(rd)
	ret.AccelerationZ = readAccl(rd)

	// Not available is signified by largest presentable number for unsigned
	// values, smallest presentable number for signed values
	ret.TxPower = 0x001f
	ret.MoveCount = 0xff
	ret.Seqno = 0xffff

	return ret, nil
}

func decodeV5Data(data []byte) (*Data, error) {

	be := binary.BigEndian
	var s16 int16
	var u16 uint16
	var u8 uint8

	ret := new(Data)
	rd := bytes.NewReader(data[formatOffset+1:])

	binary.Read(rd, be, &s16)
	ret.Temperature = float32(s16) * 0.005
	binary.Read(rd, be, &u16)
	ret.Humidity = float32(u16) * 0.0025
	binary.Read(rd, be, &u16)
	ret.Pressure = int(u16) + 50000

	ret.AccelerationX = readAccl(rd)
	ret.AccelerationY = readAccl(rd)
	ret.AccelerationZ = readAccl(rd)
	// power info
	binary.Read(rd, be, &u16)
	// Power info (11+5bit unsigned), first 11bits unsigned is the battery
	// voltage above 1.6V, in millivolts (1.6V to 3.647V range). last 5 bits
	// unsigned is the TX power above -40dBm, in 2dBm steps.
	tmp := (u16 & 0xFFE0) >> 5
	ret.Voltage = int(1600 + tmp)
	tmp = (u16 & 0x001f)
	ret.TxPower = -40 + int((tmp * 2))

	binary.Read(rd, be, &u8)
	ret.MoveCount = int(u8)
	binary.Read(rd, be, &u16)
	ret.Seqno = int(u16)

	return ret, nil
}

//Unmarshall parses Ruuvi Data from given byte array and returns Data struct
//containing the read values
//Renamed to Decode(), this alias is here just for backwards compatability
func Unmarshall(data []byte) (*Data, error) {
	return Decode(data)
}

//Decode decodes Ruuvi data from given advertising data. Returned Data
//structrure contains the decoded values.
func Decode(data []byte) (*Data, error) {

	// Check if the data contains the Ruuvi Manufacturer ID, strip it
	// Otherwise we assume the Manufacturer ID is already stripped
	if len(data) >= 2 && binary.LittleEndian.Uint16(data) == 0x0499 {
		data = data[2:]
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("not enough data for Ruuvi data")
	}
	format := data[formatOffset]
	switch int(format) {
	case formatV3:
		if len(data) < v3DataLength {
			return nil, fmt.Errorf("expected at least %d bytes of data, got %d", v3DataLength, len(data))
		}
		return decodeV3Data(data)
	case formatV5:
		if len(data) < v5DataLength {
			return nil, fmt.Errorf("expected at least %d bytes of data, got %d", v5DataLength, len(data))
		}
		return decodeV5Data(data)
	default:
		return nil, fmt.Errorf("ruuvi Data format %d not supported", data[formatOffset])
	}
}
