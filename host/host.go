package host

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/logging"
)

// how long to wait for command to execute
var cmdExecutionTimeout = time.Duration(30 * time.Second)

// Error for command execution timeout
var errExecutionTimeout = fmt.Errorf("command execution timed out")

// exec is used when HCI commands need to be sent to controller
type exec struct {
	// command to execute
	cmd *hci.CommandPacket
	// this function is called when command Complete event is received
	complete func(*hci.CommandCompleteEvent)
	// this function is called when Command Status event is received
	status func(*hci.CommandStatusEvent)
	// this is called if command can not be sent
	fail func(error)
	// Error from status event, if any
	err error
}

// setError will set error status for exec
func (e *exec) setError(err error) {
	e.err = err
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
	cc chan hci.StatusEvent
	// Channel used to inform about received scanning data
	// StartScanning() will return this channel and user will receive ScanReports
	// through it.
	ad chan *ScanReport
	// Filters for incoming advertising reports
	filters filter.AdFilter
	// flag indicating that host is closing.
	// access needs to be protected using mux as event receiving goroutine
	// is using this to indicate it should stop.
	closing bool

	// FIXME: dummy connections store
	connections map[hci.ConnectionHandle]hci.BtAddress
	// FIXME: channel for indications to user
	Indications chan Indication
}

// New returns new host which uses given transport for communicating
// with controller
func New(tr hci.Transport) *Host {

	host := new(Host)
	host.tr = tr
	host.filters = nil
	host.evt = make(chan []byte, 2)
	host.cmd = make(chan *exec)
	host.cc = make(chan hci.StatusEvent)
	host.ad = make(chan *ScanReport, 5)
	host.closing = false
	host.Indications = make(chan Indication)

	host.connections = make(map[hci.ConnectionHandle]hci.BtAddress)

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
			var again hci.ErrReadAgain
			if !errors.As(err, &again) {
				logging.Warning.Printf("Error while reading: %v", err)
			}
			continue
		}
		if len(buf) == 0 {
			continue
		}
		if buf[0] != hci.HciEventPacket {
			logging.Debug.Printf("Received unexpected packet from controller")
			continue
		}
		logging.Debug.Printf("Received %d bytes of event", len(buf)-1)
		h.evt <- buf[1:]
	}
	logging.Trace.Printf("EventReceiver closing")
}

