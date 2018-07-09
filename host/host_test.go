package host

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

type testTransport struct {
	received [][]byte
	rfunc    func() ([]byte, error)
	wfunc    func([]byte) error
}

func (tt *testTransport) Close() {

}

func (tt *testTransport) Read() ([]byte, error) {
	if tt.rfunc == nil {
		return make([]byte, 0), nil
	}
	return tt.rfunc()
}

func (tt *testTransport) Write(data []byte) error {
	if tt.wfunc == nil {
		tt.received = append(tt.received, data)
		return nil
	}
	return tt.wfunc(data)
}

func mkCommandCompleteEvent(status hci.ErrorCode, op hci.CommandOpCode, t *testing.T) *hci.CommandCompleteEvent {

	buf := make([]byte, 6)
	buf[0] = byte(hci.EventCodeCommandComplete)
	buf[1] = 4 // length
	buf[2] = 1 // num HCI packets
	binary.LittleEndian.PutUint16(buf[3:], uint16(op))
	buf[5] = byte(status)

	evt, err := hci.DecodeEvent(buf)
	if err != nil {
		t.Fatalf("Can not create event: %s", err.Error())
	}
	cc, err := hci.DecodeCommandComplete(evt)
	if err != nil {
		t.Fatalf("Can not create event: %s", err.Error())
	}
	return cc
}

func createTransport(rfunc func() ([]byte, error), wfunc func([]byte) error) {

	tt := new(testTransport)
	tt.received = make([][]byte, 0)
	tt.wfunc = wfunc
	tt.rfunc = rfunc
}

func TestCommandExecSuccess(t *testing.T) {

	h := New(nil)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	cc := mkCommandCompleteEvent(0, hci.CommandReset, t)

	ch := make(chan error)
	go func(errChan chan error, t *testing.T) {
		err := h.executeStatusCommand(&cmd)
		errChan <- err
	}(ch, t)
	exec := <-h.cmd
	exec.complete(cc)
	err := <-ch
	if err != nil {
		t.Errorf("Command execution failed unexpectedly")
	}
}

func TestCommandExecFail(t *testing.T) {

	h := New(nil)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	cc := mkCommandCompleteEvent(hci.StatusInvalidParams, hci.CommandReset, t)

	ch := make(chan error)

	go func(errChan chan error, t *testing.T) {
		err := h.executeStatusCommand(&cmd)
		errChan <- err
	}(ch, t)
	exec := <-h.cmd
	exec.complete(cc)
	err := <-ch
	if err == nil {
		t.Errorf("Expected the command execution to fail")
	}
}

func TestCommandExecFail2(t *testing.T) {

	h := New(nil)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan error)

	go func(errChan chan error, t *testing.T) {
		err := h.executeStatusCommand(&cmd)
		errChan <- err
	}(ch, t)
	exec := <-h.cmd
	exec.fail(fmt.Errorf("Execution failure"))
	err := <-ch
	if err == nil {
		t.Errorf("Expected the command execution to fail")
	}
}

func TestExecutorHappy(t *testing.T) {

	trp := new(testTransport)
	trp.received = make([][]byte, 0)
	h := New(trp)

	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan int)

	ex := exec{cmd: &cmd,
		complete: func(cc *hci.CommandCompleteEvent) {
			ch <- 1
		}, fail: func(er error) {

		}}
	cc := mkCommandCompleteEvent(hci.StatusSuccess, hci.CommandReset, t)

	go h.executor()
	h.cmd <- &ex
	h.cc <- cc

	// Wait for ex.complete() to be called
	<-ch

	// Check that the command was written
	if len(trp.received) != 1 {
		t.Errorf("Command was not written to transport")
	}
	close(h.cmd)
	close(h.cc)
}

func TestExecutorWriteFail(t *testing.T) {

	trp := new(testTransport)
	trp.wfunc = func(buf []byte) error {
		return fmt.Errorf("Test write failed")
	}
	h := New(trp)

	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan int)

	ex := exec{cmd: &cmd,
		complete: func(cc *hci.CommandCompleteEvent) {
		}, fail: func(er error) {
			ch <- 1
		}}

	go h.executor()
	h.cmd <- &ex

	// Wait for ex.fail() to be called
	<-ch
	close(h.cmd)
	close(h.cc)
}

func TestEventHandlerCC(t *testing.T) {

	h := New(nil)
	buf := make([]byte, 6)
	buf[0] = byte(hci.EventCodeCommandComplete)
	buf[1] = 4 // length
	buf[2] = 1 // num HCI packets
	binary.LittleEndian.PutUint16(buf[3:], uint16(hci.CommandReset))
	buf[5] = byte(hci.StatusSuccess)

	go h.eventHandler()
	h.evt <- buf
	cc := <-h.cc
	close(h.evt)
	if cc.GetCommandOpcode() != hci.CommandReset || cc.GetStatusParameter() != hci.StatusSuccess {
		t.Errorf("Received unexpected Command Complete event")
	}
}

func TestEventHandlerMeta(t *testing.T) {

	h := New(nil)
	buf, err := hex.DecodeString("3e1402010000010203040506080309410003084100ff")
	if err != nil {
		t.Fatalf("Invalid test data (%s)", err.Error())
	}

	go h.eventHandler()
	h.evt <- buf
	ad := <-h.ad
	close(h.evt)
	bdaddr := hci.ToBtAddress([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06})
	if ad.Address != bdaddr {
		// the Advertising report parsing is tested elsewhere
		t.Errorf("Unexpected address in edvertising report")
	}
}

