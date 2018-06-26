package hci

import (
	"encoding/hex"
	"testing"
)

func dataFromString(data string, t *testing.T) []byte {

	buf, err := hex.DecodeString(data)
	if err != nil {
		t.Fatalf("Invalid test data (%s)", err.Error())
	}
	return buf
}

func TestParseAdS(t *testing.T) {

	buf := dataFromString("03094100", t)
	ad, err := decodeAdStructure(buf)
	if err != nil {
		t.Fatalf("Unable to decode valid ad structure")
	}
	if ad.Typ != AdCompleteLocalName {
		t.Errorf("Decoded invalid AD Type")
	}
	if len(ad.Data) != 2 || (ad.Data[0] != 0x41 || ad.Data[1] != 0x00) {
		t.Errorf("Decoded invalid AD structure data")
	}
}

func TestZeroStructure(t *testing.T) {

	buf := []byte{0x00}

	ad, err := decodeAdStructure(buf)
	if err != nil {
		t.Fatalf("Was not able to decode zero -length AD structure")
	}
	if ad != nil {
		t.Errorf("Expected nil Structure")
	}
}

func TestInvalidAdStruture(t *testing.T) {

	buf := dataFromString("05094100", t)
	_, err := decodeAdStructure(buf)
	if err == nil {
		t.Errorf("Decoded AD structure with invalid length")
	}

}

func TestParseStructures(t *testing.T) {

	tests := []struct {
		name  string
		buf   string
		count int
	}{{
		"Full",
		"0309410003084100",
		2,
	}, {
		"Zero padded",
		"0309410003084100000000",
		2,
	}, {
		"Zeros in middle",
		"030941000003084100",
		2,
	}, {
		"Invalid",
		"0509410003084100",
		-1,
	},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf, _ := hex.DecodeString(test.buf)
			ads, err := parseAdData(buf)
			if err != nil {
				if test.count != -1 {
					t.Errorf("Unexpected error while parsing AD structures")
				} else {
					// error expcted, continue
					return
				}
			}
			if test.count == -1 && err == nil {
				t.Fatalf("Expected parsing to fail")
			}
			if len(ads) != test.count {
				t.Errorf("Expected %d AD structures, got %d", test.count, len(ads))
			}
		})
	}
}

func TestDecodeAdReportHappy(t *testing.T) {
	buf := dataFromString("010000010203040506080309410003084100ff", t)
	ar, err := DecodeAdvertisingReport(buf)
	if err != nil {
		t.Fatalf("Unexpected error when parsing Advertising Report")
	}
	if len(ar) != 1 {
		t.Fatalf("Expected 1 report, parsed %d", len(ar))
	}
	if ar[0].EventType != AdvInd {
		t.Errorf("Expected %s event, got %s", AdvInd.String(), ar[0].EventType.String())
	}
	expectedAddress := ToBtAddress([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06})
	expectedAddress.Atype = LePublicAddress
	if ar[0].Address != expectedAddress {
		t.Errorf("Expected event from %s, was from %s", expectedAddress.String(), ar[0].Address.String())
	}
	// AdStructure parsing is tested with other functions, just make
	// sure we have enough structures
	if len(ar[0].Data) != 2 {
		t.Errorf("Expected to parse 2 Ad Structures, parsed %d", len(ar[0].Data))
	}
	if ar[0].Rssi != int8(-1) {
		t.Errorf("Parsed invalid RSSI %d", ar[0].Rssi)
	}
}

func TestDecodeAdRecordEmpty(t *testing.T) {

	buf := dataFromString("00", t)
	ar, err := DecodeAdvertisingReport(buf)
	if err != nil {
		t.Errorf("Got unexpected error")
	}
	if len(ar) != 0 {
		t.Errorf("Parsed nonexistent data")
	}
}

func TestReportWithNoAdData(t *testing.T) {

	buf := dataFromString("01000101020304050600ff", t)
	ar, err := DecodeAdvertisingReport(buf)
	if err != nil {
		t.Fatalf("Unexpected error %s ", err.Error())
	}
	if len(ar) != 1 {
		t.Errorf("Parsed %d structres, expected 1", len(ar))
	}
	if ar[0].EventType != AdvInd {
		t.Errorf("Expected %s event, got %s", AdvInd.String(), ar[0].EventType.String())
	}
	expectedAddress := ToBtAddress([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06})
	expectedAddress.Atype = LePrivateAddress
	if ar[0].Address != expectedAddress {
		t.Errorf("Expected event from %s, was from %s", expectedAddress.String(), ar[0].Address.String())
	}
	if ar[0].Rssi != int8(-1) {
		t.Errorf("Parsed invalid RSSI %d", ar[0].Rssi)
	}

	if len(ar[0].Data) != 0 {
		t.Errorf("Expected not AD structures, got %d", len(ar[0].Data))
	}

}

func TestDecodeAdvReportInvalid(t *testing.T) {

	testData := []struct {
		name string
		data []byte
	}{{
		"Empty",
		[]byte{},
	}, {
		"Only num",
		dataFromString("01", t),
	}, {
		"No addr",
		dataFromString("0100", t),
	}, {
		"No addr bytes",
		dataFromString("010000", t),
	}, {
		"with partial Addr",
		dataFromString("0100000102", t),
	}, {
		"with addr, no data",
		dataFromString("010000010203040506", t),
	}, {
		"Invalid Adv Data length",
		dataFromString("01000001020304050608aabb", t),
	}, {
		"Invalid AD data",
		dataFromString("010000010203040506080f09410003084100ff", t),
	}, {
		"No RSSI",
		dataFromString("010000010203040506080309410003084100", t),
	},
	}

	for _, d := range testData {
		t.Run(d.name, func(t *testing.T) {
			if _, err := DecodeAdvertisingReport(d.data); err == nil {
				t.Errorf("Expected error to be returned")
			}
		})
	}
}
