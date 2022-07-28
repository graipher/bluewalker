package hci

import "fmt"

// ErrorCode defines the error codes returned by controller
// See Bluetooth 5.0, vol 2, part D
type ErrorCode byte

//Defined Error codes
const (
	StatusSuccess                   ErrorCode = 0x00
	StatusUnknownCommand            ErrorCode = 0x01
	StatusAuthenticationFailure     ErrorCode = 0x05
	StatusCommandDisallowed         ErrorCode = 0x0c
	StatusInvalidParams             ErrorCode = 0x12
	StatusRemoteUserTerminated      ErrorCode = 0x13
	StatusRemoteTerminatedResources ErrorCode = 0x14
	StatusRemoteTerminatedPowerOff  ErrorCode = 0x15
	StatusLocalTerminated           ErrorCode = 0x16
	StatusUnsupportedRemoteFeature  ErrorCode = 0x1a
)

func (e ErrorCode) String() string {
	switch e {
	case StatusSuccess:
		return "Success"
	case StatusUnknownCommand:
		return "Unknown HCI command"
	case StatusAuthenticationFailure:
		return "Authentication failure"
	case StatusCommandDisallowed:
		return "Command disallowed"
	case StatusInvalidParams:
		return "Invalid command parameters"
	case StatusRemoteUserTerminated:
		return "Remote user terminated connection"
	case StatusRemoteTerminatedResources:
		return "Remote user terminated connection due to low resources"
	case StatusRemoteTerminatedPowerOff:
		return "Remote user terminated connection due to power off"
	case StatusLocalTerminated:
		return "Connection terminated by local host"
	default:
		return fmt.Sprintf("Unknown error: 0x%.2x", byte(e))
	}
}
