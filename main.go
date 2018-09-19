package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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

// Command line settings
type settings struct {
	device       string
	active       bool
	duration     int
	debug        bool
	addrFilter   string
	vendorFilter string
	adTypeFilter string
	ruuvi        bool
}

// Information about found device
type foundDevice struct {
	structures []*hci.AdStructure
	lastSeen   time.Time
	rssi       int8
	types      []hci.AdvType
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
	flag.BoolVar(&cmdline.ruuvi, "ruuvi", false, "Scan and display information about found Ruuvi tags")
}

func parseAddressFilters(addresses string) ([]filter.AdFilter, error) {

	addrs := strings.Split(addresses, ";")
	parsed := make([]filter.AdFilter, len(addrs))
	for i, addr := range addrs {
		atype := hci.LePublicAddress
		if strings.Contains(addr, ",") {
			parts := strings.Split(addr, ",")
			if len(parts) != 2 {
				return nil, fmt.Errorf("Invalid address specification \"%s\"", addresses)
			}
			switch parts[1] {
			case "public":
				atype = hci.LePublicAddress
			case "private":
				fallthrough
			case "random":
				atype = hci.LeRandomAddress
			default:
				return nil, fmt.Errorf("Invalid address type \"%s\"", parts[1])
			}
			addr = parts[0]
		}
		baddr, err := hci.BtAddressFromString(addr)
		if err != nil {
			return nil, fmt.Errorf("Invalid filter (%s)", err.Error())
		}
		baddr.Atype = atype
		log.Printf("Parsed address %s", baddr.String())
		parsed[i] = filter.ByAddress(baddr)
	}
	return parsed, nil
}

func parseVendorSpecFilter(data string) (filter.AdFilter, error) {

	if strings.HasPrefix(data, "0x") {
		data = data[2:]
	}
	bytes, err := hex.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("Invalid vendor specific data specification (%s)", err.Error())
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
			return nil, fmt.Errorf("Invalid Ad Type value \"%s\" (%s)", part, err.Error())
		}
		if len(data) > 1 {
			return nil, fmt.Errorf("Invald value for Ad Structure type (%s), expected one byte in hexadecimal", part)
		}
		filters[i] = filter.ByAdType(hci.AdType(data[0]))
	}
	return filters, nil
}

func formatAddress(addr hci.BtAddress) string {
	addrstr := fmt.Sprintf("%s", addr.String())
	if addr.Atype == hci.LeRandomAddress {
		addrstr += ",random"
	}
	return addrstr
}

func checkFlag(flags byte, flag int) bool {
	return (int(flags) & flag) == flag
}

