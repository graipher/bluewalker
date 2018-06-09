package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
)

// Command line settings
type settings struct {
	device string
	active bool
}

// Command line settings from user
var cmdline settings

func init() {
	flag.StringVar(&cmdline.device, "device", "", "HCI device to use")
	flag.BoolVar(&cmdline.active, "active", false, "Active scanning")
}

func main() {

	flag.Parse()
	if cmdline.device == "" {
		fmt.Printf("Missing device name\n")
		os.Exit(255)
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

	collected := make(map[hci.BtAddress][]*hci.AdStructure)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for sr := range reportChan {
			structs, found := collected[sr.Address]
			if !found {
				structs = sr.Data
				collected[sr.Address] = structs
			} else {
				for _, ads := range sr.Data {
					structs = append(structs, ads)
					collected[sr.Address] = structs
				}
			}
		}
		wg.Done()
	}()

	ch := time.Tick(5 * time.Second)
	<-ch
	host.StopScanning()
	host.Deinit()
	wg.Wait()

	fmt.Printf("Found %d devices:\n", len(collected))
	for key, val := range collected {
		fmt.Printf("Device %s:\n", key.String())
		for _, ad := range val {
			fmt.Printf("\t%s\n", ad.String())
			if ad.Typ == hci.AdCompleteLocalName || ad.Typ == hci.AdShortenedLocalName {
				fmt.Printf("\t\tName: \"%s\"\n", string(ad.Data))
			}
		}
	}
}
