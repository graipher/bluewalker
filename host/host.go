package host

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
)

// exec is used when HCI commands need to be sent to controller
type exec struct {
	// command to execute
	cmd *hci.CommandPacket
	// this function is called when execution has completed
	complete func(*hci.CommandCompleteEvent)
	// this is called if command can not be sent
	fail func(error)
}

//ScanReport contains information about a device found on scanning
//See Bluetooth 5.0, vol 2, part E, ch 7.7.65.2
type ScanReport struct {
	Type    hci.AdvType
	Address hci.BtAddress
	Rssi    int8
	Data    []*hci.AdStructure
}

// Host implements the host side of Bluetooth Host - Controller interface
type Host struct {
	// Transport we are using for Host - Controller communication
	tr hci.Transport
	// WaitGroup to signal when host event receiver has stopped
	wg sync.WaitGroup
	// Mutex synchronizing access to host internals
	mux sync.Mutex
	// Event channel
	// Event receiver pushes received events to this channel
	// Event handler will read events from this channel
	evt chan []byte
	// Commands for executor
	// Command executor will read commands to execute from this channel
	cmd chan *exec
	// CommandComplete events to executor
	// Command executor will read this channel when it is waiting command to complete
	cc chan *hci.CommandCompleteEvent
	// Channel used to inform about received scanning data
	// StartScanning() will return this channel and user will receive ScanReports
	// through it.
	ad chan *ScanReport
	// Filters for incoming advertising reports
	filters adfilters
	// flag indicating that host is closing.
	// access needs to be protected using mux as event receiving goroutine
	// is using this to indicate it should stop.
	closing bool
}

// New returns new host which uses given transport for communicating
// with controller
func New(tr hci.Transport) *Host {

	host := new(Host)
	host.tr = tr
	host.filters = filterList()
	host.evt = make(chan []byte, 2)
	host.cmd = make(chan *exec)
	host.cc = make(chan *hci.CommandCompleteEvent)
	host.ad = make(chan *ScanReport, 5)
	host.closing = false

	return host
}

// isClosing returns true if closing flag is set.
// This is safe way to check the status of closing flag
func (h *Host) isClosing() bool {
	var ret bool
	h.mux.Lock()
	ret = h.closing
	h.mux.Unlock()
	return ret
}

// eventReceiver is run on its own goroutine and it uses the transport to
// receive events from Controller. No other goroutine should read from
// transport. The received events are written to 'evt' channel in host
// this method will return when isClosing() returns true
func (h *Host) eventReceiver() {

	defer h.wg.Done()
	for !h.isClosing() {
		buf, err := h.tr.Read()
		if err != nil {
			if !os.IsTimeout(err) {
				log.Printf("Error while reading: %s", err.Error())
			}
			continue
		}
		if len(buf) == 0 {
			continue
		}
		if buf[0] != hci.HciEventPacket {
			log.Printf("Received unexpected packet from controller")
			continue
		}
		log.Printf("Received %d bytes of event", len(buf)-1)
		h.evt <- buf[1:]
	}
	log.Printf("EventReceiver closing")
}

// eventHandler is run on its own goroutine and it reads the events
// from 'evt' channel. Event handler is responsible for routing the events
// to correct channel or calling proper handlers for the events.
// This method returns when 'evt' channel is closed
func (h *Host) eventHandler() {

	for buf := range h.evt {
		evt, err := hci.DecodeEvent(buf)
		if err != nil {
			log.Printf("Received invalid event: %s", err.Error())
			continue
		}
		log.Printf("Received %s event", evt.Code.String())
		switch evt.Code {
		case hci.EventCodeCommandComplete:
			cc, err := hci.DecodeCommandComplete(evt)
			if err != nil {
				log.Printf("Received invalid Command Complete event: %s", err.Error())
				continue
			}
			h.cc <- cc
		case hci.EventCodeLeMeta:
			meta, err := hci.DecodeLeMeta(evt)
			if err != nil {
				log.Printf("Received invalid LE Meta event: %s", err.Error())
				continue
			}
			if meta.GetSubeventCode() == hci.SubeventAdvertisingReport {
				if err := handleAdvertisingReport(h.ad, h.filters, meta.GetParameters()); err != nil {
					log.Printf("Error while parsing Advertising report: %s", err.Error())
				}
			}
		default:
			log.Printf("Received unexpected event %s", evt.Code.String())
		}
	}
	log.Printf("Event handler stopping")
}

// executor is run on its own goroutine and it is responsible for for
// executing HCI commands. The commands to execute are read from 'cmd'
// channel and written to controller. Then Command Complete event is waited
// and the status of command execution is communicated back to one requesting
// the command to be sent.
// This method returns once the 'cmd' channel is closed.
func (h *Host) executor() {

	// number of commands we can execute
	numCommands := 1

	for e := range h.cmd {
		log.Printf("Executing command %s", e.cmd.OpCode.String())
		if numCommands == 0 {
			e.fail(fmt.Errorf("Flow control error"))
			continue
		}
		if err := h.tr.Write(e.cmd.Encode()); err != nil {
			e.fail(fmt.Errorf("Can not write: %s", err.Error()))
			continue
		}
		numCommands--
		completed := false
		// we need to wait until the command has completed before starting
		// with new command.
		for !completed || numCommands == 0 {
			select {
			case cc := <-h.cc:
				numCommands = int(cc.GetNumHciCommandPackets())
				log.Printf("Number of HCI packets increased to %d", numCommands)
				if !completed && cc.GetCommandOpcode() == e.cmd.OpCode {
					completed = true
					e.complete(cc)
				} else if !completed {
					log.Printf("Received unexepcted cc for %s ", cc.GetCommandOpcode().String())
				}
			}
		}
	}
}

