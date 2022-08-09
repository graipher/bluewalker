package hci

import (
	"encoding/hex"
	"testing"
)

func TestDecodeValidEvent(t *testing.T) {

	buf := make([]byte, 3)
	buf[0] = byte(EventCodeCommandComplete)
	buf[1] = 1
	buf[2] = 0x12

	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if ev.Code != EventCodeCommandComplete {
		t.Errorf("Deoced invalid event code")
	}
	param := ev.parameters
	if len(param) != 1 || param[0] != 0x12 {
		t.Errorf("Decoded invalid parameters")
	}
}

func TestDecodeShortEvent(t *testing.T) {

	buf := make([]byte, 2)
	buf[0] = byte(EventCodeCommandComplete)
	buf[1] = 0

	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if ev.Code != EventCodeCommandComplete {
		t.Errorf("Decoded invalid event code")
	}
	if len(ev.parameters) != 0 {
		t.Errorf("Decoded parameters even there were none")
	}
}

func TestDecodeTooShortEvent(t *testing.T) {

	buf := make([]byte, 2)
	buf[0] = byte(EventCodeCommandComplete)
	buf[1] = 1

	_, err := DecodeEvent(buf)
	if err == nil {
		t.Errorf("Decoded invalid event!")
	}
}

func TestDecodeCommandComplete(t *testing.T) {

	buf, err := hex.DecodeString("0e0401030c00")
	if err != nil {
		t.Fatalf("Invalid hex string for test") // should not happen
	}
	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Fatalf("Invalid event in test") // should not happen
	}
	cc, err := DecodeCommandComplete(ev)
	if err != nil {
		t.Fatalf("Unable to decode command complete")
	}
	if cc.GetCommandOpCode() != CommandReset {
		t.Errorf("Decoded invalid opcode")
	}
	if cc.GetNumHciCommandPackets() != 1 {
		t.Errorf("Decoded invalid num HCI packets")
	}
	if !cc.HasReturnParameters() {
		t.Errorf("Expected CC to have return parameters")
	}
	params := cc.GetReturnParameters()
	if len(params) != 1 || params[0] != 0x00 {
		t.Errorf("Decoded invalid return parameters")
	}
	if cc.GetStatusParameter() != StatusSuccess || cc.GetStatus() != StatusSuccess {
		t.Errorf("Decoded invalid status")
	}
	if cc.GetEventCode() != EventCodeCommandComplete {
		t.Errorf("Received invalid Event Code")
	}

}

func TestCommandCompleteHasNoParamaters(t *testing.T) {
	buf, _ := hex.DecodeString("0e0301030c")
	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Errorf("Unexpected error while parsing event: %v", err)
	}
	cc, err := DecodeCommandComplete(ev)
	if err != nil {
		t.Errorf("Was not able to decode CC with no return parameters")
	}
	if cc.GetCommandOpCode() != CommandReset {
		t.Errorf("Decoded invalid opcode")
	}
	if cc.GetNumHciCommandPackets() != 1 {
		t.Errorf("Decoded invalid num HCI packets")
	}
	if cc.HasReturnParameters() {
		t.Errorf("Expected CC to have no return parameters")
	}
}

func TestDecodeCommandCompleteInvalidOp(t *testing.T) {
	buf, _ := hex.DecodeString("0f0401030c00")
	ev, _ := DecodeEvent(buf)
	_, err := DecodeCommandComplete(ev)
	if err == nil {
		t.Errorf("Decoded command complete with invalid event code")
	}
}
func TestDecodeTooShortCommandComplete(t *testing.T) {

	buf, _ := hex.DecodeString("0e020103")
	ev, _ := DecodeEvent(buf)
	_, err := DecodeCommandComplete(ev)
	if err == nil {
		t.Errorf("Decoded command complete with invalid length")
	}
}

func TestDecodeLeMeta(t *testing.T) {

	buf, _ := hex.DecodeString("3e020201")
	ev, _ := DecodeEvent(buf)

	meta, err := DecodeLeMeta(ev)
	if err != nil {
		t.Fatalf("Unable to decode valid LE Meta event")
	}
	if meta.GetSubeventCode() != SubeventAdvertisingReport {
		t.Errorf("Decoded invalid subevent code")
	}
	param := meta.GetParameters()
	if len(param) != 1 || param[0] != 0x01 {
		t.Errorf("Decoded invalid parameters for LE Meta")
	}
}

func TestDecodeLeMetaInvalidCode(t *testing.T) {
	buf, _ := hex.DecodeString("3f020201")
	ev, _ := DecodeEvent(buf)

	_, err := DecodeLeMeta(ev)
	if err == nil {
		t.Errorf("Decoded LE Meta with invalid opcode")
	}
}

func TestDecodeLeMetaInvalidLength(t *testing.T) {
	buf, _ := hex.DecodeString("3e00")
	ev, _ := DecodeEvent(buf)

	_, err := DecodeLeMeta(ev)
	if err == nil {
		t.Errorf("Decoded LE Meta with invalid length")
	}
}

func TestDecodeCommandStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *CommandStatusEvent
	}{
		{
			"valid",
			"0f0400010604",
			&CommandStatusEvent{Status: StatusSuccess, NumCommands: 1, Command: CommandDisconnect},
		},
		{
			"short",
			"0f03000106",
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf, _ := hex.DecodeString(test.input)
			ev, err := DecodeEvent(buf)
			if err != nil {
				t.Fatalf("Could not parse input as event")
			}
			cs, err := DecodeCommandStatus(ev)
			if test.expected == nil {
				if err == nil {
					t.Errorf("Expected parsing to fail")
				}
			} else if *cs != *test.expected {
				t.Errorf("Expected %#v, received %#v", test.expected, cs)
			} else {
				if cs.GetStatus() != test.expected.Status ||
					cs.GetNumHciCommandPackets() != test.expected.NumCommands ||
					cs.GetCommandOpCode() != test.expected.Command ||
					cs.GetEventCode() != EventCodeCommandStatus {
					t.Errorf("StatusCommand getters returned invalid value")
				}
			}
		})

	}
}

func TestDecodeDisconnectionComplete(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *DisconnectionCompleteEvent
	}{
		{
			"valid",
			"0504002c0013",
			&DisconnectionCompleteEvent{Status: StatusSuccess,
				Handle: ConnectionHandle(0x002c),
				Reason: StatusRemoteUserTerminated},
		},
		{
			"short",
			"0502002c",
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf, _ := hex.DecodeString(test.input)
			ev, err := DecodeEvent(buf)
			if err != nil {
				t.Fatalf("Could not parse input as event")
			}
			cs, err := DecodeDisconnectionComplete(ev)
			if test.expected == nil {
				if err == nil {
					t.Errorf("Expected parsing to fail")
				}
			} else if *cs != *test.expected {
				t.Errorf("Expected %#v, received %#v", test.expected, cs)
			} else {
				if cs.String() != test.expected.String() {
					t.Errorf("Unexpected result from String()")
				}
			}
		})

	}
}

func TestDecodeLeConnectionComplete(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *LeConnectionCompleteEvent
	}{
		{
			"valid",
			"3e1301002d00010155532fe0e77a18000000480001",
			&LeConnectionCompleteEvent{
				Status:              StatusSuccess,
				Handle:              ConnectionHandle(0x002d),
				Role:                LeConnectionRoleSlave,
				Peer:                BtAddress{raw: [6]byte{0x55, 0x53, 0x2f, 0xe0, 0xe7, 0x7a}, Atype: LeRandomAddress},
				interval:            0x18,
				latency:             0x00,
				supervisionTimeout:  0x48,
				masterClockAccuracy: 0x01,
			},
		},
		{
			"valid-public-addr",
			"3e1301002d00010055532fe0e77a18000000480001",
			&LeConnectionCompleteEvent{
				Status:              StatusSuccess,
				Handle:              ConnectionHandle(0x002d),
				Role:                LeConnectionRoleSlave,
				Peer:                BtAddress{raw: [6]byte{0x55, 0x53, 0x2f, 0xe0, 0xe7, 0x7a}, Atype: LePublicAddress},
				interval:            0x18,
				latency:             0x00,
				supervisionTimeout:  0x48,
				masterClockAccuracy: 0x01,
			},
		},
		{
			"valid-connection-master",
			"3e1301002d00000155532fe0e77a18000000480001",
			&LeConnectionCompleteEvent{
				Status:              StatusSuccess,
				Handle:              ConnectionHandle(0x002d),
				Role:                LeConnectionRoleMaster,
				Peer:                BtAddress{raw: [6]byte{0x55, 0x53, 0x2f, 0xe0, 0xe7, 0x7a}, Atype: LeRandomAddress},
				interval:            0x18,
				latency:             0x00,
				supervisionTimeout:  0x48,
				masterClockAccuracy: 0x01,
			},
		},
		{
			"invalid-peer-addrtype",
			"3e1301002d00010355532fe0e77a18000000480001",
			nil,
		},
		{
			"invalid-connection-role",
			"3e1301002d00020155532fe0e77a18000000480001",
			nil,
		},
		{
			"invalid-short",
			"3e1001002d00010155532fe0e77a18000000",
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf, _ := hex.DecodeString(test.input)
			ev, err := DecodeEvent(buf)
			if err != nil {
				t.Fatalf("Could not parse input as event")
			}
			meta, err := DecodeLeMeta(ev)
			if err != nil {
				t.Fatalf("Could not parse input as LE meta event")
			}
			cc, err := DecodeLeConnectionComplete(meta)
			if test.expected == nil {
				if err == nil {
					t.Errorf("Expected parsing to fail")
				}
			} else if *cc != *test.expected {
				t.Errorf("Expected %#v, received %#v", test.expected, cc)
			} else {
				if cc.String() != test.expected.String() {
					t.Errorf("Unrxpected result from String()")
				}
			}
		})

	}
}
