package main

import (
	"bytes"
	"testing"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

func toAddr(addr string, atype hci.BtAddressType) hci.BtAddress {
	ret, _ := hci.BtAddressFromString(addr)
	ret.Atype = atype
	return ret
}

func TestParseAddressFilter(t *testing.T) {

	testData := []struct {
		name     string
		input    string
		valid    bool
		expected []hci.BtAddress
	}{
		{
			"valid",
			"11:22:33:44:55:66",
			true,
			[]hci.BtAddress{toAddr("11:22:33:44:55:66", hci.LePublicAddress)},
		},
		{
			"valid",
			"11:22:33:44:55:66,public",
			true,
			[]hci.BtAddress{toAddr("11:22:33:44:55:66", hci.LePublicAddress)},
		},
		{
			"valid, random",
			"11:22:33:44:55:66, random",
			true,
			[]hci.BtAddress{toAddr("11:22:33:44:55:66", hci.LeRandomAddress)},
		},
		{
			"valid, private",
			"11:22:33:44:55:66,private",
			true,
			[]hci.BtAddress{toAddr("11:22:33:44:55:66", hci.LeRandomAddress)},
		},
		{
			"valid, two addresses",
			"11:22:33:44:55:66,random;aa:bb:cc:11:22:33",
			true,
			[]hci.BtAddress{toAddr("11:22:33:44:55:66", hci.LeRandomAddress),
				toAddr("aa:bb:cc:11:22:33", hci.LePublicAddress)},
		},
		{
			"valid, two addresses",
			"11:22:33:44:55:66,random; aa:bb:cc:11:22:33",
			true,
			[]hci.BtAddress{toAddr("11:22:33:44:55:66", hci.LeRandomAddress),
				toAddr("aa:bb:cc:11:22:33", hci.LePublicAddress)},
		},
		{
			"Invalid address",
			"11:22:44",
			false,
			[]hci.BtAddress{},
		},
		{
			"Invalid address type",
			"11:22:33:44:55:66,foo",
			false,
			[]hci.BtAddress{},
		},
		{
			"too more types",
			"11:22:44:55:66,public,private",
			false,
			[]hci.BtAddress{},
		},
		{
			"No 2nd address",
			"11:22:33:44:55:66;",
			false,
			[]hci.BtAddress{},
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			filts, err := parseAddressFilters(test.input)
			if !test.valid {
				if err == nil {
					t.Errorf("%s: expected parsing to fail", test.name)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %s", test.name, err.Error())
				} else {
					for _, a := range test.expected {
						report := hci.AdvertisingReport{
							EventType: hci.AdvInd,
							Address:   a,
							Data:      nil,
							Rssi:      -70,
						}
						if !filts.Filter(&report) {
							t.Errorf("%s: no match in any of the filters for %s", test.name, a.String())
						}
					}
				}
			}
		})
	}
}

func TestParsePartialAddressFilter(t *testing.T) {
	testdata := []struct {
		name  string
		input string
		valid bool
		match []hci.BtAddress
	}{
		{
			"valid short",
			"0xaa",
			true,
			[]hci.BtAddress{toAddr("aa:bb:cc:dd:ee:ff", hci.LePublicAddress)},
		},
		{
			"valid, short, BD_ADDR",
			"aa:",
			true,
			[]hci.BtAddress{toAddr("aa:bb:cc:dd:ee:ff", hci.LePublicAddress)},
		},
		{
			"valid, longer, BD_ADDR",
			"aa:bb",
			true,
			[]hci.BtAddress{toAddr("aa:bb:cc:dd:ee:ff", hci.LePublicAddress)},
		},
		{
			"valid, longer, BD_ADDR, trailing :",
			"aa:bb:cc:",
			true,
			[]hci.BtAddress{toAddr("aa:bb:cc:dd:ee:ff", hci.LePublicAddress)},
		},
		{
			"valid longer",
			"0xaabb",
			true,
			[]hci.BtAddress{toAddr("aa:bb:cc:dd:ee:ff", hci.LePublicAddress)},
		},
		{
			"valid full",
			"0xaabbccddeeff",
			true,
			[]hci.BtAddress{toAddr("aa:bb:cc:dd:ee:ff", hci.LePublicAddress)},
		},
		{
			"valid, full, BD_ADDR",
			"aa:bb:cc:dd:ee:ff",
			true,
			[]hci.BtAddress{toAddr("aa:bb:cc:dd:ee:ff", hci.LePublicAddress)},
		},
		{
			"invalid, too long",
			"0xaabbccddeeff11",
			false,
			[]hci.BtAddress{},
		},
		{
			"invalid, too long, BD_ADDR",
			"aa:bb:cc:dd:ee:ff:11",
			false,
			[]hci.BtAddress{},
		},
		{
			"invalid chars",
			"0xaabbccjjeeff",
			false,
			[]hci.BtAddress{},
		},
		{
			"invalid chars, BD_ADDR",
			"aa:bb:cc:jj:ee:ff",
			false,
			[]hci.BtAddress{},
		},
		{
			"invalid, too long parts, BD_ADDR",
			"aa:bb:cc01:dd:ee:ff",
			false,
			[]hci.BtAddress{},
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			filt, err := parsePartialAddrFilter(test.input)
			if !test.valid {
				if err == nil {
					t.Errorf("%s: expected parsing to fail", test.name)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error %v", test.name, err)
					return
				}
				for _, a := range test.match {
					report := hci.AdvertisingReport{
						EventType: hci.AdvInd,
						Address:   a,
						Data:      nil,
						Rssi:      -70,
					}
					if !filt.Filter(&report) {
						t.Errorf("%s: no match for filter with address %s", test.name, a.String())
					}
				}
			}
		})
	}
}

