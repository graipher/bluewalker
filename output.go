package main

import (
	"container/list"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/logging"
)

// output interface defines type which can be used to output data
type output interface {
	write(data string) error
	isHumanReadable() bool
	Close()
	writeAsJSON(data interface{}) error
}

type outputImpl struct {
	wr            io.Writer
	cl            io.Closer
	humanReadable bool
}

func (out *outputImpl) write(data string) error {
	_, err := out.wr.Write([]byte(data))
	return err
}

func (out *outputImpl) isHumanReadable() bool {
	return out.humanReadable
}

func (out *outputImpl) Close() {
	if out.cl != nil {
		out.cl.Close()
	}
}

// writeAsJSON marshals the given data into JSON and prints
// the resulting string(s) using this output. If output is marked
// as HumanReadble, the JSON is indented and printed so that it
// is easy to read. If ouput is not HumanReadable, the json is
// printed as a single -line where data is terminated by newline ('\n')
func (out *outputImpl) writeAsJSON(data interface{}) error {
	var err error
	var jdata []byte
	if out.isHumanReadable() {
		jdata, err = json.MarshalIndent(data, "", "\t")
	} else {
		jdata, err = json.Marshal(data)
		if err == nil {
			jdata = []byte(string(jdata) + "\n")
		}
	}
	if err == nil {
		return out.write(string(jdata))
	}
	return fmt.Errorf("unable to marshal %q as JSON: %v", data, err)
}

// outputForSocket returns output which connects to given UNIX socket
// and uses the socket for output
func outputForSocket(path string) (output, error) {
	unixConn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}
	return &outputImpl{wr: unixConn, cl: unixConn, humanReadable: false}, nil
}

// outputForFile returns output which writes the data to file with
// given path. If path is "-", then data is written to stdout
func outputForFile(path string) (output, error) {
	if path == "-" {
		return &outputImpl{wr: os.Stdout, humanReadable: false}, nil
	}
	fs, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &outputImpl{wr: fs, cl: fs, humanReadable: false}, nil
}

// defaultOutput returns default output to use.
func defaultOutput() output {
	return &outputImpl{wr: os.Stdout, humanReadable: true}
}

type sockListener struct {
	mux   sync.Mutex
	wg    sync.WaitGroup
	conns list.List
	l     net.Listener
}

func (s *sockListener) init(path string) error {
	l, err := net.Listen("unix", path)
	if err != nil {
		return fmt.Errorf("unable to start listener: %v", err)
	}
	logging.Debug.Printf("Started listener on %s", path)
	s.l = l
	s.wg.Add(1)
	go s.loop()
	return nil
}

func (s *sockListener) loop() {
	for {
		c, err := s.l.Accept()
		if err != nil {
			logging.Debug.Printf("Error in accept: %v", err)
			break
		}
		s.mux.Lock()
		// add the new connection to list
		s.conns.PushBack(c)
		logging.Debug.Printf("New connection accepted, %d active connections", s.conns.Len())
		s.mux.Unlock()
	}
	s.l.Close()
	logging.Trace.Printf("sockListener loop terminating")
	s.wg.Done()
}

func (s *sockListener) Close() error {
	logging.Debug.Printf("Closing sockListener")
	s.l.Close()
	s.mux.Lock()
	var next *list.Element
	// Close all connections and remove them from the list
	for e := s.conns.Front(); e != nil; e = next {
		next = e.Next()
		val := s.conns.Remove(e)
		if conn, ok := val.(net.Conn); ok {
			conn.Close()
		} else {
			panic("Unexpected value in connection list")
		}
	}
	s.mux.Unlock()
	// wait for the loop to terminate
	s.wg.Wait()
	logging.Debug.Printf("sockListener closed")
	return nil
}

func (s *sockListener) Write(p []byte) (int, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	var next *list.Element
	for e := s.conns.Front(); e != nil; e = next {
		next = e.Next()
		conn, ok := e.Value.(net.Conn)
		if !ok {
			panic("Unexpected value in connection list")
		}
		logging.Trace.Print("writing to client")
		_, err := conn.Write(p)
		if err != nil {
			conn.Close()
			s.conns.Remove(e)
			logging.Debug.Printf("Error %v while writing to client, dropping connection, %d connections left", err, s.conns.Len())
		}
	}
	return len(p), nil
}

// outputForListeningSocket returns output which starts listening UNIX socket on given
// path and writes output to all connecting clients.
func outputForListeningSocket(path string) (output, error) {

	sockl := &sockListener{}
	if err := sockl.init(path); err != nil {
		return nil, err
	}
	return &outputImpl{humanReadable: false, wr: sockl, cl: sockl}, nil
}

func formatAddress(addr hci.BtAddress) string {
	addrstr := addr.String()
	if addr.Atype == hci.LeRandomAddress {
		addrstr += ",random ("
		if addr.IsNonResolvable() {
			addrstr += "non-resolvable private"
		} else if addr.IsResolvable() {
			addrstr += "resolvable private"
		} else if addr.IsStatic() {
			addrstr += "static"
		} else {
			addrstr += "??"
		}
		addrstr += ")"

	}
	return addrstr
}

