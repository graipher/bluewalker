package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/logging"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

const (
	//BluewalkerVersion contains the current version string
	BluewalkerVersion string = "0.2.5"
)

// Command line settings
type settings struct {
	device         string
	active         bool
	duration       int
	debug          bool
	trace          bool
	addrFilter     string
	partAddrFilter string
	vendorFilter   string
	adTypeFilter   string
	adDataFilter   string
	irkFilter      string
	ruuvi          bool
	json           bool
	socketPath     string
	lsocketPath    string
	observer       bool
	version        bool
	broadcaster    bool
	advData        string
	scanResp       string
	randomAddr     string
	filePath       string
}

// Information about found device
type foundDevice struct {
	Structures []*hci.AdStructure `json:"data"`
	LastSeen   time.Time          `json:"last"`
	Rssi       int8               `json:"rssi"`
	Types      []hci.AdvType      `json:"types"`
	Device     hci.BtAddress      `json:"device"`
}

func (dev *foundDevice) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Device %s (RSSI:%d dBm; last seen %s):\nEvents:", formatAddress(dev.Device), dev.Rssi, dev.LastSeen.Format(time.Stamp)))
	for i, t := range dev.Types {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(t.String())
	}
	sb.WriteString("\nAdvertising Data Structures:\n")
	for _, ad := range dev.Structures {
		sb.WriteString(fmt.Sprintf("\t%s\n", decodeAdStructure(ad)))
	}
	return sb.String()
}

// Command line settings from user
var cmdline settings

func init() {
	flag.StringVar(&cmdline.device, "device", "", "HCI device to use")
	flag.BoolVar(&cmdline.active, "active", false, "Active scanning")
	flag.IntVar(&cmdline.duration, "duration", 5, "Number of seconds to scan, -1 to scan indefinitely")
	flag.BoolVar(&cmdline.debug, "debug", false, "Enable debug messages")
	flag.BoolVar(&cmdline.trace, "log-trace", false, "Enable more verbose trace logging in addition to debugging")
	flag.StringVar(&cmdline.addrFilter, "filter-addr", "", "List of addresses where advertisement data is accepted from")
	flag.StringVar(&cmdline.partAddrFilter, "filter-partial-addr", "", "Filter by partial address bytes")
	flag.StringVar(&cmdline.vendorFilter, "filter-vendor", "", "Only show devices whose vendor specific advertising data starts with given bytes")
	flag.StringVar(&cmdline.adTypeFilter, "filter-adtype", "", "Only show devices whose Advertising data contains structures with specified type(s)")
	flag.StringVar(&cmdline.adDataFilter, "filter-addata", "", "Only show devices whose Advertising Data matches given filter (Format: \"<type>,<data>;<type>,<data>\", all values hexadecimal)")
	flag.StringVar(&cmdline.irkFilter, "filter-irk", "", "Only show devices which can be resolved by given IRK")
	flag.BoolVar(&cmdline.ruuvi, "ruuvi", false, "Scan and display information about found Ruuvi tags")
	flag.BoolVar(&cmdline.json, "json", false, "Output data as json")
	flag.StringVar(&cmdline.socketPath, "unix", "", "Unix socket path where to write results")
	flag.BoolVar(&cmdline.observer, "observer", false, "Do scanning in observer mode (display advertising packets as they are received)")
	flag.BoolVar(&cmdline.broadcaster, "broadcast", false, "Send advertising data instead of scanning for it")
	flag.StringVar(&cmdline.advData, "adv-data", "", "Advertising data to send on broadcast mode (Format: \"<type>,<data>;<type>,<data>\", all values hexadecimal)")
	flag.StringVar(&cmdline.scanResp, "scan-resp", "", "Scan response data to send on broadcast mode (Format: \"<type>,<data>;<type>,<data>\", all values hexadecimal)")
	flag.StringVar(&cmdline.randomAddr, "random-addr", "", "Random LE Address to set")
	flag.BoolVar(&cmdline.version, "version", false, "Print version number of the program")
	flag.StringVar(&cmdline.filePath, "output-file", "", "Write output to given file, ('-' to indicate stdout)")
	flag.StringVar(&cmdline.lsocketPath, "listen-unix", "", "Path to socket for listening incoming UNIX socket connections")

}

