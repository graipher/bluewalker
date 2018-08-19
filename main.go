package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
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
}

// Information about found device
type foundDevice struct {
	structures []*hci.AdStructure
	lastSeen   time.Time
	rssi       int8
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
				atype = hci.LePrivateAddress
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
		var err error
		if filters, err = parseAddressFilters(cmdline.addrFilter); err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(255)
		}
	}

	if cmdline.vendorFilter != "" {
		filt, err := parseVendorSpecFilter(cmdline.vendorFilter)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(255)
		}
		filters = append(filters, filt)
	}

	if cmdline.adTypeFilter != "" {
		filt, err := parseAdTypeFilters(cmdline.adTypeFilter)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(255)
		}
		for _, f := range filt {
			filters = append(filters, f)
		}
	}

	log.Printf("Using device %s ", cmdline.device)

	raw, err := hci.Raw(cmdline.device)
	if err != nil {
		log.Printf("Error while opening RAW HCI socket: %s", err.Error())
		os.Exit(255)
	}

	host := host.New(raw)
	if err = host.Init(); err != nil {
		log.Printf("Unable to initialize host: %s", err.Error())
		host.Deinit()
		os.Exit(255)
	}

	reportChan, err := host.StartScanning(cmdline.active, filters)
	if err != nil {
		log.Printf("Unable to start scanning: %s", err.Error())
		host.Deinit()
		os.Exit(255)
	}

	collected := make(map[hci.BtAddress]*foundDevice)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for sr := range reportChan {
			dev, found := collected[sr.Address]
			if !found {
				if !cmdline.debug {
					fmt.Printf(".")
				}
				collected[sr.Address] = &foundDevice{structures: sr.Data, rssi: sr.Rssi, lastSeen: time.Now()}
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
					dev.rssi = sr.Rssi
					dev.lastSeen = time.Now()
					if !discard {
						dev.structures = append(dev.structures, ads)
					}
				}
			}
		}
		wg.Done()
	}()

	ch := time.Tick(time.Duration(cmdline.duration) * time.Second)
	<-ch
	host.StopScanning()
	host.Deinit()
	wg.Wait()

	fmt.Printf("\nFound %d devices:\n", len(collected))
	for key, val := range collected {
		addrstr := ""
		if key.Atype == hci.LePrivateAddress {
			addrstr = fmt.Sprintf("%s,private", key.String())
		} else {
			addrstr = fmt.Sprintf("%s", key.String())
		}
		fmt.Printf("Device %s (RSSI:%d dBm; last seen %s):\n", addrstr, val.rssi, val.lastSeen.Format(time.Stamp))
		for _, ad := range val.structures {
			fmt.Printf("\t%s\n", ad.String())
			if ad.Typ == hci.AdCompleteLocalName || ad.Typ == hci.AdShortenedLocalName {
				fmt.Printf("\t\tName: \"%s\"\n", string(ad.Data))
			}
		}
	}
}