func TestParseAdTypeFilter(t *testing.T) {

	testdata := []struct {
		name  string
		input string
		valid bool
		types []hci.AdType
	}{
		{
			"valid",
			"0x01",
			true,
			[]hci.AdType{hci.AdFlags},
		},
		{
			"valid, no 0x",
			"01",
			true,
			[]hci.AdType{hci.AdFlags},
		},
		{
			"two",
			"0x01, 0x03",
			true,
			[]hci.AdType{hci.AdFlags, hci.AdComplete16BitService},
		},
		{
			"too may bytes",
			"0x0102",
			false,
			[]hci.AdType{},
		},
		{
			"invalid byte array",
			"0xfj",
			false,
			[]hci.AdType{},
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			filts, err := parseAdTypeFilters(test.input)
			if !test.valid {
				if err == nil {
					t.Errorf("%s: expected error", test.name)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error %s", test.name, err.Error())
				}
				addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
				for _, adt := range test.types {
					report := hci.AdvertisingReport{
						EventType: hci.AdvInd,
						Address:   addr,
						Data:      []*hci.AdStructure{{Typ: adt, Data: []byte{}}},
					}
					if !filts.Filter(&report) {
						t.Errorf("%s : No match in filter for type %v", test.name, adt)
					}
				}
			}
		})
	}
}

func TestParseVendorSpecFilter(t *testing.T) {

	testdata := []struct {
		name       string
		input      string
		valid      bool
		vendorData []byte
	}{
		{
			"valid",
			"0x010203",
			true,
			[]byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			"valid, no 0x",
			"010203",
			true,
			[]byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			"Invalid byte array",
			"10203",
			false,
			[]byte{},
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			filter, err := parseVendorSpecFilter(test.input)
			if !test.valid {
				if err == nil {
					t.Errorf("%s Expected error", test.name)
				}
			} else {
				if err != nil {
					t.Fatalf("%s unexpected error: %s", test.name, err.Error())
				}
				addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
				report := hci.AdvertisingReport{
					EventType: hci.AdvInd,
					Address:   addr,
					Data:      []*hci.AdStructure{{Typ: hci.AdManufacturerSpecific, Data: test.vendorData}},
				}
				if !filter.Filter(&report) {
					t.Errorf("%s Filter did not match vendor data", test.name)
				}
			}
		})
	}
}
func TestParseAdDataFilter(t *testing.T) {

	testdata := []struct {
		name   string
		input  string
		valid  bool
		values []struct {
			data []byte
			typ  hci.AdType
		}
	}{
		{
			"valid",
			"0x09,0x010203",
			true,
			[]struct {
				data []byte
				typ  hci.AdType
			}{{
				[]byte{0x01, 0x02, 0x03, 0x04},
				hci.AdCompleteLocalName,
			}},
		},
		{
			"valid, no 0x",
			"09, 010203",
			true,
			[]struct {
				data []byte
				typ  hci.AdType
			}{{
				[]byte{0x01, 0x02, 0x03, 0x04},
				hci.AdCompleteLocalName,
			}},
		},
		{
			"valid, multiple",
			"0x09, 0x010203; 0x01, 0x41",
			true,
			[]struct {
				data []byte
				typ  hci.AdType
			}{
				{
					[]byte{0x01, 0x02, 0x03, 0x04},
					hci.AdCompleteLocalName,
				},
				{

					[]byte{0x41, 0x42},
					hci.AdFlags,
				},
			},
		},
		{
			"Invalid bytes on data",
			"0x09, 0xaab",
			false,
			[]struct {
				data []byte
				typ  hci.AdType
			}{{}},
		},
		{
			"Invalid bytes on type",
			"9, 0xaabb",
			false,
			[]struct {
				data []byte
				typ  hci.AdType
			}{{}},
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			filter, err := parseAdDataFilters(test.input)
			if !test.valid {
				if err == nil {
					t.Errorf("%s Expected error", test.name)
				}
			} else {
				if err != nil {
					t.Fatalf("%s unexpected error: %s", test.name, err.Error())
				}
				addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")

				structs := make([]*hci.AdStructure, len(test.values))
				for i, v := range test.values {
					structs[i] = &hci.AdStructure{Typ: v.typ, Data: v.data}
				}

				report := hci.AdvertisingReport{
					EventType: hci.AdvInd,
					Address:   addr,
					Data:      structs,
				}
				if !filter.Filter(&report) {
					t.Errorf("%s Filter did not match vendor data", test.name)
				}
			}
		})
	}
}

