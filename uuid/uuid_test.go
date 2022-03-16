package uuid

import (
	"errors"
	"strings"
	"testing"
)

func TestUuidString(t *testing.T) {

	input := "00000000-0000-1000-8000-00805f9b34fb"

	uuid, err := FromString(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := uuid.String()
	if output != input {
		t.Errorf("Unexpected output \"%s\", expected \"%s\"", output, input)
	}
}

func TestUuid16String(t *testing.T) {
	input := "0x1234"

	uuid, err := Uuid16FromString(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := uuid.String()
	input = strings.TrimPrefix(input, "0x")
	if output != input {
		t.Errorf("Unexpected output \"%s\", expected \"%s\"", output, input)
	}
}

func TestUuid16StringInvalid(t *testing.T) {
	testdata := []struct {
		name  string
		input string
	}{
		{
			name:  "too-short",
			input: "0xaa",
		},
		{
			name:  "too-short-no0x",
			input: "aa",
		},
		{
			name:  "invalid-hex",
			input: "0xaaoo",
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			_, err := Uuid16FromString(test.input)
			if !errors.Is(err, ErrString) {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestUuid32String(t *testing.T) {
	input := "0x12345678"

	uuid, err := Uuid32FromString(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := uuid.String()
	input = strings.TrimPrefix(input, "0x")
	if output != input {
		t.Errorf("Unexpected output \"%s\", expected \"%s\"", output, input)
	}
}

func TestUuid32StringInvalid(t *testing.T) {
	testdata := []struct {
		name  string
		input string
	}{
		{
			name:  "too-short",
			input: "0xaa",
		},
		{
			name:  "too-short-no0x",
			input: "aabbcc",
		},
		{
			name:  "invalid-hex",
			input: "0xaabbccoo",
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			_, err := Uuid32FromString(test.input)
			if !errors.Is(err, ErrString) {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestFromBytes(t *testing.T) {
	input := []byte{0xfc, 0x9d, 0xd0, 0xb3, 0xcb, 0x84, 0xe0, 0x84, 0x06, 0x42, 0xf3, 0xf7, 0xe2, 0xe0, 0xbf, 0xcb}

	uuid, err := FromBytes(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if uuid.String() != "cbbfe0e2-f7f3-4206-84e0-84cbb3d09dfc" {
		t.Errorf("unexpected string for parsed UUID: %s", uuid.String())
	}
}

func TestFromBytesInvalid(t *testing.T) {
	input := []byte{0xd0, 0xb3, 0xcb, 0x84, 0xe0, 0x84, 0x06, 0x42, 0xf3, 0xf7, 0xe2, 0xe0, 0xbf, 0xcb}
	_, err := FromBytes(input)
	if !errors.Is(err, ErrData) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUuid16FromBytes(t *testing.T) {
	input := []byte{0xfe, 0xef}

	uuid, err := Uuid16FromBytes(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if uuid.String() != "effe" {
		t.Errorf("unexpected string for parsed UUID: %s", uuid)
	}
}

func TestUuid16FromBytesInvalid(t *testing.T) {
	input := []byte{0xaa}
	_, err := Uuid16FromBytes(input)
	if !errors.Is(err, ErrData) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUuid32FromBytes(t *testing.T) {
	input := []byte{0xfe, 0xef, 0x01, 0x02}

	uuid, err := Uuid32FromBytes(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if uuid.String() != "0201effe" {
		t.Errorf("unexpected string for parsed UUID: %s", uuid)
	}
}
func TestUuid32FromBytesInvalid(t *testing.T) {
	input := []byte{0xaa, 0xbb, 0xcc}
	_, err := Uuid32FromBytes(input)
	if !errors.Is(err, ErrData) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFromInvalidString(t *testing.T) {

	tests := []struct {
		name  string
		input string
	}{{
		name:  "empty",
		input: "",
	},
		{
			name:  "too short",
			input: "cbbfe0e2-f7f3",
		},
		{
			name:  "too many -",
			input: "cbbfe0e2-f7f3-4206-84e0-84cbb-d09dfc",
		},
		{
			name:  "not enough -",
			input: "cbbfe0e2-f7f3-4206-84e0084cbb3d09dfc",
		},
		{
			name:  "invalid part",
			input: "cbbfe0e2-f7f3-4206-84e08-4cbb3d09dfc",
		},
		{
			name:  "invalid part",
			input: "cbbfe0e2-f7f3-4206-x4e0-84cbb3d09dfc",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := FromString(test.input)
			if !errors.Is(err, ErrString) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
