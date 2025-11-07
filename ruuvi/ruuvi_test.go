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
		"Invalid length v3",
		"990403000000000000000000000000",
		nil,
	},
	{
		"Invalid Length v5",
		"990405000000000000000000000000",
		nil,
	},
	{
		"Zero",
		"99040300000000000000000000000000",
		&Data{Humidity: 0, Temperature: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0, Voltage: 0,
			TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff},
	},
	{
		"Zero - v5",
		"9904050000000000000000000000000000000000000000000000",
		&Data{Temperature: 0, Humidity: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0,
			Voltage: 1600, TxPower: -40, MoveCount: 0, Seqno: 0,
		},
	},
	{
		"Zero - No Company identifier",
		"0300000000000000000000000000",
		&Data{Humidity: 0, Temperature: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0, Voltage: 0,
			TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff},
	},
	{
		"vector1",
		"990403808145c87dfc1803e8fc180e10",
		&Data{
			Humidity: 64, Temperature: -1.69, Pressure: 101325,
			AccelerationX: -1, AccelerationY: 1, AccelerationZ: -1,
			Voltage: 3600, TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff,
		},
	},
	{
		"vector2",
		"990403800145c87dfc1803e8fc170e10",
		&Data{
			Humidity: 64, Temperature: 1.69, Pressure: 101325,
			AccelerationX: -1, AccelerationY: 1, AccelerationZ: -1.001,
			Voltage: 3600, TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff,
		},
	},
	{
		"vector 1 - v5",
		"99040501c4271ac87dfc18fc18fc18af166403e8010203040506",
		&Data{
			Temperature: 2.26, Humidity: 25.025, Pressure: 101325,
			AccelerationX: -1, AccelerationY: -1, AccelerationZ: -1,
			Voltage: 3000, TxPower: 4, MoveCount: 100, Seqno: 1000,
		},
	},
	{
		"Vector 2 - v5",
		"990405fe3e9c40fffe03e803e803e8af166403e8010203040506",
		&Data{
			Temperature: -2.25, Humidity: 100, Pressure: 115534,
			AccelerationX: 1, AccelerationY: 1, AccelerationZ: 1,
			Voltage: 3000, TxPower: 4, MoveCount: 100, Seqno: 1000,
		},
	},
	{
		"Vector 3 - v5 real world Ruuvi Pro",
		"990405fcbaffffffffff8cfe70fc709576c6f88ac3ddedb16a93",
		&Data{
			Temperature: -4.19, Humidity: HumidityNA, Pressure: PressureNA,
			AccelerationX: -0.116, AccelerationY: -0.4, AccelerationZ: -0.912,
			Voltage: 2795, TxPower: 4, MoveCount: 198, Seqno: 63626,
		},
	},
	{
		"Vector 4 - v5 doc: valid data",
		"0512FC5394C37C0004FFFC040CAC364200CDCBB8334C884F",
		&Data{
			Temperature: 24.3, Humidity: 53.489998, Pressure: 100044,
			AccelerationX: 0.004, AccelerationY: -0.004, AccelerationZ: 1.036,
			Voltage: 2977, TxPower: 4, MoveCount: 66, Seqno: 205,
		},
	},
	{
		"Vector 5 - v5 doc: maximum values",
		"057FFFFFFEFFFE7FFF7FFF7FFFFFDEFEFFFECBB8334C884F",
		&Data{
			Temperature: 163.83499, Humidity: 163.83499, Pressure: 115534,
			AccelerationX: 32.767, AccelerationY: 32.767, AccelerationZ: 32.767,
			Voltage: 3646, TxPower: 20, MoveCount: 254, Seqno: 65534,
		},
	},
	{
		"Vector 6 - v5 doc: minimum values",
		"058001000000008001800180010000000000CBB8334C884F",
		&Data{
			Temperature: -163.83499, Humidity: 0, Pressure: 50000,
			AccelerationX: -32.767, AccelerationY: -32.767, AccelerationZ: -32.767,
			Voltage: 1600, TxPower: -40, MoveCount: 0, Seqno: 0,
		},
	},
	{
		"Vector 7 - v5 doc: invalid values",
		"058000FFFFFFFF800080008000FFFFFFFFFFFFFFFFFFFFFF",
		&Data{
			Temperature: TemperatureNA, Humidity: HumidityNA, Pressure: PressureNA,
			AccelerationX: AccelerationNA, AccelerationY: AccelerationNA, AccelerationZ: AccelerationNA,
			Voltage: VoltageNA, TxPower: TxPowerNA, MoveCount: MoveCountNA, Seqno: SeqnoNA,
		},
	},
	{
		"Vector 8 - v6 doc: valid data",
		"06170C5668C79E007000C90501D900CD004C884F",
		&Data{
			Temperature: 29.5, Pressure: 101102, Humidity: 55.3,
			PM2_5: 11.2, CO2: 201, VOC: 10, NOX: 2, Luminosity: 13026.669,
			Seqno: 205, Calibrating: false,
		},
	},
	{
		"Vector 9 - v6 doc: maximum values",
		"067FFF9C40FFFE27109C40FAFAFE00FF074C8F4F",
		&Data{
			Temperature: 163.83499, Pressure: 115534, Humidity: 100.0,
			PM2_5: 1000.0, CO2: 40000, VOC: 500, NOX: 500, Luminosity: 65535.0,
			Seqno: 255, Calibrating: true,
		},
	},
	{
		"Vector 10 - v6 doc: minimum values",
		"06800100000000000000000000000000004C884F",
		&Data{
			Temperature: -163.83499, Pressure: 50000, Humidity: 0.0,
			PM2_5: 0.0, CO2: 0, VOC: 0, NOX: 0, Luminosity: 0.0,
			Seqno: 0, Calibrating: false,
		},
	},
	{
		"Vector 11 - v6 doc: invalid values",
		"068000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
		&Data{
			Temperature: TemperatureNA, Pressure: PressureNA, Humidity: HumidityNA,
			PM2_5: PM2_5NA, CO2: CO2NA, VOC: VOCNA, NOX: NOXNA, Luminosity: LuminosityNA,
			Seqno: 255, Calibrating: true,
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