// executeStatusCommand executes single HCI command which expects to have
// 'status' parameter in the following CommandComplete event. This status
// is checked and error is returned command execution failed.
func (h *Host) executeStatusCommand(cmd *hci.CommandPacket) error {

	var wg sync.WaitGroup

	var err error

	e := new(exec)
	e.cmd = cmd
	e.complete = func(cc *hci.CommandCompleteEvent) {
		if cc.GetStatusParameter() != hci.StatusSuccess {
			err = fmt.Errorf("Command Failed: %s", cc.GetStatusParameter().String())
		}
		wg.Done()
	}
	e.fail = func(er error) {
		err = fmt.Errorf("Command execution failed: %s", er.Error())
		wg.Done()
	}
	wg.Add(1)
	h.cmd <- e
	wg.Wait()
	return err
}

// initializeController sends the necessary commands to initialize
// communication with Controller. The controller is reset first.
func (h *Host) initializeController() error {

	commands := make([]*hci.CommandPacket, 4)

	// Reset
	commands[0] = &hci.CommandPacket{OpCode: hci.CommandReset}

	// LE Host Supported command
	commands[1] = &hci.CommandPacket{OpCode: hci.CommandWriteLeHostSupported}
	params := make([]byte, 2)
	params[0] = 0x01 // LE Supported Host enabeled
	params[1] = 0x00 // Simultaneous LE Host parameter
	commands[1].Parameters(params)

	// Set Event mask
	commands[2] = &hci.CommandPacket{OpCode: hci.CommandSetEventMask}
	params = make([]byte, 8)
	// Default mask, all events, we might want to optimize this
	binary.LittleEndian.PutUint64(params, 0x3fffffffffffffff)
	commands[2].Parameters(params)

	// Set LE event mask
	commands[3] = &hci.CommandPacket{OpCode: hci.CommandLeSetEventMask}
	params = make([]byte, 8)
	binary.LittleEndian.PutUint64(params, 0x000000000000001f)
	commands[3].Parameters(params)

	for _, cmd := range commands {
		if err := h.executeStatusCommand(cmd); err != nil {
			return err
		}
	}
	return nil
}

// Init initializes the host
func (h *Host) Init() error {

	log.Printf("Initializing Host")

	h.wg.Add(1)
	// Start the event and command handlng goroutines
	go h.eventReceiver()
	go h.eventHandler()
	go h.executor()

	log.Printf("Resetting...")

	if err := h.initializeController(); err != nil {
		// XXX: Deinitialize
		return fmt.Errorf("Unable to initialize controller: %s", err.Error())
	}

	return nil
}

//StartScanning will start scanning for Bluetooth LE Advertisements
//Active defines if active or passive scanning should be done
//The returned channel can be used to receive all scan reports matching _any_
//of the filters on list. The returned cannel should _not_ be closed.
func (h *Host) StartScanning(active bool, filters []filter.AdFilter) (chan *ScanReport, error) {

	if filters != nil && len(filters) > 0 {
		for _, f := range filters {
			h.filters = addFilter(h.filters, f)
		}
	}

	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetScanParameters}
	// See Bluetooth v5.0, vol 2, part E, ch 7.8.10
	parameters := make([]byte, 7)
	if active {
		// active scanning
		parameters[0] = 0x01
	}
	// Scan interval
	binary.LittleEndian.PutUint16(parameters[1:], 0x0010)
	// Scan window
	binary.LittleEndian.PutUint16(parameters[3:], 0x0010)
	// Own address type, public
	parameters[5] = 0x00
	// Filter policy
	parameters[6] = 0x00
	cmd.Parameters(parameters)

	log.Printf("Setting scan parameters")
	if err := h.executeStatusCommand(&cmd); err != nil {
		return nil, fmt.Errorf("Unable to set Scan Parameters: %s", err.Error())
	}

	cmd = hci.CommandPacket{OpCode: hci.CommandLeSetScanEnable}
	// See Bluetooth v5.0, vol 2, part E, ch 7.8.11
	parameters = make([]byte, 2)
	// Scan enable
	parameters[0] = 0x01
	// Filter duplicates
	parameters[1] = 0x00
	cmd.Parameters(parameters)

	log.Printf("Starting scan")
	if err := h.executeStatusCommand(&cmd); err != nil {
		return nil, fmt.Errorf("Unable to start scanning: %s", err.Error())
	}
	return h.ad, nil
}

//StopScanning stops scanning for advertising LE devices
func (h *Host) StopScanning() error {

	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetScanEnable}
	parameters := make([]byte, 2)
	// Scan enable
	parameters[0] = 0x00
	// filter duplicates
	parameters[1] = 0x00
	cmd.Parameters(parameters)
	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("Unable to stop scanning: %s", err.Error())
	}
	return nil
}

// Deinit will deinitialize Host
func (h *Host) Deinit() {
	log.Printf("Deinitializing host")
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}
	// not checking the return value since there is not much we can do on error
	h.executeStatusCommand(&cmd)
	h.mux.Lock()
	h.closing = true
	h.mux.Unlock()

	h.tr.Close()
	h.wg.Wait()
	// Now the eventReceiver has stopped. Rest should stop when we close
	// the channels
	close(h.evt)
	close(h.cc)
	close(h.ad)
	log.Printf("Deinitialization done")
}