// eventHandler is run on its own goroutine and it reads the events
// from 'evt' channel. Event handler is responsible for routing the events
// to correct channel or calling proper handlers for the events.
// This method returns when 'evt' channel is closed
func (h *Host) eventHandler() {

	for buf := range h.evt {
		evt, err := hci.DecodeEvent(buf)
		if err != nil {
			logging.Warning.Printf("Received invalid event: %s", err.Error())
			continue
		}
		logging.Debug.Printf("Received %s event", evt.Code.String())
		switch evt.Code {
		case hci.EventCodeCommandComplete:
			cc, err := hci.DecodeCommandComplete(evt)
			if err != nil {
				logging.Warning.Printf("Received invalid Command Complete event: %s", err.Error())
				continue
			}
			h.cc <- cc
		case hci.EventCodeCommandStatus:
			cs, err := hci.DecodeCommandStatus(evt)
			if err != nil {
				logging.Warning.Printf("Received invalid Command Status event: %s", err.Error())
				continue
			}
			h.cc <- cs
		case hci.EventCodeDisconnectionComplete:
			dc, err := hci.DecodeDisconnectionComplete(evt)
			if err != nil {
				logging.Warning.Printf("Received invalid Disconnection Complete: %s", err.Error())
				continue
			}
			logging.Debug.Printf("Received Disconection Complete: %s", dc.String())
			if dc.Status != hci.StatusSuccess {
				logging.Warning.Printf("Disconnection did not succeed: %s", dc.Status.String())
			} else {
				p, found := h.connections[dc.Handle]
				if !found {
					logging.Warning.Printf("Could not find connection with handle %s", dc.Handle.String())
					continue
				}
				logging.Debug.Printf("Peer %s disconnected", p)
				delete(h.connections, dc.Handle)
				h.Indications <- Indication{Type: DisconnectionIndication, Peer: p, Handle: dc.Handle}
			}
		case hci.EventCodeLeMeta:
			meta, err := hci.DecodeLeMeta(evt)
			if err != nil {
				logging.Warning.Printf("Received invalid LE Meta event: %s", err.Error())
				continue
			}
			if meta.GetSubeventCode() == hci.SubeventAdvertisingReport {
				if err := handleAdvertisingReport(h.ad, h.filters, meta.GetParameters()); err != nil {
					logging.Warning.Printf("Error while parsing Advertising report: %s", err.Error())
				}
			} else if meta.GetSubeventCode() == hci.SubeventLeConnectionComplete {
				ev, err := hci.DecodeLeConnectionComplete(meta)
				if err != nil {
					logging.Warning.Printf("Could not parse LE Connection Complete event: %s", err.Error())
				}
				logging.Debug.Printf("Connection Complete: %s", ev)
				h.connections[ev.Handle] = ev.Peer
				h.Indications <- Indication{Type: ConnectionIndication, Peer: ev.Peer, Handle: ev.Handle}
			}
		default:
			logging.Debug.Printf("Received unexpected event %s", evt.Code.String())
		}
	}
	logging.Trace.Printf("Event handler stopping")
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
	execTimer := time.NewTimer(cmdExecutionTimeout)

	for e := range h.cmd {
		logging.Debug.Printf("Executing command %s", e.cmd.OpCode.String())
		if numCommands == 0 {
			e.fail(fmt.Errorf("flow control error"))
			continue
		}
		if err := h.tr.Write(e.cmd.Encode()); err != nil {
			e.fail(fmt.Errorf("can not write: %s", err.Error()))
			continue
		}

		if !execTimer.Stop() {
			<-execTimer.C
		}
		execTimer.Reset(cmdExecutionTimeout)

		numCommands--
		completed := false
		// we need to wait until the command has completed before starting
		// with new command.
		for !completed || numCommands == 0 {
			select {
			case cc := <-h.cc:
				if cc == nil {
					// channel is closed and we should thus be breaking out
					break
				}
				numCommands = int(cc.GetNumHciCommandPackets())
				logging.Debug.Printf("Number of HCI packets increased to %d", numCommands)
				if !completed && cc.GetCommandOpCode() == e.cmd.OpCode {
					completed = true
					switch cc.GetEventCode() {
					case hci.EventCodeCommandComplete:
						e.complete(cc.(*hci.CommandCompleteEvent))
					case hci.EventCodeCommandStatus:
						e.status(cc.(*hci.CommandStatusEvent))
					}
				} else if !completed {
					logging.Warning.Printf("Received unexepcted cc for %s ", cc.GetCommandOpCode().String())
				}
			case <-execTimer.C:
				if !completed {
					logging.Warning.Printf("Command %s execution timed out", e.cmd.OpCode.String())
					completed = true
					e.fail(errExecutionTimeout)
					// The command complete has likely been lost somewhere
					// increase the numCommands to allow sending new command
					// otherwise we'll be stuck here looping and never
					numCommands = 1
				}
			}
		}
	}
}

// CommandExecutionError is returned when HCI command could not be
// executed.
type CommandExecutionError struct {
	status hci.ErrorCode     // Status code for the failure
	op     hci.CommandOpCode // command that failed to execute
}

func (e *CommandExecutionError) Error() string {
	return fmt.Sprintf("Command %s execution failed: %s ", e.op.String(), e.status.String())
}

// ErrorCode returns the error code indicating reason for command
// execution failure
func (e *CommandExecutionError) ErrorCode() hci.ErrorCode {
	return e.status
}

// executeStatusParamCommand executes single HCI command which expects to have
// 'status' parameter in the following CommandComplete event. This status
// is checked and error is returned command execution failed.
func (h *Host) executeStatusParamCommand(cmd *hci.CommandPacket) error {

	var wg sync.WaitGroup
	e := new(exec)
	e.cmd = cmd
	e.complete = func(cc *hci.CommandCompleteEvent) {
		if cc.HasReturnParameters() {
			if cc.GetStatusParameter() != hci.StatusSuccess {
				e.setError(&CommandExecutionError{status: cc.GetStatusParameter(), op: cmd.OpCode})
			}
		} else {
			e.setError(fmt.Errorf("received unexpected Command Complete with no status"))
		}
		wg.Done()
	}
	e.fail = func(er error) {
		e.setError(fmt.Errorf("command execution failed: %s", er.Error()))
		wg.Done()
	}
	wg.Add(1)
	h.cmd <- e
	wg.Wait()
	return e.err
}

