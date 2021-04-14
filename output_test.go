package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

func MustCreateAddress(str string, random bool) hci.BtAddress {
	addr, err := hci.BtAddressFromString(str)
	if err != nil {
		panic(err)
	}
	if random {
		addr.Atype = hci.LeRandomAddress
	}
	return addr
}

func TestFormatRandomAddress(t *testing.T) {

	testdata := []struct {
		address  hci.BtAddress
		expected string
	}{
		{
			address:  MustCreateAddress("55:D0:F7:48:79:D1", true),
			expected: "55:d0:f7:48:79:d1,random (resolvable private)",
		},
		{
			address:  MustCreateAddress("30:7D:0F:37:8C:FA", true),
			expected: "30:7d:0f:37:8c:fa,random (non-resolvable private)",
		},
		{
			address:  MustCreateAddress("C8:C6:4B:BD:12:10", true),
			expected: "c8:c6:4b:bd:12:10,random (static)",
		},
	}

	for _, test := range testdata {
		out := formatAddress(test.address)
		if out != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, out)
		}
	}
}

func TestDecodeAdStructure(t *testing.T) {
	testdata := []struct {
		data     hci.AdStructure
		expected string
	}{
		{ // Unknown type
			data:     hci.AdStructure{Typ: hci.AdType(0xFE), Data: []byte{0x00}},
			expected: "Unknown (fe): Data: 0x00",
		},
		{ // empty flags
			data:     hci.AdStructure{Typ: hci.AdFlags, Data: []byte{}},
			expected: "Flags: (Invalid)",
		},
		{ // some flags
			data:     hci.AdStructure{Typ: hci.AdFlags, Data: []byte{0x05}},
			expected: "Flags: [00000101](LE Limited Discoverable,BR/EDR not supported)",
		},
		{ // no flags set
			data:     hci.AdStructure{Typ: hci.AdFlags, Data: []byte{0x00}},
			expected: "Flags: [00000000]",
		},

		{ // Device name
			data:     hci.AdStructure{Typ: hci.AdCompleteLocalName, Data: []byte{'a'}},
			expected: "Complete local name: Name: \"a\"",
		},
		{ // name, no data
			data:     hci.AdStructure{Typ: hci.AdCompleteLocalName, Data: []byte{}},
			expected: "Complete local name: Name: \"\"",
		},
		{ // device address, no data
			data:     hci.AdStructure{Typ: hci.AdDeviceAddress, Data: []byte{}},
			expected: "LE Bluetooth Device Address: (invalid)",
		},
		{ // random device address
			data:     hci.AdStructure{Typ: hci.AdDeviceAddress, Data: []byte{0x01, 0xd1, 0x79, 0x48, 0xf7, 0xd0, 0x55}},
			expected: "LE Bluetooth Device Address: 55:d0:f7:48:79:d1,random (resolvable private)",
		},
		{ // public device address
			data:     hci.AdStructure{Typ: hci.AdDeviceAddress, Data: []byte{0x00, 0xad, 0xbd, 0xcf, 0x79, 0xbd, 0x54}},
			expected: "LE Bluetooth Device Address: 54:bd:79:cf:bd:ad",
		},
		{ // vendor specific, valid
			data:     hci.AdStructure{Typ: hci.AdManufacturerSpecific, Data: []byte{0x4c, 0x00, 0xaa, 0xbb}},
			expected: "Manufacturer Specific: Apple, Inc. (0x004c), Data: 0xaabb",
		},
		{ // vendor specific, valid, unknown company
			data:     hci.AdStructure{Typ: hci.AdManufacturerSpecific, Data: []byte{0x00, 0x4c, 0xaa, 0xbb}},
			expected: "Manufacturer Specific: Unknown company ID 0x4c00, Data: 0xaabb",
		},
		{ // vendor specific, No data
			data:     hci.AdStructure{Typ: hci.AdManufacturerSpecific, Data: []byte{0x4c, 0x00}},
			expected: "Manufacturer Specific: Apple, Inc. (0x004c)",
		},
		{ // vendor specific, not enough data
			data:     hci.AdStructure{Typ: hci.AdManufacturerSpecific, Data: []byte{0x4c}},
			expected: "Manufacturer Specific: 0x4c",
		},
		{ // vendor specific, no data
			data:     hci.AdStructure{Typ: hci.AdManufacturerSpecific, Data: []byte{}},
			expected: "Manufacturer Specific: 0x",
		},
		{ // service data, valid
			data:     hci.AdStructure{Typ: hci.AdServiceData, Data: []byte{0x11, 0x22, 0xaa, 0xbb}},
			expected: "Service Data: UUID: 0x2211, Data: 0xaabb",
		},
		{ // service data, valid, just UUID
			data:     hci.AdStructure{Typ: hci.AdServiceData, Data: []byte{0x11, 0x22}},
			expected: "Service Data: UUID: 0x2211",
		},
		{ // service data, no UUID
			data:     hci.AdStructure{Typ: hci.AdServiceData, Data: []byte{0x11}},
			expected: "Service Data: 0x11",
		},
		{ // service data, empty
			data:     hci.AdStructure{Typ: hci.AdServiceData, Data: []byte{}},
			expected: "Service Data: 0x",
		},
		{ // service data, exposure notification
			data:     hci.AdStructure{Typ: hci.AdServiceData, Data: []byte{0x6f, 0xfd, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x22, 0x22, 0x22, 0x22}},
			expected: "Service Data: UUID: 0xfd6f, Exposure Notification\n\t\tProximity Identifier: 0x11111111111111111111111111111111, Encrypted Metadata: 0x22222222",
		},
		{ // service data, invalid exposure notification
			data:     hci.AdStructure{Typ: hci.AdServiceData, Data: []byte{0x6f, 0xfd, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x22}},
			expected: "Service Data: UUID: 0xfd6f, Exposure Notification\n\t\t(invalid data) 0x1111111111111111111111111111111122",
		},
		{ // service data, invalid exposure notification
			data:     hci.AdStructure{Typ: hci.AdServiceData, Data: []byte{0x6f, 0xfd}},
			expected: "Service Data: UUID: 0xfd6f, Exposure Notification\n\t\t(invalid data) 0x",
		},
	}

	for _, test := range testdata {
		out := decodeAdStructure(&test.data)
		if out != test.expected {
			t.Errorf("Expected \"%s\", got \"%s\"\n", test.expected, out)
		}
	}
}

