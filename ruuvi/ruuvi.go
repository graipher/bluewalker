// Package ruuvi implements parsing for the vendor specific data sent by
// Ruuvi tag. See https://github.com/ruuvi/ruuvi-sensor-protocols for details
// of Ruuvi data protocols.
//
// This package supports protocol specifications for dataformats
// version 3 (RAWv1), 5 (RAWv2), and 6.
package ruuvi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// Ruuvi Tag data format as specified in
// https://github.com/ruuvi/ruuvi-sensor-protocols

// Offsets for the ruuvi measurement data
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
	v6DataLength int = 19
)

const (
	formatV3 int = 3
	formatV5 int = 5
	formatV6 int = 6
)

// Values returned for Invalid / Not available" values.
//
// If Ruuvi tag does not have a sensor or data is not available, the following
// values are returned.
const (
	// TemperatureNA indicates temperature is invalid or not available.
	TemperatureNA float32 = -163.84 // 0x8000
	// PressureNA indicates pressure is invalid or not available.
	PressureNA int = 115535 // 65535
	// HumidityNA indicates humidity is invalid or not available.
	HumidityNA float32 = 163.8375 // 65535
	// VoltageNA indicates voltage is invalid or not available.
	VoltageNA int = 3647 // 2047
	// AccelerationNA indicates acceleration is invalid or not available.
	AccelerationNA float32 = -32.768 // 0x8000
	// TxPowerNA indicates transmit power is invalid or not available.
	TxPowerNA int = 22 // 31
	// MoveCountNA indicates movement counter is invalid or not available.
	MoveCountNA int = 255
	// SeqnoNA indicates sequence number is invalid or not available.
	SeqnoNA int = 0xffff
	// PM2_5NA indicates PM2.5 is invalid or not available.
	PM2_5NA float32 = 6553.5 // 65535
	//CO2NA indicates CO2 is invalid or not available.
	CO2NA float32 = 65535 // 65535
	// VOCNA indicates VOC is invalid or not available.
	VOCNA int = 511 // 0x1FF
	// NOXNA indicates VOC is invalid or not available.
	NOXNA int = 511 // 0x1FF
	// LuminosityNA indicates luminosity is invalid or not available.
	LuminosityNA float32 = 68459.87570473587 // 255
)

// Data contains measurement information parsed from the vendor specific
// data sent by Ruuvi tag. See
// https://github.com/ruuvi/ruuvi-sensor-protocols for details of Ruuvi
// data protocols.
//
// If Ruuvi tag does not have a sensor or data is not available, the following
// values are returned accordingly: TemperatureNA, PressureNA, HumidityNA,
// VoltageNA, AccelerationNA, TxPowerNA, MoveCountNA, SeqnoNA.
//
// Ruuvi tag data protocol version 3 does not contain the TxPower, MoveCount
// or Seqno. Some models of Ruuvi tag may not contain all sensors.
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
	// V6 Only
	PM2_5       float32 `json:"pm2_5"`
	CO2         float32 `json:"co2"`
	VOC         int     `json:"voc"`
	NOX         int     `json:"nox"`
	Luminosity  float32 `json:"luminosity"`
	Calibrating bool    `json:"calibrating"`
}

// TemperatureValid is true when Temperature is valid.
func (d Data) TemperatureValid() bool { return d.Temperature != TemperatureNA }

// PressureValid is true when Pressure is valid.
func (d Data) PressureValid() bool { return d.Pressure != PressureNA }

// HumidityValid is true when Humidity is valid.
func (d Data) HumidityValid() bool { return d.Humidity != HumidityNA }

// VoltageValid is true when Voltage is valid.
func (d Data) VoltageValid() bool { return d.Voltage != VoltageNA }

// TxPowerValid is true when TxPower is valid.
func (d Data) TxPowerValid() bool { return d.TxPower != TxPowerNA }

// MoveCountValid is true when MoveCount is valid.
func (d Data) MoveCountValid() bool { return d.MoveCount != MoveCountNA }