//executeStatusCommand executes single HCI command which expectes Command Status
//event as return event.
func (h *Host) executeStatusCommand(cmd *hci.CommandPacket) error {

	var wg sync.WaitGroup
	e := new(exec)
	e.cmd = cmd
	e.status = func(cs *hci.CommandStatusEvent) {
		if cs.GetStatus() != hci.StatusSuccess {
			e.setError(&CommandExecutionError{status: cs.GetStatus(), op: cmd.OpCode})
		}
		wg.Done()
	}
	e.fail = func(er error) {
		e.setError(fmt.Errorf("command execution failed: %s", er.Error()))
		wg.Done()
	}
	wg.Add(1)
	h.cmd <- e
	wg.Wait()
	return e.err

}

// initializeController sends the necessary commands to initialize
// communication with Controller. The controller is reset first.
func (h *Host) initializeController() error {

	commands := make([]hci.CommandPacket, 4)

	// Reset
	commands[0] = hci.CommandPacket{OpCode: hci.CommandReset}

	// LE Host Supported command
	commands[1] = hci.NewCommandBuilder(hci.CommandWriteLeHostSupported, 2).
		// Le supported host enabeled
		AddByte(0x01).
		// Simultaneous LE Host parameter
		AddByte(0x00).Command()

	// Set Event mask
	commands[2] = hci.NewCommandBuilder(hci.CommandSetEventMask, 8).
		// Default mask, all events, we might want to optimize this
		AddUint64(0x3fffffffffffffff).Command()

	// Set LE event mask
	commands[3] = hci.NewCommandBuilder(hci.CommandLeSetEventMask, 8).
		AddUint64(0x000000000000001f).Command()

	for _, cmd := range commands {
		if err := h.executeStatusParamCommand(&cmd); err != nil {
			return err
		}
	}
	return nil
}

// Init initializes the host
func (h *Host) Init() error {

	logging.Debug.Printf("Initializing Host")

	h.wg.Add(1)
	// Start the event and command handlng goroutines
	go h.eventReceiver()
	go h.eventHandler()
	go h.executor()

	logging.Debug.Printf("Resetting...")

	if err := h.initializeController(); err != nil {
		// XXX: Deinitialize
		var execErr *CommandExecutionError
		if errors.As(err, &execErr) {
			return fmt.Errorf("unable to initialize controller: %w", execErr)
		}
		return fmt.Errorf("unable to initialize controller: %s", err.Error())
	}

	return nil
}

//StartScanning will start scanning for Bluetooth LE Advertisements
//Active defines if active or passive scanning should be done
//The returned channel can be used to receive all scan reports matching _any_
//of the filters on list. The returned cannel should _not_ be closed.
func (h *Host) StartScanning(active bool, filters []filter.AdFilter) (chan *ScanReport, error) {

	if len(filters) > 0 {
		h.filters = filter.All(filters)
	}
	// // See Bluetooth v5.0, vol 2, part E, ch 7.8.10
	bld := hci.NewCommandBuilder(hci.CommandLeSetScanParameters, 7)
	if active {
		// active scanning
		bld.AddByte(0x01)
	} else {
		// passive scanning
		bld.AddByte(0x00)
	}
	// Scan interval
	cmd := bld.AddUint16(0x0010).
		// Scan window
		AddUint16(0x0010).
		// Own address type, public
		AddByte(0x00).
		// Filter policy
		AddByte(0x00).Command()

	logging.Debug.Printf("Setting scan parameters")
	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return nil, fmt.Errorf("unable to set Scan Parameters: %s", err.Error())
	}

	// // See Bluetooth v5.0, vol 2, part E, ch 7.8.11
	cmd = hci.NewCommandBuilder(hci.CommandLeSetScanEnable, 2).
		// Scan enable
		AddByte(0x01).
		// Filter duplicates
		AddByte(0x00).Command()

	logging.Debug.Printf("Starting scan")
	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return nil, fmt.Errorf("unable to start scanning: %s", err.Error())
	}
	return h.ad, nil
}

//StopScanning stops scanning for advertising LE devices
func (h *Host) StopScanning() error {

	cmd := hci.NewCommandBuilder(hci.CommandLeSetScanEnable, 2).
		// Scan enable
		AddByte(0x00).
		// filter duplicates
		AddByte(0x00).
		Command()
	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return fmt.Errorf("unable to stop scanning: %s", err.Error())
	}
	return nil
}