func TestFileOutput(t *testing.T) {
	dirname, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Unable to create tmp directory: %v", err)
	}
	defer os.RemoveAll(dirname)

	fname := filepath.Join(dirname, "out")
	outdata := "a line\n"

	o, err := outputForFile(fname)
	if err != nil {
		t.Errorf("Unable to create output for %s : %v", fname, err)
	}

	if o.isHumanReadable() {
		t.Error("File output is human readable")
	}

	if e := o.write("a line\n"); e != nil {
		t.Errorf("Can not write to output: %v", e)
	}
	o.Close()

	data, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Errorf("Unable to read file created by output: %v", err)
	}
	if string(data) != outdata {
		t.Errorf("Unexpected contents \"%s\" in output file", string(data))
	}

	fname = filepath.Join(dirname, "out.json")
	o, err = outputForFile(fname)
	if err != nil {
		t.Errorf("Unable to create output for %s : %v", fname, err)
	}
	if e := o.writeAsJSON(struct {
		A int
		B string
	}{A: 1, B: "test"}); e != nil {
		t.Errorf("Unable to write JSON output: %v", e)
	}

	data, err = ioutil.ReadFile(fname)
	if err != nil {
		t.Errorf("Unable to read file created by output: %v", err)
	}
	if string(data) != "{\"A\":1,\"B\":\"test\"}\n" {
		t.Errorf("Unexpected contents \"%s\" in output file", string(data))
	}
}
