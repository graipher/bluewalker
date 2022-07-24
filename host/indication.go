package host

import "gitlab.com/jtaimisto/bluewalker/hci"

type IndicationType byte

const (
	ConnectionIndication    IndicationType = 0x00
	DisconnectionIndication IndicationType = 0x01
)

//Indication contains information from Host to user
type Indication struct {
	Type   IndicationType
	Peer   hci.BtAddress
	Handle hci.ConnectionHandle
}