// SeqnoValid is true when Seqno is valid.
func (d Data) SeqnoValid() bool { return d.Seqno != SeqnoNA }

// PM2_5Valid is true when PM2_5 is valid.
func (d Data) PM2_5Valid() bool { return d.PM2_5 != PM2_5NA }

// CO2Valid is true when CO2 is valid.
func (d Data) CO2Valid() bool { return d.CO2 != CO2NA }

// VOCValid is true when VOC is valid.
func (d Data) VOCValid() bool { return d.VOC != VOCNA }

// NOXValid is true when NOX is valid.
func (d Data) NOXValid() bool { return d.NOX != NOXNA }

// LuminoValid is true when Lumino is valid.
func (d Data) LuminoValid() bool { return d.Luminosity != LuminosityNA }

// AccelerationValid is true when AccelerationX, AccelerationY and AccelerationZ
// are valid.
func (d Data) AccelerationValid() bool {
	var valid bool
	switch {
	case d.AccelerationX == AccelerationNA:
		valid = false
	case d.AccelerationY == AccelerationNA:
		valid = false
	case d.AccelerationZ == AccelerationNA:
		valid = false
	default:
		valid = true
	}
	return valid
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
	ret.TxPower = TxPowerNA
	ret.MoveCount = MoveCountNA
	ret.Seqno = SeqnoNA
	ret.CO2 = CO2NA
	ret.Calibrating = false
	ret.Luminosity = LuminosityNA
	ret.NOX = NOXNA
	ret.VOC = VOCNA
	ret.PM2_5 = PM2_5NA

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

	// Not available is signified by largest presentable number for unsigned
	// values, smallest presentable number for signed values
	ret.CO2 = CO2NA
	ret.Calibrating = false
	ret.Luminosity = LuminosityNA
	ret.NOX = NOXNA
	ret.VOC = VOCNA
	ret.PM2_5 = PM2_5NA

	return ret, nil
}

func decodeV6Data(data []byte) (*Data, error) {

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

	binary.Read(rd, be, &u16)
	ret.PM2_5 = float32(u16) * 0.1
	binary.Read(rd, be, &u16)
	ret.CO2 = float32(u16)
	binary.Read(rd, be, &u8) // VOC
	voc := u8
	binary.Read(rd, be, &u8) // NOX
	nox := u8

	binary.Read(rd, be, &u8)
	ret.Luminosity = float32(math.Exp(float64(u8)*math.Log(65535+1)/254) - 1)

	binary.Read(rd, be, &u8) // Reserved

	binary.Read(rd, be, &u8)
	ret.Seqno = int(u8)

	binary.Read(rd, be, &u8) // Flags
	ret.Calibrating = u8&0b00000001 != 0
	ret.VOC = (int(voc) << 1) | int((u8&0b01000000)>>6)
	ret.NOX = (int(nox) << 1) | int((u8&0b10000000)>>7)

	// Not available is signified by largest presentable number for unsigned
	// values, smallest presentable number for signed values
	ret.AccelerationX = AccelerationNA
	ret.AccelerationY = AccelerationNA
	ret.AccelerationZ = AccelerationNA
	ret.MoveCount = MoveCountNA
	ret.TxPower = TxPowerNA
	ret.Voltage = VoltageNA

	return ret, nil
}

// Unmarshall parses Ruuvi Data from given byte array and returns Data struct
// containing the read values
// Renamed to Decode(), this alias is here just for backwards compatability
func Unmarshall(data []byte) (*Data, error) {
	return Decode(data)
}

// Decode decodes Ruuvi data from given advertising data. Returned Data
// structrure contains the decoded values.
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
	case formatV6:
		if len(data) < v6DataLength {
			return nil, fmt.Errorf("expected at least %d bytes of data, got %d", v6DataLength, len(data))
		}
		return decodeV6Data(data)
	default:
		return nil, fmt.Errorf("ruuvi Data format %d not supported", data[formatOffset])
	}
}