//print the collected information about found devices
func printCollectedInfo(infoMap map[hci.BtAddress]*foundDevice, out output) error {

	if cmdline.json {
		size := len(infoMap)
		devices := make([]*foundDevice, size)
		i := 0
		for _, val := range infoMap {
			devices[i] = val
			i++
		}
		return out.writeAsJSON(devices)
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("\nFound %d devices:\n", len(infoMap)))
	for _, val := range infoMap {
		sb.WriteString(val.String())
	}
	return out.write(sb.String())
}

type loopFunc func(chan *host.ScanReport, output, chan int)

func ruuviOutputJSON(out output, data *ruuvi.Data, address hci.BtAddress, rssi int8) error {

	return out.writeAsJSON(struct {
		Device hci.BtAddress `json:"device"`
		Rssi   int8          `json:"rssi"`
		Values *ruuvi.Data   `json:"sensors"`
	}{address, rssi, data})
}

func ruuviOutput(out output, data *ruuvi.Data, address hci.BtAddress, rssi int8) error {
	bld := new(strings.Builder)

	v5data := data.Seqno != ruuvi.SeqnoNA

	fmt.Fprintf(bld, "Ruuvi device %s, Data format:", formatAddress(address))
	if v5data {
		fmt.Fprintf(bld, "v5 ")
	} else {
		fmt.Fprintf(bld, "v3 ")
	}
	fmt.Fprintf(bld, "(RSSI %d dBm)\n", rssi)
	fmt.Fprintf(bld, "\tHumidity: %.2f%% Temperature: %.2fC Pressure: %dPa Battery voltage: %dmV\n", data.Humidity, data.Temperature, data.Pressure, data.Voltage)
	fmt.Fprintf(bld, "\tAcceleration X: %.2fG, Y: %.2fG, Z: %.2fG\n", data.AccelerationX, data.AccelerationY, data.AccelerationZ)
	if v5data {
		fmt.Fprintf(bld, "\tTxPower: %d dBm, Moves: %d, Seqno: %d\n", data.TxPower, data.MoveCount, data.Seqno)
	}
	return out.write(bld.String())
}

//listen for ruuvi tag advertisments and print out the decoded information
func ruuviLoop(reportChan chan *host.ScanReport, out output, term chan int) {
	var outputf func(*ruuvi.Data, hci.BtAddress, int8) error
	if cmdline.json {
		outputf = func(data *ruuvi.Data, addr hci.BtAddress, rssi int8) error {
			return ruuviOutputJSON(out, data, addr, rssi)
		}
	} else {
		outputf = func(data *ruuvi.Data, addr hci.BtAddress, rssi int8) error {
			return ruuviOutput(out, data, addr, rssi)
		}
	}
	for sr := range reportChan {
		for _, ads := range sr.Data {
			if ads.Typ == hci.AdManufacturerSpecific && len(ads.Data) >= 2 && binary.LittleEndian.Uint16(ads.Data) == 0x0499 {
				ruuviData, err := ruuvi.Decode(ads.Data)
				if err != nil {
					logging.Warning.Printf("Unable to parse ruuvi data: %v", err)
					continue
				}
				if err := outputf(ruuviData, sr.Address, sr.Rssi); err != nil {
					errorMessage(fmt.Sprintf("Unable to write output (%s), terminating", err.Error()))
					term <- 1
					break
				}
			}
		}
	}
}

func observerLoop(reportChan chan *host.ScanReport, out output, term chan int) {
	var outputf func(*foundDevice) error
	if cmdline.json {
		outputf = func(found *foundDevice) error {
			return out.writeAsJSON(found)
		}
	} else {
		outputf = func(found *foundDevice) error {
			return out.write(found.String())
		}
	}
	for sr := range reportChan {
		found := &foundDevice{Structures: sr.Data,
			Rssi:     sr.Rssi,
			LastSeen: time.Now(),
			Device:   sr.Address}

		found.Types = []hci.AdvType{sr.Type}
		if err := outputf(found); err != nil {
			errorMessage(fmt.Sprintf("Unable to write output (%s), terminating", err.Error()))
			term <- 1
			break
		}
	}
}

