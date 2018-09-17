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
	formatOffset        int = 0
	humidityOffset      int = 1
	tempOffset          int = 2
	tempFractionOffset  int = 3
	pressureOffset      int = 4
	accelerationXOffset int = 6
	accelerationYOffet  int = 8
	accelerationZOffset int = 10
	batteryVoltageOffet int = 12
)

// number of bytes of data to expect
const (
	dataLength int = 14
)

const (
	supportedFormat int = 3
)

//Data contains measurement information parsed from the vendor specific
//data sent by Ruuvi tag
type Data struct {
	Humidity      float32
	Temperature   float32
	Pressure      int
	AccelerationX float32
	AccelerationY float32
	AccelerationZ float32
	Voltage       int
}

func readAccl(rd *bytes.Reader) (float32, error) {

	var accl int16
	err := binary.Read(rd, binary.BigEndian, &accl)
	return float32(accl) / 1000, err
}

//Unmarshall parses Ruuvi Data from given byte array and returns Data struct
//containing the read values
func Unmarshall(data []byte) (*Data, error) {

	if len(data) < dataLength {
		return nil, fmt.Errorf("Expected at least %d bytes of data, got %d", dataLength, len(data))
	}
	// Check if the data contains the Ruuvi Manufacturer ID, strip it
	// Otherwise we assume the Manufacturer ID is already stripped
	if len(data) >= dataLength+2 && binary.LittleEndian.Uint16(data) == 0x0499 {
		data = data[2:]
	}

	if int(data[formatOffset]) != supportedFormat {
		return nil, fmt.Errorf("Ruuvi Data format %d not supported", data[formatOffset])
	}
	ret := new(Data)

	ret.Humidity = float32(data[humidityOffset]) / 2

	//MSB is sign, next 7 bits are decimal value
	temp := int(data[tempOffset] & 0x7f)
	if data[tempOffset]&0x80 == 0x80 {
		temp = -temp
		ret.Temperature = float32(temp) - float32(data[tempFractionOffset])/100
	} else {
		ret.Temperature = float32(temp) + float32(data[tempFractionOffset])/100
	}

	ret.Pressure = int(binary.BigEndian.Uint16(data[pressureOffset:pressureOffset+2])) + 50000
	ret.Voltage = int(binary.BigEndian.Uint16(data[batteryVoltageOffet:]))

	rd := bytes.NewReader(data[accelerationXOffset:batteryVoltageOffet])
	var err error
	if ret.AccelerationX, err = readAccl(rd); err != nil {
		return nil, fmt.Errorf("Error while reading acceleration: %s", err.Error())
	}
	if ret.AccelerationY, err = readAccl(rd); err != nil {
		return nil, fmt.Errorf("Error while reading acceleration: %s", err.Error())
	}
	if ret.AccelerationZ, err = readAccl(rd); err != nil {
		return nil, fmt.Errorf("Error while reading acceleration: %s", err.Error())
	}

	return ret, nil
}
