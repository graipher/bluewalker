package hci

import "testing"

func TestDecodeConnectionHandle(t *testing.T) {
	data := []byte{0x2d, 0x00}

	expected := ConnectionHandle(0x002d)

	handle := DecodeConnectionHandle(data)
	if handle != expected {
		t.Errorf("expetced handle %s, decoded handle %s", expected, handle)
	}
	if handle.String() != "002d" {
		t.Errorf("unexpected String() value \"%s\"", handle.String())
	}

}
