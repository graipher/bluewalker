package hci

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func verifyUint16(array []byte, expected uint16, name string, t *testing.T) {

	val := binary.LittleEndian.Uint16(array)
	if val != expected {
		t.Errorf("Invalid value for %s (expected %.4x got %.4x)", name, expected, val)
	}
}

func TestCommandExport(t *testing.T) {

	cmd := &CommandPacket{OpCode: CommandReset}
	raw := cmd.Encode()

	if len(raw) != headerLength+1 {
		t.Errorf("Invalid length for encoded command %d ", len(raw))
	}
	verifyUint16(raw[1:], uint16(CommandReset), "OpCode", t)
	if raw[3] != 0 {
		t.Errorf("Invalid parameter length")
	}
}

func TestEncodeWithParameters(t *testing.T) {

	cmd := &CommandPacket{OpCode: CommandReset}
	param := make([]byte, 2)
	param[0] = 0x01
	param[1] = 0x02
	cmd.Parameters(param)

	raw := cmd.Encode()

	if len(raw) != headerLength+len(param)+1 {
		t.Errorf("Invalid total length for packet")
	}

	if raw[3] != byte(len(param)) {
		t.Errorf("Invalid parameter length")
	}
	for i := 0; i < len(param); i++ {
		if raw[4+i] != param[i] {
			t.Errorf("Parameter %d incorrectly encoded", i)
		}
	}
}

type dummyEncodeable uint16

func (d *dummyEncodeable) EncodeTo(data []byte) (int, error) {
	data[0] = byte(uint16(*d) & 0xFF)
	data[1] = byte((uint16(*d) & 0xff00) >> 8)
	return 2, nil
}

func TestCommmandBuilder(t *testing.T) {

	tests := []struct {
		name     string
		build    func() CommandPacket
		expected []byte
	}{
		{
			"add byte",
			func() CommandPacket {
				return NewCommandBuilder(CommandOpCode(CommandLeSetScanEnable), 2).AddByte(0x01).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x02, 0x01, 0x00},
		},
		{
			"add two bytes",
			func() CommandPacket {
				return NewCommandBuilder(CommandOpCode(CommandLeSetScanEnable), 2).AddByte(0x01).AddByte(0x02).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x02, 0x01, 0x02},
		},
		{
			"add uint16",
			func() CommandPacket {
				return NewCommandBuilder(CommandOpCode(CommandLeSetScanEnable), 2).AddUint16(0x0403).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x02, 0x03, 0x04},
		},
		{
			"add two uint16s",
			func() CommandPacket {
				return NewCommandBuilder(CommandOpCode(CommandLeSetScanEnable), 4).AddUint16(0x0403).AddUint16(0x0605).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x04, 0x03, 0x04, 0x05, 0x06},
		},
		{
			"add uint64",
			func() CommandPacket {
				return NewCommandBuilder(CommandOpCode(CommandLeSetScanEnable), 8).AddUint64(0x0807060504030201).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x08, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
		{
			"add two uint64s",
			func() CommandPacket {
				return NewCommandBuilder(CommandOpCode(CommandLeSetScanEnable), 16).
					AddUint64(0x0807060504030201).AddUint64(0x0f0e0d0c0b0a0908).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x10, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
		},
		{
			"add encodeable",
			func() CommandPacket {
				d := dummyEncodeable(0x1122)

				return NewCommandBuilder(CommandLeSetScanEnable, 2).AddEncodeable(&d).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x02, 0x22, 0x11},
		},
		{
			"add encodeable and byte",
			func() CommandPacket {
				d := dummyEncodeable(0x1122)

				return NewCommandBuilder(CommandLeSetScanEnable, 3).
					AddEncodeable(&d).AddByte(0xaa).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x03, 0x22, 0x11, 0xaa},
		},
		{
			"add address and byte",
			func() CommandPacket {
				addr, _ := BtAddressFromString("aa:bb:cc:dd:ee:ff")

				return NewCommandBuilder(CommandLeSetScanEnable, 7).
					AddBtAddress(addr).AddByte(0x11).Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x07, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x11},
		},
		{
			"put byte",
			func() CommandPacket {
				return NewCommandBuilder(CommandLeSetScanEnable, 4).
					AddUint16(0x2211).
					PutByte(1, 0x33).
					AddUint16(0xbbaa).
					Command()
			},
			[]byte{0x01, 0x0c, 0x20, 0x04, 0x11, 0x33, 0xaa, 0xbb},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.build()
			raw := cmd.Encode()
			if !bytes.Equal(raw, test.expected) {
				t.Errorf("%#v did not match expected %#v", raw, test.expected)
			}
		})
	}
}