//listen for incoming scan reports, collect data and print it once the channel closes
func collectorLoop(reportChan chan *host.ScanReport, out output, term chan int) {
	collected := make(map[hci.BtAddress]*foundDevice)
	for sr := range reportChan {
		dev, found := collected[sr.Address]
		if !found {
			if !cmdline.debug {
				fmt.Printf(".")
			}
			ndev := &foundDevice{Device: sr.Address, Structures: sr.Data, Rssi: sr.Rssi, LastSeen: time.Now()}
			ndev.Types = make([]hci.AdvType, 1, 2)
			ndev.Types[0] = sr.Type
			collected[sr.Address] = ndev
		} else {
			for _, ads := range sr.Data {
				discard := false
				for _, s := range dev.Structures {
					// Do not add the data if we already have the
					// exact data
					if s.Typ == ads.Typ && bytes.Equal(s.Data, ads.Data) {
						discard = true
						break
					}
				}
				if !discard {
					dev.Structures = append(dev.Structures, ads)
				}
			}
			newType := true
			for _, t := range dev.Types {
				if t == sr.Type {
					newType = false
					break
				}
			}
			if newType {
				dev.Types = append(dev.Types, sr.Type)
			}
			dev.Rssi = sr.Rssi
			dev.LastSeen = time.Now()
		}
	}
	// ignore error, we are about to close anyway
	printCollectedInfo(collected, out)
}

// write error message to user
func errorMessage(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
}

//error_critical will print given error message and terminate the program
// if host is non-nil, it will be deinitialized befor stoppping
func errorCritical(host *host.Host, message string) {
	errorMessage(message)
	if host != nil {
		host.Deinit()
	}
	os.Exit(255)
}