func TestEventReceiver(t *testing.T) {

	trp := new(testTransport)
	count := 0
	trp.rfunc = func() ([]byte, error) {
		if count > 1 {
			// we need to return error to trigger closing
			// of the event loop. Ugly, but works
			return nil, fmt.Errorf("Intentional error")
		}
		count++
		return hex.DecodeString("040e0401030c00")
	}
	h := New(trp)
	h.wg.Add(1)
	go h.eventReceiver()

	buf := <-h.evt
	h.closing = true

	rcv, err := hci.DecodeEvent(buf)
	if err != nil {
		t.Errorf("Received malformed event: %s", err.Error())
	}
	if rcv.Code != hci.EventCodeCommandComplete {
		t.Errorf("Received unexpected event")
	}
	h.wg.Wait()
}

func TestInitHappy(t *testing.T) {

	initCommands := []hci.CommandOpCode{hci.CommandReset, hci.CommandWriteLeHostSupported, hci.CommandSetEventMask, hci.CommandLeSetEventMask}
	h := New(nil)
	ch := make(chan error)
	go func() {
		err := h.initializeController()
		ch <- err
	}()

	for i := 0; i < len(initCommands); i++ {
		cmd := <-h.cmd
		if cmd.cmd.OpCode != initCommands[i] {
			t.Errorf("Expected command %s, got %s", initCommands[i].String(), cmd.cmd.OpCode.String())
		}
		cmd.complete(mkCommandCompleteEvent(hci.StatusSuccess, initCommands[i], t))
	}
	err := <-ch
	if err != nil {
		t.Errorf("Unexpected error in initialization")
	}
}

func TestInitFail(t *testing.T) {

	h := New(nil)
	ch := make(chan error)
	go func() {
		err := h.initializeController()
		ch <- err
	}()

	// make sure we signal error and stop when command fails
	cmd := <-h.cmd
	cmd.complete(mkCommandCompleteEvent(hci.StatusInvalidParams, cmd.cmd.OpCode, t))
	err := <-ch
	if err == nil {
		t.Errorf("Expected initialization to fail")
	}
}

func TestStartScanHappy(t *testing.T) {

	scanCommands := []hci.CommandOpCode{hci.CommandLeSetScanParameters, hci.CommandLeSetScanEnable}
	h := New(nil)
	ch := make(chan error)
	go func() {
		_, err := h.StartScanning(false)
		ch <- err
	}()

	for i := 0; i < len(scanCommands); i++ {
		cmd := <-h.cmd
		if cmd.cmd.OpCode != scanCommands[i] {
			t.Errorf("Expected command %s, got %s", scanCommands[i].String(), cmd.cmd.OpCode.String())
		}
		cmd.complete(mkCommandCompleteEvent(hci.StatusSuccess, scanCommands[i], t))
	}
	err := <-ch
	if err != nil {
		t.Errorf("Unexpected error while starting scan")
	}
}

func TestStartScanFail(t *testing.T) {

	h := New(nil)
	ch := make(chan error)
	go func() {
		_, err := h.StartScanning(false)
		ch <- err
	}()

	// make sure we signal error and stop when command fails
	cmd := <-h.cmd
	cmd.complete(mkCommandCompleteEvent(hci.StatusInvalidParams, cmd.cmd.OpCode, t))
	err := <-ch
	if err == nil {
		t.Errorf("Expected start scan to fail")
	}
}

func TestStartScan2ndFail(t *testing.T) {

	h := New(nil)
	ch := make(chan error)
	go func() {
		_, err := h.StartScanning(false)
		ch <- err
	}()

	// make sure we signal error and stop when command fails
	cmd := <-h.cmd
	cmd.complete(mkCommandCompleteEvent(hci.StatusSuccess, cmd.cmd.OpCode, t))
	cmd = <-h.cmd
	cmd.complete(mkCommandCompleteEvent(hci.StatusInvalidParams, cmd.cmd.OpCode, t))
	err := <-ch
	if err == nil {
		t.Errorf("Expected start scan to fail")
	}
}

func TestStopScanning(t *testing.T) {

	h := New(nil)
	ch := make(chan error)
	go func() {
		err := h.StopScanning()
		ch <- err
	}()

	// make sure we signal error and stop when command fails
	cmd := <-h.cmd
	if cmd.cmd.OpCode != hci.CommandLeSetScanEnable {
		t.Errorf("Unexpected command")
	}
	cmd.complete(mkCommandCompleteEvent(hci.StatusSuccess, cmd.cmd.OpCode, t))
	err := <-ch
	if err != nil {
		t.Errorf("Unexpected error while stopping scan")
	}
}

func TestStopScanningFail(t *testing.T) {

	h := New(nil)
	ch := make(chan error)
	go func() {
		err := h.StopScanning()
		ch <- err
	}()

	// make sure we signal error and stop when command fails
	cmd := <-h.cmd
	if cmd.cmd.OpCode != hci.CommandLeSetScanEnable {
		t.Errorf("Unexpected command")
	}
	cmd.complete(mkCommandCompleteEvent(hci.StatusInvalidParams, cmd.cmd.OpCode, t))
	err := <-ch
	if err == nil {
		t.Errorf("Expected stopping scanning to fail")
	}
}
