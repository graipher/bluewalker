package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

const (
	//BluewalkerVersion contains the current version string
	BluewalkerVersion string = "0.1.0"
)

// Command line settings
type settings struct {
	device       string
	active       bool
	duration     int
	debug        bool
	addrFilter   string
	vendorFilter string
	adTypeFilter string
	irkFilter    string
	ruuvi        bool
	json         bool
	socketPath   string
	observer     bool
	version      bool
}

type output struct {
	wr            io.Writer
	cl            io.Closer
	humanReadable bool
}

func (out *output) write(data string) {
	out.wr.Write([]byte(data))
}

func (out *output) isHumanReadable() bool {
	return out.humanReadable
}

func (out *output) Close() {
	if out.cl != nil {
		out.cl.Close()
	}
}

// writeAsJSON marshals the given data into JSON and prints
// the resulting string(s) using this output. If output is marked
// as HumanReadble, the JSON is indented and printed so that it
// is easy to read. If ouput is not HumanReadable, the json is
// printed as a single -line where data is terminated by newline ('\n')
func (out *output) writeAsJSON(data interface{}) {
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
		out.wr.Write(jdata)
	} else {
		log.Printf("WARN: unable to marshal %q as JSON: %v", data, err)
	}
}

func outputForSocket(path string) (*output, error) {
	unixConn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}
	return &output{wr: unixConn, cl: unixConn, humanReadable: false}, nil
}

func defaultOutput() *output {
	return &output{wr: os.Stdout, humanReadable: true}
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
		switch ad.Typ {
		case hci.AdFlags:
			sb.WriteString(fmt.Sprintf("\t%s; %s\n", ad, decodeAdFlags(ad.Data)))
		case hci.AdCompleteLocalName:
			fallthrough
		case hci.AdShortenedLocalName:
			sb.WriteString(fmt.Sprintf("\t%s\n\t\tName: \"%s\"\n", ad, string(ad.Data)))
		case hci.AdDeviceAddress:
			sb.WriteString(fmt.Sprintf("\t%s (%s)\n", ad, decodeDeviceAddress(ad.Data)))
		default:
			sb.WriteString(fmt.Sprintf("\t%s\n", ad))
		}
	}
	return sb.String()
}

// Command line settings from user
var cmdline settings

func init() {
	flag.StringVar(&cmdline.device, "device", "", "HCI device to use")
	flag.BoolVar(&cmdline.active, "active", false, "Active scanning")
	flag.IntVar(&cmdline.duration, "duration", 5, "Number of seconds to scan")
	flag.BoolVar(&cmdline.debug, "debug", false, "Enable debug messages")
	flag.StringVar(&cmdline.addrFilter, "filter-addr", "", "List of addresses where advertisement data is accepted from")
	flag.StringVar(&cmdline.vendorFilter, "filter-vendor", "", "Only show devices whose vendor specific advertising data starts with given bytes")
	flag.StringVar(&cmdline.adTypeFilter, "filter-adtype", "", "Only show devices whose Advertising data contains structures with specified type(s)")
	flag.StringVar(&cmdline.irkFilter, "filter-irk", "", "Only show devices which can be resolved by given IRK")
	flag.BoolVar(&cmdline.ruuvi, "ruuvi", false, "Scan and display information about found Ruuvi tags")
	flag.BoolVar(&cmdline.json, "json", false, "Output data as json")
	flag.StringVar(&cmdline.socketPath, "unix", "", "Unix socket path where to write results")
	flag.BoolVar(&cmdline.observer, "observer", false, "Do scanning in observer mode (display advertising packets as they are received)")
	flag.BoolVar(&cmdline.version, "version", false, "Print version number of the program")
}

func parseAddressFilters(addresses string) ([]filter.AdFilter, error) {

	addrs := strings.Split(addresses, ";")
	parsed := make([]filter.AdFilter, len(addrs))
	for i, addr := range addrs {
		atype := hci.LePublicAddress
		if strings.Contains(addr, ",") {
			parts := strings.Split(addr, ",")
			if len(parts) != 2 {
				return nil, fmt.Errorf("Invalid address specification %q", addresses)
			}
			switch parts[1] {
			case "public":
				atype = hci.LePublicAddress
			case "private":
				fallthrough
			case "random":
				atype = hci.LeRandomAddress
			default:
				return nil, fmt.Errorf("Invalid address type %q", parts[1])
			}
			addr = parts[0]
		}
		baddr, err := hci.BtAddressFromString(addr)
		if err != nil {
			return nil, fmt.Errorf("Invalid filter (%v)", err)
		}
		baddr.Atype = atype
		log.Printf("Parsed address %s", baddr)
		parsed[i] = filter.ByAddress(baddr)
	}
	return parsed, nil
}