func checkFlag(flags byte, flag int) bool {
	return (int(flags) & flag) == flag
}

//Description for each flag in AD Flags bitmask
var flagNames = []struct {
	flag int
	name string
}{
	{hci.AdFlagLimitedDisc, "LE Limited Discoverable"},
	{hci.AdFlagGeneralDisc, "LE General Discoverable"},
	{hci.AdFlagNoBrEdr, "BR/EDR not supported"},
	{hci.AdFlagLeBrEdrController, "LE & BR/EDR (controller)"},
	{hci.AdFlagLeBrEdrHost, "LE & BR/EDR (host)"},
}

func decodeAdFlags(flags []byte) string {
	if len(flags) != 1 {
		return "(Invalid)"
	}
	str := strings.Builder{}
	str.WriteString("[")
	for i := 7; i >= 0; i-- {
		if checkFlag(flags[0], (0x01 << uint8(i))) {
			str.WriteString("1")
		} else {
			str.WriteString("0")
		}
	}
	str.WriteString("]")
	if flags[0] == 0 {
		return str.String()
	}
	str.WriteString("(")
	hasFlag := false
	for _, fl := range flagNames {
		if checkFlag(flags[0], fl.flag) {
			if hasFlag {
				str.WriteString(",")
			}
			str.WriteString(fl.name)
			hasFlag = true
		}
	}
	str.WriteString(")")
	return str.String()
}

func decodeDeviceAddress(data []byte) string {
	if len(data) != 7 {
		return "(invalid)"
	}
	addr := hci.ToBtAddress(data[1:])
	if data[0]&0x01 == 0x01 {
		addr.Atype = hci.LeRandomAddress
	}
	return formatAddress(addr)
}

func decodeServiceData(data []byte) string {
	// Service Data starts with 16-bit UUID followed by service data
	// Supplement to Bluetooth Core Specification ch 1.11
	if len(data) < 2 {
		return fmt.Sprintf("0x%x", data)
	}
	sb := strings.Builder{}
	uuid := binary.LittleEndian.Uint16(data[0:2])
	sb.WriteString(fmt.Sprintf("UUID: 0x%.4x", uuid))
	switch uuid {
	case 0xfd6f:
		// Google & Apple Exposure Notification for COVID-19
		// https://covid19-static.cdn-apple.com/applications/covid19/current/static/contact-tracing/pdf/ExposureNotification-BluetoothSpecificationv1.2.pdf?1
		sb.WriteString(", Exposure Notification")
		if len(data) < 22 {
			sb.WriteString(fmt.Sprintf("\n\t\t(invalid data) 0x%x", data[2:]))
		} else {
			sb.WriteString(fmt.Sprintf("\n\t\tProximity Identifier: 0x%x, Encrypted Metadata: 0x%x", data[2:18], data[18:]))
		}
	default:
		if len(data) > 2 {
			sb.WriteString(fmt.Sprintf(", Data: 0x%x", data[2:]))
		}
	}
	return sb.String()
}

func decodeVendorSpecificData(data []byte) string {
	if len(data) < 2 {
		return fmt.Sprintf("0x%x", data)
	}
	sb := strings.Builder{}
	companyID := binary.LittleEndian.Uint16(data[0:2])
	if name, found := btCompanyIdentifiers[int(companyID)]; found {
		sb.WriteString(fmt.Sprintf("%s (0x%.4x)", name, companyID))
	} else {
		sb.WriteString(fmt.Sprintf("Unknown company ID 0x%.4x", companyID))
	}
	if len(data) > 2 {
		sb.WriteString(fmt.Sprintf(", Data: 0x%x", data[2:]))
	}
	return sb.String()
}

func decodeDeviceName(data []byte) string {
	return fmt.Sprintf("Name: \"%s\"", string(data))
}

func defaultDecode(data []byte) string {
	return fmt.Sprintf("Data: 0x%x", data)
}

type adDataDecodingFunc func([]byte) string

var adDataDecoders = map[hci.AdType]adDataDecodingFunc{
	hci.AdFlags:                decodeAdFlags,
	hci.AdDeviceAddress:        decodeDeviceAddress,
	hci.AdServiceData:          decodeServiceData,
	hci.AdManufacturerSpecific: decodeVendorSpecificData,
	hci.AdCompleteLocalName:    decodeDeviceName,
	hci.AdShortenedLocalName:   decodeDeviceName,
}

func decodeAdStructure(ad *hci.AdStructure) string {

	decFunc, ok := adDataDecoders[ad.Typ]
	if !ok {
		decFunc = defaultDecode
	}
	decoded := decFunc(ad.Data)
	return fmt.Sprintf("%s: %s", ad.Typ, decoded)
}
