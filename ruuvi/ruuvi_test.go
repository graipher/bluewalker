package ruuvi

import (
	"encoding/hex"
	"testing"
)

type unmarshallTest struct {
	name   string
	data   string
	result *Data
}

var unmarshallTests = []unmarshallTest{
	{
		"empty",
		"",
		nil,
	},
	{
		"Invalid format",
		"99040a00000000000000000000000000",
		nil,
	},
	{
		"Invalid length",
		"990403000000000000000000000000",
		nil,
	},
	{
		"Zero",
		"99040300000000000000000000000000",
		&Data{Humidity: 0, Temperature: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0, Voltage: 0},
	},
	{
		"Zero - No Company identifier",
		"0300000000000000000000000000",
		&Data{Humidity: 0, Temperature: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0, Voltage: 0},
	},
	{
		"vector1",
		"990403808145c87dfc1803e8fc180e10",
		&Data{
			Humidity: 64, Temperature: -1.69, Pressure: 101325,
			AccelerationX: -1, AccelerationY: 1, AccelerationZ: -1,
			Voltage: 3600,
		},
	},
	{
		"vector2",
		"990403800145c87dfc1803e8fc170e10",
		&Data{
			Humidity: 64, Temperature: 1.69, Pressure: 101325,
			AccelerationX: -1, AccelerationY: 1, AccelerationZ: -1.001,
			Voltage: 3600,
		},
	},
}

func TestUnmarshall(t *testing.T) {

	for _, test := range unmarshallTests {

		data, err := hex.DecodeString(test.data)
		if err != nil {
			t.Fatalf("Invalid data definition on test %s : %s ", test.name, err.Error())
		}
		t.Run(test.name, func(t *testing.T) {

			ret, err := Unmarshall(data)
			if test.result != nil {
				if err != nil {
					t.Errorf("Unexpected error while unmarshalling: %s", err.Error())
				} else {
					if *test.result != *ret {
						t.Errorf("Unexpected result. Expected %+v, got %+v", *test.result, *ret)
					}
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got none")

				}
			}
		})
	}
}
