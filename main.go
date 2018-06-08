package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

	if err := host.StartScanning(cmdline.active); err != nil {
		log.Printf("Unable to start scanning: %s", err.Error())
		host.Deinit()
		os.Exit(255)
	}
	ch := time.Tick(5 * time.Second)
	<-ch
	host.StopScanning()
	host.Deinit()
}
