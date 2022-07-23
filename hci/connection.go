package hci

import (
	"encoding/binary"
	"fmt"
)

// ConnectionHandle identifies the connection
// Bluetooth 5.2 vol 4 part E 5.3
type ConnectionHandle uint16

// DecodeConnectionHandle will read ConnectionHandle from given data
func DecodeConnectionHandle(data []byte) ConnectionHandle {
	return ConnectionHandle(binary.LittleEndian.Uint16(data))
}

func (h ConnectionHandle) String() string {
	return fmt.Sprintf("%.2x", uint16(h))
}

type LeConnectionRole byte

const (
	LeConnectionRoleMaster LeConnectionRole = 0x00
	LeConnectionRoleSlave  LeConnectionRole = 0x01
)