func parseIrkFilter(data string) (filter.AdFilter, error) {
	if strings.HasPrefix(data, "0x") {
		data = data[2:]
	}
	bytes, err := hex.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("Invalid IRK data (%v)", err)
	}
	if len(bytes) != hci.IrkLength {
		return nil, fmt.Errorf("Invalid length for IRK, expected %d bytes, got %d", hci.IrkLength, len(bytes))
	}

	// We assume here that IRK given has LSB in position 0, that is because
	// Linux has it that way. However, the address resolving assumes
	// that key for AES has MSB in position 0 we must change it here.
	irk := make([]byte, len(bytes))
	for i, b := range bytes {
		irk[len(bytes)-i-1] = b
	}

	return filter.ByIrk(irk), nil
}

func parseVendorSpecFilter(data string) (filter.AdFilter, error) {

	if strings.HasPrefix(data, "0x") {
		data = data[2:]
	}
	bytes, err := hex.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("Invalid vendor specific data specification (%v)", err)
	}
	return filter.ByVendor(bytes), nil
}

func parseAdTypeFilters(types string) ([]filter.AdFilter, error) {

	parts := strings.Split(types, ",")
	filters := make([]filter.AdFilter, len(parts))
	for i, part := range parts {
		if strings.HasPrefix(part, "0x") {
			part = part[2:]
		}
		data, err := hex.DecodeString(part)
		if err != nil {
			return nil, fmt.Errorf("Invalid Ad Type value %q (%v)", part, err)
		}
		if len(data) > 1 {
			return nil, fmt.Errorf("Invald value for Ad Structure type (%s), expected one byte in hexadecimal", part)
		}
		filters[i] = filter.ByAdType(hci.AdType(data[0]))
	}
	return filters, nil
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

//print the collected information about found devices
func printCollectedInfo(infoMap map[hci.BtAddress]*foundDevice, out *output) {

	if cmdline.json {
		size := len(infoMap)
		devices := make([]*foundDevice, size)
		i := 0
		for _, val := range infoMap {
			devices[i] = val
			i++
		}
		out.writeAsJSON(devices)
		return
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("\nFound %d devices:\n", len(infoMap)))
	for _, val := range infoMap {
		sb.WriteString(val.String())
	}
	out.write(sb.String())
}

type loopFunc func(chan *host.ScanReport, *output)

func ruuviOuputJSON(out *output, data *ruuvi.Data, address hci.BtAddress, rssi int8) {

	out.writeAsJSON(struct {
		Device hci.BtAddress `json:"device"`
		Rssi   int8          `json:"rssi"`
		Values *ruuvi.Data   `json:"sensors"`
	}{address, rssi, data})
}

func ruuviOutput(out *output, data *ruuvi.Data, address hci.BtAddress, rssi int8) {
	bld := new(strings.Builder)

	fmt.Fprintf(bld, "Ruuvi device %s (RSSI:%d dBm)\n", formatAddress(address), rssi)
	fmt.Fprintf(bld, "\tHumidity: %.2f%% Temperature: %.2fC Pressure: %dPa Battery voltage: %dmV\n", data.Humidity, data.Temperature, data.Pressure, data.Voltage)
	fmt.Fprintf(bld, "\tAcceleration X: %.2fG, Y: %.2fG, Z: %.2fG\n", data.AccelerationX, data.AccelerationY, data.AccelerationZ)
	out.write(bld.String())
}

//listen for ruuvi tag advertisments and print out the decoded information
func ruuviLoop(reportChan chan *host.ScanReport, out *output) {
	for sr := range reportChan {
		for _, ads := range sr.Data {
			if ads.Typ == hci.AdManufacturerSpecific && len(ads.Data) >= 2 && binary.LittleEndian.Uint16(ads.Data) == 0x0499 {
				ruuviData, err := ruuvi.Unmarshall(ads.Data)
				if err != nil {
					log.Printf("Unable to parse ruuvi data: %v", err)
					continue
				}
				if cmdline.json {
					ruuviOuputJSON(out, ruuviData, sr.Address, sr.Rssi)
				} else {
					ruuviOutput(out, ruuviData, sr.Address, sr.Rssi)
				}
			}
		}
	}
}

func observerLoop(reportChan chan *host.ScanReport, out *output) {
	for sr := range reportChan {
		found := &foundDevice{Structures: sr.Data,
			Rssi:     sr.Rssi,
			LastSeen: time.Now(),
			Device:   sr.Address}

		found.Types = []hci.AdvType{sr.Type}

		if cmdline.json {
			out.writeAsJSON(found)
		} else {
			out.write(found.String())
		}
	}
}

//listen for incoming scan reports, collect data and print it once the channel closes
func collectorLoop(reportChan chan *host.ScanReport, out *output) {
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
	printCollectedInfo(collected, out)
}

func main() {

	flag.Parse()

	if cmdline.version {
		fmt.Fprintf(os.Stdout, "%s v%s\n", os.Args[0], BluewalkerVersion)
		os.Exit(0)
	}

	if cmdline.device == "" {
		fmt.Fprintf(os.Stderr, "Missing device name\n")
		os.Exit(255)
	}

	if !cmdline.debug {
		log.SetOutput(ioutil.Discard)
	}
	var filters []filter.AdFilter
	if cmdline.addrFilter != "" {
		if cmdline.ruuvi {
			fmt.Fprintf(os.Stderr, "Address filters not supported on Ruuvi tag mode\n")
			os.Exit(255)
		}
		var err error
		if filters, err = parseAddressFilters(cmdline.addrFilter); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(255)
		}
	}

	if cmdline.vendorFilter != "" {
		if cmdline.ruuvi {
			fmt.Fprintf(os.Stderr, "Vendor filter not supported on Ruuvi tag mode\n")
			os.Exit(255)
		}
		filt, err := parseVendorSpecFilter(cmdline.vendorFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(255)
		}
		filters = append(filters, filt)
	}

	if cmdline.adTypeFilter != "" {
		if cmdline.ruuvi {
			fmt.Fprintf(os.Stderr, "AD type filter not supported on Ruuvi tag mode\n")
			os.Exit(255)
		}
		filt, err := parseAdTypeFilters(cmdline.adTypeFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(255)
		}
		for _, f := range filt {
			filters = append(filters, f)
		}
	}

	if cmdline.irkFilter != "" {
		filt, err := parseIrkFilter(cmdline.irkFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(255)
		}
		filters = append(filters, filt)
	}

	if cmdline.duration == 0 || cmdline.duration < -1 {
		fmt.Fprintf(os.Stderr, "Invalid duration %d\n", cmdline.duration)
		os.Exit(255)
	}

	var out *output
	if cmdline.socketPath != "" {
		if !cmdline.json {
			fmt.Fprintf(os.Stderr, "Forcing JSON mode when writing to socket. Use -json to silence this warning\n")
			cmdline.json = true
		}
		var err error
		out, err = outputForSocket(cmdline.socketPath)
		if err != nil {
			fmt.Printf("Unable to open unix socket at %s (%v)\n", cmdline.socketPath, err)
			os.Exit(255)
		}
		defer out.Close()
	} else {
		out = defaultOutput()
	}

	var loop loopFunc
	if cmdline.ruuvi {
		filters = append(filters, filter.ByVendor([]byte{0x99, 0x04}))
		loop = ruuviLoop
	} else if cmdline.observer {
		loop = observerLoop
	} else {
		loop = collectorLoop
	}

	log.Printf("Using device %s ", cmdline.device)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	raw, err := hci.Raw(cmdline.device)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while opening RAW HCI socket: %v\nAre you running as root and have you run sudo hciconfig %s down?\n", err, cmdline.device)
		os.Exit(255)
	}

	host := host.New(raw)
	if err = host.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to initialize host: %v\n", err)
		host.Deinit()
		os.Exit(255)
	}

	reportChan, err := host.StartScanning(cmdline.active, filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start scanning: %v\n", err)
		host.Deinit()
		os.Exit(255)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		loop(reportChan, out)
		wg.Done()
	}()

	if cmdline.duration == -1 {
		select {
		case s := <-sig:
			log.Printf("Received signal %s, stopping ", s)

		}
	} else {
		ch := time.Tick(time.Duration(cmdline.duration) * time.Second)
		select {
		case <-ch:
		case s := <-sig:
			log.Printf("Received signal %s, stopping ", s)

		}
	}
	host.StopScanning()
	host.Deinit()
	wg.Wait()
}
