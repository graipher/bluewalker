package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
)

// Command line settings
type settings struct {
	device   string
	active   bool
	duration int
	debug    bool
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

	reportChan, err := host.StartScanning(cmdline.active)
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
		fmt.Printf("Device %s (RSSI:%d dBm; last seen %s):\n", key.String(), val.rssi, val.lastSeen.Format(time.Stamp))
		for _, ad := range val.structures {
			fmt.Printf("\t%s\n", ad.String())
			if ad.Typ == hci.AdCompleteLocalName || ad.Typ == hci.AdShortenedLocalName {
				fmt.Printf("\t\tName: \"%s\"\n", string(ad.Data))
			}
		}
	}
}
