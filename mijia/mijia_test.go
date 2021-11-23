package mijia

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
		"Invalid Length",
		"181a0000000000000000",
		nil,
	},
	{
		"Custom",
		"1a18deadbeef0123bf08d50bf90b600105",
		&Data{Uuid: 0x181a, Mac: [6]byte{222, 173, 190, 239, 1, 35},
			Temperature: 22.39, Humidity: 30.29, Voltage: 3.065,
			Level: 96, Counter: 1, Flags: 5},
	},
}

func TestUnmarshall(t *testing.T) {

	for _, test := range unmarshallTests {

		data, err := hex.DecodeString(test.data)
		if err != nil {
			t.Fatalf("Invalid data definition on test %s : %s ", test.name, err.Error())
		}
		t.Run(test.name, func(t *testing.T) {

			ret, err := Decode(data)
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