func decodeAdFlags(flags []byte) string {
	if len(flags) != 1 {
		return "(Invalid)"
	}
	str := ""
	str += "["
	for i := 7; i >= 0; i-- {
		if checkFlag(flags[0], (0x01 << uint8(i))) {
			str += "1"
		} else {
			str += "0"
		}
	}
	str += "] "
	if flags[0] == 0 {
		return str
	}
	str += "("
	if checkFlag(flags[0], hci.AdFlagLimitedDisc) {
		str += "LE Limited Discoverable,"
	}
	if checkFlag(flags[0], hci.AdFlagGeneralDisc) {
		str += "LE General Discoverable,"
	}
	if checkFlag(flags[0], hci.AdFlagNoBrEdr) {
		str += "BR/EDR not supported,"
	}
	if checkFlag(flags[0], hci.AdFlagLeBrEdrController) {
		str += "LE & BR/EDR (controller),"
	}
	if checkFlag(flags[0], hci.AdFlagLeBrEdrHost) {
		str += "LE & BR/EDR (host),"
	}
	str = str[:len(str)-1]
	str += ")"
	return str
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
func printCollectedInfo(infoMap map[hci.BtAddress]*foundDevice) {
	fmt.Printf("\nFound %d devices:\n", len(infoMap))
	for key, val := range infoMap {
		fmt.Printf("Device %s (RSSI:%d dBm; last seen %s):\n", formatAddress(key), val.rssi, val.lastSeen.Format(time.Stamp))
		fmt.Printf("Events: ")
		for i, t := range val.types {
			if i > 0 {
				fmt.Printf(",")
			}
			fmt.Printf("%s", t.String())
		}
		fmt.Printf("\n")
		fmt.Printf("Advertising Data Structures:\n")
		for _, ad := range val.structures {
			switch ad.Typ {
			case hci.AdFlags:
				fmt.Printf("\t%s; %s\n", ad.String(), decodeAdFlags(ad.Data))
			case hci.AdCompleteLocalName:
				fallthrough
			case hci.AdShortenedLocalName:
				fmt.Printf("\t%s\n\t\tName: \"%s\"\n", ad.String(), string(ad.Data))
			case hci.AdDeviceAddress:
				fmt.Printf("\t%s (%s)\n", ad.String(), decodeDeviceAddress(ad.Data))
			default:
				fmt.Printf("\t%s\n", ad.String())
			}
		}
	}
}

type loopFunc func(chan *host.ScanReport)

//listen for ruuvi tag advertisments and print out the decoded information
func ruuviLoop(reportChan chan *host.ScanReport) {
	for sr := range reportChan {
		for _, ads := range sr.Data {
			if ads.Typ == hci.AdManufacturerSpecific && len(ads.Data) >= 2 && binary.LittleEndian.Uint16(ads.Data) == 0x0499 {
				ruuvi, err := ruuvi.Unmarshall(ads.Data)
				if err != nil {
					log.Printf("Unable to parse ruuvi data: %s\n", err.Error())
					continue
				}
				fmt.Printf("Ruuvi device %s (RSSI:%d dBm)\n", formatAddress(sr.Address), sr.Rssi)
				fmt.Printf("\tHumidity: %.2f%% Temperature: %.2fC Pressure: %dPa Battery voltage: %dmV\n", ruuvi.Humidity, ruuvi.Temperature, ruuvi.Pressure, ruuvi.Voltage)
				fmt.Printf("\tAcceleration X: %.2fG, Y: %.2fG, Z: %.2fG\n", ruuvi.AccelerationX, ruuvi.AccelerationY, ruuvi.AccelerationZ)
			}
		}
	}
}

//listen for incoming scan reports, collect data and print it once the channel closes
func collectorLoop(reportChan chan *host.ScanReport) {
	collected := make(map[hci.BtAddress]*foundDevice)
	for sr := range reportChan {
		dev, found := collected[sr.Address]
		if !found {
			if !cmdline.debug {
				fmt.Printf(".")
			}
			ndev := &foundDevice{structures: sr.Data, rssi: sr.Rssi, lastSeen: time.Now()}
			ndev.types = make([]hci.AdvType, 1, 2)
			ndev.types[0] = sr.Type
			collected[sr.Address] = ndev
		} else {
			for _, ads := range sr.Data {
				discard := false
				for _, s := range dev.structures {
					// Do not add the data if we already have the
					// exact data
					if s.Typ == ads.Typ && bytes.Equal(s.Data, ads.Data) {
						discard = true
						break
					}
				}
				if !discard {
					dev.structures = append(dev.structures, ads)
				}
			}
			newType := true
			for _, t := range dev.types {
				if t == sr.Type {
					newType = false
					break
				}
			}
			if newType {
				dev.types = append(dev.types, sr.Type)
			}
			dev.rssi = sr.Rssi
			dev.lastSeen = time.Now()
		}
	}
	printCollectedInfo(collected)
}

func main() {

	flag.Parse()
	if cmdline.device == "" {
		fmt.Printf("Missing device name\n")
		os.Exit(255)
	}

	if !cmdline.debug {
		log.SetOutput(ioutil.Discard)
	}
	var filters []filter.AdFilter
	if cmdline.addrFilter != "" {
		if cmdline.ruuvi {
			fmt.Printf("Address filters not supported on Ruuvi tag mode\n")
			os.Exit(255)
		}
		var err error
		if filters, err = parseAddressFilters(cmdline.addrFilter); err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(255)
		}
	}

	if cmdline.vendorFilter != "" {
		if cmdline.ruuvi {
			fmt.Printf("Vendor filter not supported on Ruuvi tag mode\n")
			os.Exit(255)
		}
		filt, err := parseVendorSpecFilter(cmdline.vendorFilter)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(255)
		}
		filters = append(filters, filt)
	}

	if cmdline.adTypeFilter != "" {
		if cmdline.ruuvi {
			fmt.Printf("AD type filter not supported on Ruuvi tag mode\n")
			os.Exit(255)
		}
		filt, err := parseAdTypeFilters(cmdline.adTypeFilter)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(255)
		}
		for _, f := range filt {
			filters = append(filters, f)
		}
	}

	var loop loopFunc
	if cmdline.ruuvi {
		filters = append(filters, filter.ByVendor([]byte{0x99, 0x04}))
		loop = ruuviLoop
	} else {
		loop = collectorLoop
	}

	log.Printf("Using device %s ", cmdline.device)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	raw, err := hci.Raw(cmdline.device)
	if err != nil {
		fmt.Printf("Error while opening RAW HCI socket: %s\nAre you running as root and have you run sudo hciconfig %s down?\n", err.Error(), cmdline.device)
		os.Exit(255)
	}

	host := host.New(raw)
	if err = host.Init(); err != nil {
		fmt.Printf("Unable to initialize host: %s\n", err.Error())
		host.Deinit()
		os.Exit(255)
	}

	reportChan, err := host.StartScanning(cmdline.active, filters)
	if err != nil {
		fmt.Printf("Unable to start scanning: %s\n", err.Error())
		host.Deinit()
		os.Exit(255)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		loop(reportChan)
		wg.Done()
	}()

	ch := time.Tick(time.Duration(cmdline.duration) * time.Second)
	select {
	case <-ch:
	case s := <-sig:
		log.Printf("Received signal %s, stopping ", s.String())

	}
	host.StopScanning()
	host.Deinit()
	wg.Wait()
}
