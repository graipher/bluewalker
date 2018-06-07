package host

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

type exec struct {
	// command to execute
	cmd *hci.CommandPacket
	// this function is called when execution has completed
	complete func(*hci.CommandCompleteEvent)
	// this is called if command can not be sent
	fail func(error)
}

// Host implements the host side of Bluetooth Host - Controller interface
type Host struct {
	tr hci.Transport
	wg sync.WaitGroup
	// Event channel
	evt chan []byte
	// Commands for executor
	cmd chan *exec
	// CommandComplete events to executor
	cc      chan *hci.CommandCompleteEvent
	closing bool
}

// New returns new host which uses given transport for communicating
// with controller
func New(tr hci.Transport) *Host {

	host := new(Host)
	host.tr = tr
	host.evt = make(chan []byte, 2)
	host.cmd = make(chan *exec)
	host.cc = make(chan *hci.CommandCompleteEvent)
	host.closing = false

	return host
}

func (h *Host) eventReceiver() {

	defer h.wg.Done()
	for !h.closing {
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
		default:
			log.Printf("Received unexpected event %s", evt.Code.String())
		}
	}
	log.Printf("Event handler stopping")
}

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
		select {
		case cc := <-h.cc:
			numCommands = int(cc.GetNumHciCommandPackets())
			log.Printf("Number of HCI packets increased to %d", numCommands)
			if cc.GetCommandOpcode() == e.cmd.OpCode {
				e.complete(cc)
			} else {
				log.Printf("Received unexepcted cc for %s ", cc.GetCommandOpcode().String())
			}
		}
		// XXX check numCommands, we should wait if it is 0
	}
}

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

// Deinit will deinitialize Host
func (h *Host) Deinit() {
	log.Printf("Deinitializing host")
	h.closing = true
	h.tr.Close()
	h.wg.Wait()
	// Now the eventReceiver has stopped. Rest should stop when we close
	// the channels
	close(h.evt)
	close(h.cc)
	log.Printf("Deinitialization done")
}