//SetAdvertisingParams set advertising params to the controller.
// hci.DefaultAdvParameters() can be used to get default set of parameters.
func (h *Host) SetAdvertisingParams(advParams hci.AdvertisingParameters) error {

	bld := hci.NewCommandBuilder(hci.CommandLeSetAdvParameters, 15).
		// Min advertising interval
		AddUint16(advParams.IntervalMin).
		// Max advertising interval
		AddUint16(advParams.IntervalMax).
		// advertising type
		AddByte(byte(advParams.Type)).
		// Own Address Type
		AddByte(byte(advParams.OwnAddrType))

	// Peer address type
	if advParams.PeerAddress.Atype == hci.LePublicAddress {
		bld.AddByte(0x00)
	} else {
		bld.AddByte(0x01)
	}

	// peer address
	cmd := bld.AddBtAddress(advParams.PeerAddress).
		// Channel Map
		AddByte(byte(advParams.ChannelMap)).
		// Filter policy
		AddByte(byte(advParams.FilterPolicy)).Command()

	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return fmt.Errorf("unable to set advertising parameters: %s", err.Error())
	}
	return nil
}

func putAdvData(bld *hci.CommandBuilder, datas []*hci.AdStructure) (int, error) {
	totalLength := 0
	for _, ad := range datas {
		totalLength += ad.EncodedLength()
		if totalLength > 32 { // FIXME: constant
			return totalLength, fmt.Errorf("too many bytes of advertising data")
		}
		bld.AddEncodeable(ad)
	}
	return totalLength, nil
}

func (h *Host) setAdvData(data []*hci.AdStructure, scanResp bool) error {
	var opcode hci.CommandOpCode
	if scanResp {
		opcode = hci.CommandLeSetScanResponse
	} else {
		opcode = hci.CommandLeSetAdvData
	}
	bld := hci.NewCommandBuilder(opcode, 32).
		// total length, we'll fill this later
		AddByte(0)

	len, err := putAdvData(bld, data)
	if err != nil {
		return err
	}
	cmd := bld.PutByte(0, byte(len)).Command()

	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return fmt.Errorf("unable set advertising data: %s", err.Error())
	}

	return nil
}

//SetAdvertisingData sets the advertising data that will be sent
//when advertising is enabled
func (h *Host) SetAdvertisingData(data []*hci.AdStructure) error {
	return h.setAdvData(data, false)
}

//SetScanResponse sets the scan response data which will be sent
//when advertising and the mode allows it.
func (h *Host) SetScanResponse(data []*hci.AdStructure) error {
	return h.setAdvData(data, true)
}

//StartAdvertising directs the controller to start sending advertisments
func (h *Host) StartAdvertising() error {

	cmd := hci.NewCommandBuilder(hci.CommandLeSetAdvEnable, 1).
		// enabled
		AddByte(0x01).Command()

	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return fmt.Errorf("unable to start advertising: %s", err.Error())
	}
	return nil
}

//StopAdvertising directs the controller to stop sending advertisments
func (h *Host) StopAdvertising() error {

	cmd := hci.NewCommandBuilder(hci.CommandLeSetAdvEnable, 1).
		// disabled
		AddByte(0x00).Command()

	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return fmt.Errorf("unable to start advertising: %s", err.Error())
	}
	return nil
}

//SetRandomAddress sets LE Random Device Address to the Controller
func (h *Host) SetRandomAddress(addr hci.BtAddress) error {
	if addr.Atype != hci.LeRandomAddress {
		return fmt.Errorf("invalid address type %s, expected %s",
			addr.Atype.String(), hci.LeRandomAddress.String())
	}

	cmd := hci.NewCommandBuilder(hci.CommandLeSetRandomAddress, 6).
		AddBtAddress(addr).Command()

	if err := h.executeStatusParamCommand(&cmd); err != nil {
		return fmt.Errorf("unable to set random address: %s", err.Error())
	}
	return nil
}

// Disconnect starts disconnecting the connection with given handle.
// Indication will be sent once connection is disconnected
func (h *Host) Disconnect(handle hci.ConnectionHandle) error {
	cmd := hci.NewCommandBuilder(hci.CommandDisconnect, 3).
		// Handle for the connection to disconnect
		AddConnectionHandle(handle).
		// reason for disconnection
		AddByte(0x13).Command()
	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("unable to initiate disconnection: %s", err.Error())
	}
	return nil
}

// Deinit will deinitialize Host
func (h *Host) Deinit() {
	logging.Debug.Printf("Deinitializing host")
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}
	// not checking the return value since there is not much we can do on error
	h.executeStatusParamCommand(&cmd)
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
	close(h.Indications)
	logging.Debug.Printf("Deinitialization done")
}
