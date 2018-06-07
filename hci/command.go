package hci

import (
	"encoding/binary"
	"fmt"
)

// CommandOpCode represents the Operation code for HCI command
// See Bluetooth 5.0 vol 2, part E, ch 5.4.1
type CommandOpCode uint16

const (
	// CommandReset : HCI Reset command
	CommandReset CommandOpCode = 0x0c03
	// CommandSetEventMask : Set Event mask command
	CommandSetEventMask CommandOpCode = 0x0c01
	// CommandWriteLeHostSupported : Set LE Host supported command
	CommandWriteLeHostSupported CommandOpCode = 0x0c6d
	// CommandLeSetEventMask : Set LE Event mask command
	CommandLeSetEventMask CommandOpCode = 0x2001
)

func (op CommandOpCode) String() string {
	switch op {
	case CommandReset:
		return "Reset"
	case CommandLeSetEventMask:
		return "Set LE Event Mask"
	case CommandSetEventMask:
		return "Set Event Mask"
	case CommandWriteLeHostSupported:
		return "Write LE Host Supported"
	default:
		return fmt.Sprintf("Unknown command 0x%.2x", int(op))
	}
}

const (
	headerLength int = 3
)

// the byte order we are using for encoding data
var le binary.ByteOrder = binary.LittleEndian

// CommandPacket defines HCI Command packet
type CommandPacket struct {
	// OpCode for this command
	OpCode     CommandOpCode
	parameters []byte
}

// Parameters add parameters for command packets
// the given byte slice is not copied, do not modify it
func (pkt *CommandPacket) Parameters(params []byte) {
	pkt.parameters = params
}

// Encode encodes the command into properly formed byte array
// See Bluetooth 5.0 vol 2, part E, ch 5.4.1
func (pkt *CommandPacket) Encode() []byte {

	paramlen := 0
	if pkt.parameters != nil {
		paramlen = len(pkt.parameters)
	}
	len := headerLength + paramlen + 1
	ret := make([]byte, len)
	// The RAW transport expects first byte to be the type of packet
	ret[0] = hciCommandPacket
	le.PutUint16(ret[1:], uint16(pkt.OpCode))
	ret[3] = byte(paramlen)
	copy(ret[4:], pkt.parameters)
	return ret
}
