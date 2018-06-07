package hci

import (
	"encoding/hex"
	"fmt"
	"log"
)

// EventCode identifies the HCI Event received
type EventCode byte

const (
	//EventCodeCommandComplete is event code for CommandComplete event
	EventCodeCommandComplete EventCode = 0x0e
	// EventCodeCommandStatus is event code for Command Status event
	EventCodeCommandStatus EventCode = 0x0f
)

func (evt EventCode) String() string {
	switch evt {
	case EventCodeCommandComplete:
		return "Command Complete"
	case EventCodeCommandStatus:
		return "Command Status"
	default:
		return fmt.Sprintf("Unknown event 0x%.2x", int(evt))
	}
}

// Event structure represents received HCI Event
// See Bluetooth 5.0, vol 2, Part E, ch 5.4.4
type Event struct {
	Code       EventCode
	parameters []byte
}

// DecodeEvent decodes HCI Event from given buffer, error is received if
// event can not be decoded
func DecodeEvent(buf []byte) (*Event, error) {

	log.Printf("Event:\n%s", hex.Dump(buf))

	op := buf[0]
	plen := int(buf[1])
	if int(plen) > len(buf)-2 {
		return nil, fmt.Errorf("Too short event packet, expected at least %d bytes of parameters", plen)
	}
	return &Event{Code: EventCode(op), parameters: buf[2 : plen+2]}, nil
}

// CommandCompleteEvent represents Command Complete HCI Event
// see Bluetooth v5.0, vol2, part E, ch 7.7.14
type CommandCompleteEvent struct {
	Event
}

const (
	// minimum length for command complete event
	ccMinParamLength int = 3
)

// DecodeCommandComplete decodes Command Complete HCI Event from
// buffer
func DecodeCommandComplete(evt *Event) (*CommandCompleteEvent, error) {

	if evt.Code != EventCodeCommandComplete {
		return nil, fmt.Errorf("Unexpected event code %.2x", evt.Code)
	}
	if len(evt.parameters) < ccMinParamLength {
		return nil, fmt.Errorf("Not enough paramaters for Command Complete")
	}
	return &CommandCompleteEvent{Event: *evt}, nil
}

// GetNumHciCommandPackets returns the number of Hci commands parameter value
func (cc *CommandCompleteEvent) GetNumHciCommandPackets() byte {
	return cc.parameters[0]
}

// GetCommandOpcode returns the opcode this command complete event was for
func (cc *CommandCompleteEvent) GetCommandOpcode() CommandOpCode {
	return CommandOpCode(le.Uint16(cc.parameters[1:]))
}

// GetReturnParameters returns the return parameters for this event
func (cc *CommandCompleteEvent) GetReturnParameters() []byte {
	return cc.parameters[3:]
}

//GetStatusParameter returns the first parameter as status code.
// XXX bounds check
func (cc *CommandCompleteEvent) GetStatusParameter() ErrorCode {
	return ErrorCode(cc.parameters[3])
}
