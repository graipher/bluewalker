package hci

// Transport allows sending and receiving raw HCI packets
// Use hci.Raw() to create transport
type Transport interface {
	// Close closes the transport
	Close()
	Read() ([]byte, error)
	Write(buffer []byte) error
}

const (
	hciCommandPacket byte = 0x01
	hciACLPacket     byte = 0x02
	// HciEventPacket indicates that data from transport contains HCI Event
	HciEventPacket byte = 0x04
)