func main() {

	flag.Parse()

	if cmdline.version {
		fmt.Fprintf(os.Stdout, "%s v%s\n", os.Args[0], BluewalkerVersion)
		os.Exit(0)
	}

	if cmdline.device == "" {
		errorCritical(nil, "Missing device name")
	}

	if cmdline.debug {
		logging.SetLogLevel(logging.DEBUG)
	}
	if cmdline.trace {
		logging.SetLogLevel(logging.TRACE)
	}
	var filters []filter.AdFilter
	if cmdline.addrFilter != "" {
		if filt, err := parseAddressFilters(cmdline.addrFilter); err != nil {
			errorCritical(nil, fmt.Sprintf("%v", err))
		} else {
			filters = append(filters, filt)
		}
	}

	if cmdline.vendorFilter != "" {
		if cmdline.ruuvi {
			errorCritical(nil, "Vendor filter not supported on Ruuvi tag mode")
		}
		filt, err := parseVendorSpecFilter(cmdline.vendorFilter)
		if err != nil {
			errorCritical(nil, fmt.Sprintf("%v", err))
		}
		filters = append(filters, filt)
	}

	if cmdline.adTypeFilter != "" {
		if cmdline.ruuvi {
			errorCritical(nil, "AD type filter not supported on Ruuvi tag mode")
		}
		if filt, err := parseAdTypeFilters(cmdline.adTypeFilter); err != nil {
			errorCritical(nil, fmt.Sprintf("%v", err))
		} else {
			filters = append(filters, filt)
		}
	}

	if cmdline.adDataFilter != "" {
		if cmdline.ruuvi {
			errorCritical(nil, "AD type filter not supported on Ruuvi tag mode")
		}
		if filt, err := parseAdDataFilters(cmdline.adDataFilter); err != nil {
			errorCritical(nil, fmt.Sprintf("%v", err))
		} else {
			filters = append(filters, filt)
		}
	}

	if cmdline.irkFilter != "" {
		filt, err := parseIrkFilter(cmdline.irkFilter)
		if err != nil {
			errorCritical(nil, fmt.Sprintf("%v", err))
		}
		filters = append(filters, filt)
	}

	if cmdline.partAddrFilter != "" {
		filt, err := parsePartialAddrFilter(cmdline.partAddrFilter)
		if err != nil {
			errorCritical(nil, fmt.Sprintf("%v", err))
		}
		filters = append(filters, filt)
	}

	if cmdline.broadcaster {
		if len(filters) > 0 {
			errorCritical(nil, "Filters not available on broadcaster mode")
		}
		if cmdline.ruuvi {
			errorCritical(nil, "Ruuvi mode not available on broadcaster mode")
		}
		if cmdline.json {
			errorCritical(nil, "JSON output not available on broadcaster mode")
		}
		if cmdline.observer {
			errorCritical(nil, "observer mode not available on broadcaster mode")
		}
		if cmdline.socketPath != "" || cmdline.lsocketPath != "" {
			errorCritical(nil, "No unix socket support on broadcaster mode")
		}
		if cmdline.filePath != "" {
			errorCritical(nil, "File output not available on broadcaster mode")
		}
		if cmdline.advData == "" {
			errorCritical(nil, "No Advertising Data set")
		}
	} else {
		if cmdline.advData != "" || cmdline.scanResp != "" {
			errorCritical(nil, "Advertising or scan response data can be set only on broadcaster mode")
		}
	}
	var rAddr hci.BtAddress
	if cmdline.randomAddr != "" {
		addr, err := parseAddress(cmdline.randomAddr)
		if err != nil {
			errorCritical(nil, fmt.Sprintf("Could not parse random address (%v)", err))
		}
		if addr.Atype != hci.LeRandomAddress {
			// force the address type to be random, as we are setting the
			// randome address
			addr.Atype = hci.LeRandomAddress
		}
		rAddr = addr
	}

	if cmdline.duration == 0 || cmdline.duration < -1 {
		errorCritical(nil, fmt.Sprintf("Invalid duration %d", cmdline.duration))
	}

	var out output
	if cmdline.socketPath != "" {
		if cmdline.filePath != "" {
			errorCritical(nil, "Socket and file output can not be defined at the same time")
		}
		if cmdline.lsocketPath != "" {
			errorCritical(nil, "Listening and connection UNIX socket output can not be defined at the same time")
		}
		if !cmdline.json {
			fmt.Fprintf(os.Stderr, "Forcing JSON mode when writing to socket. Use -json to silence this warning\n")
			cmdline.json = true
		}
		var err error
		out, err = outputForSocket(cmdline.socketPath)
		if err != nil {
			errorCritical(nil, fmt.Sprintf("Unable to open unix socket at %s (%v)", cmdline.socketPath, err))
		}
		defer out.Close()
	} else if cmdline.lsocketPath != "" {
		if cmdline.filePath != "" {
			errorCritical(nil, "Socket and file output can not be defined at the same time")
		}
		if !cmdline.json {
			fmt.Fprintf(os.Stderr, "Forcing JSON mode when writing to socket. Use -json to silence this warning\n")
			cmdline.json = true
		}
		var err error
		out, err = outputForListeningSocket(cmdline.lsocketPath)
		if err != nil {
			errorCritical(nil, fmt.Sprintf("Unable to open unix socket at %s (%v)", cmdline.socketPath, err))
		}
		defer out.Close()
	} else if cmdline.filePath != "" {
		var err error
		out, err = outputForFile(cmdline.filePath)
		if err != nil {
			errorCritical(nil, fmt.Sprintf("Unable to open output file %s (%v)", cmdline.filePath, err))
		}
		defer out.Close()
	} else {
		out = defaultOutput()
	}

	var loop loopFunc
	termChan := make(chan int)
	if cmdline.ruuvi {
		filters = append(filters, filter.ByVendor([]byte{0x99, 0x04}))
		loop = ruuviLoop
	} else if cmdline.observer {
		loop = observerLoop
	} else if cmdline.broadcaster {
		loop = nil
	} else {
		loop = collectorLoop
	}

	var ads []*hci.AdStructure
	var scanResp []*hci.AdStructure
	if cmdline.broadcaster {
		if data, err := parseAdStructures(cmdline.advData); err != nil {
			errorCritical(nil, fmt.Sprintf("Unable to parse advertising data: %v", err))
		} else {
			ads = data
		}
		if cmdline.scanResp != "" {
			if data, err := parseAdStructures(cmdline.scanResp); err != nil {
				errorCritical(nil, fmt.Sprintf("Unable to parse scan response data: %v", err))
			} else {
				scanResp = data
			}
		}
	}

	logging.Debug.Printf("Using device %s ", cmdline.device)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	raw, err := hci.Raw(cmdline.device)
	if err != nil {
		errorCritical(nil, fmt.Sprintf("Error while opening RAW HCI socket: %v\nAre you running as root and have you run sudo hciconfig %s down?", err, cmdline.device))
	}

	var execErr *host.CommandExecutionError
	host := host.New(raw)
	if err = host.Init(); err != nil {
		if errors.As(err, &execErr) && execErr.ErrorCode() == hci.StatusUnknownCommand {
			errorCritical(host, fmt.Sprintf("Host initialization failed. This is likely beacause %s does not support Bluetooth LE (%v)", cmdline.device, err))
		}

		errorCritical(host, fmt.Sprintf("Unable to initialize host: %v", err))
	}

	if cmdline.randomAddr != "" {
		if err = host.SetRandomAddress(rAddr); err != nil {
			errorCritical(host, fmt.Sprintf("%v", err))
		}
	}

	var wg sync.WaitGroup
	if cmdline.broadcaster {
		params := hci.DefaultAdvParameters()
		if scanResp != nil {
			// scan response will be set also, set advertising type to
			// scannable
			params.Type = hci.AdvScanInd
		}
		if cmdline.randomAddr != "" {
			// random address has been set, use that to advertise
			params.OwnAddrType = hci.AdvAddressRandom
		}
		if err := host.SetAdvertisingParams(params); err != nil {
			errorCritical(host, fmt.Sprintf("Unable to set advertising parameters: %v", err))
		}

		bld := new(strings.Builder)
		fmt.Fprintf(bld, "Setting advertising data:\n")
		for _, a := range ads {
			fmt.Fprintf(bld, "\t%s\n", a.String())
		}
		out.write(bld.String())
		if err := host.SetAdvertisingData(ads); err != nil {
			errorCritical(host, fmt.Sprintf("Unable to set advertising data: %v", err))
		}
		if scanResp != nil {
			bld.Reset()
			fmt.Fprintf(bld, "Setting scan response data:\n")
			for _, a := range scanResp {
				fmt.Fprintf(bld, "\t%s\n", a.String())
			}
			out.write(bld.String())
			if err := host.SetScanResponse(scanResp); err != nil {
				errorCritical(host, fmt.Sprintf("Unable to set scan response data: %v", err))
			}
		}
		if err := host.StartAdvertising(); err != nil {
			errorCritical(host, fmt.Sprintf("Unable to start advertising: %v", err))
		}
		out.write("Advertising...")
	} else {
		reportChan, err := host.StartScanning(cmdline.active, filters)
		if err != nil {
			errorCritical(host, fmt.Sprintf("Unable to start scanning: %v", err))
		}

		wg.Add(1)
		go func() {
			loop(reportChan, out, termChan)
			wg.Done()
		}()
	}

	var tick <-chan time.Time
	if cmdline.duration != -1 {
		tick = time.Tick(time.Duration(cmdline.duration) * time.Second)
	}
	select {
	case <-tick:
	case <-termChan:
	case s := <-sig:
		logging.Debug.Printf("Received signal %s, stopping ", s)
	}

	if cmdline.broadcaster {
		host.StopAdvertising()
		out.write(".Done\n")
	} else {
		host.StopScanning()
	}

	host.Deinit()
	wg.Wait()
}
