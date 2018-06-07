package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
)

// Command line settings
type settings struct {
	device string
}

// Command line settings from user
var cmdline settings

func init() {
	flag.StringVar(&cmdline.device, "device", "", "HCI device to use")
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
	}
	host.Deinit()
}