func TestParseIrkFilter(t *testing.T) {

	testdata := []struct {
		name  string
		input string
		valid bool
		addr  hci.BtAddress
	}{
		{
			"valid",
			"0x1abc39e76110ff5ec8715b7907d056ad",
			true,
			toAddr("75:d3:32:a3:db:3a", hci.LeRandomAddress),
		},
		{
			"valid, no 0x",
			" 1abc39e76110ff5ec8715b7907d056ad ",
			true,
			toAddr("75:d3:32:a3:db:3a", hci.LeRandomAddress),
		},
		{
			"too short",
			"bc39e76110ff5ec8715b7907d056ad",
			false,
			toAddr("75:d3:32:a3:db:3a", hci.LeRandomAddress),
		},
		{
			"invalid byte array",
			"1qbc39e76110ff5ec8715b7907d056ad",
			false,
			toAddr("75:d3:32:a3:db:3a", hci.LeRandomAddress),
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			filter, err := parseIrkFilter(test.input)
			if !test.valid {
				if err == nil {
					t.Errorf("%s Expected error", test.name)
				}
			} else {
				if err != nil {
					t.Fatalf("%s unexpected error %s", test.name, err.Error())
				}

				report := hci.AdvertisingReport{
					EventType: hci.AdvInd,
					Address:   test.addr,
					Data:      []*hci.AdStructure{},
				}

				if filter.Filter(&report) {
					t.Errorf("%s expected the filter to match", test.name)
				}
			}
		})
	}
}
func TestParseAdStructue(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		valid    bool
		expected []hci.AdStructure
	}{
		{
			"valid",
			"0x01, 0x010203",
			true,
			[]hci.AdStructure{{Typ: hci.AdFlags, Data: []byte{0x01, 0x02, 0x03}}},
		},
		{
			"valid, multiple",
			"0x01, 0x010203; 0x0a, 0x0001",
			true,
			[]hci.AdStructure{
				{Typ: hci.AdFlags, Data: []byte{0x01, 0x02, 0x03}},
				{Typ: hci.AdTxPower, Data: []byte{0x00, 0x01}},
			},
		},
		{
			"invalid",
			"0x01, 0x02, 0x03",
			false,
			nil,
		},
		{
			"invalid, second",
			"0x01, 0x0203; 0x02, 0x03, 0x04",
			false,
			nil,
		},
		{
			"invalid type",
			"0xgg, 0x010203",
			false,
			nil,
		},
		{
			"invalid data",
			"0x01, 0x02gg03",
			false,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ads, err := parseAdStructures(test.input)
			if test.valid {
				if err != nil {
					t.Errorf("Unexpected error %s", err.Error())
				}

				if len(ads) != len(test.expected) {
					t.Errorf("Parsed invalid number of structures (%d, exepected %d)", len(ads), len(test.expected))
				}
				for i, a := range ads {
					if a.Typ != test.expected[i].Typ {
						t.Errorf("Invalid type %v parsed, expected %v", a.Typ, test.expected[i].Typ)
					}
					if bytes.Compare(a.Data, test.expected[i].Data) != 0 {
						t.Errorf("Parsed data did not match expected data")
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
