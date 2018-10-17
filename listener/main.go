package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

var path string
var ruuviMode bool

func init() {
	flag.StringVar(&path, "unix", "", "Unix socket path to listen for connections")
	flag.BoolVar(&ruuviMode, "ruuvi", false, "Expect to receive ruuvi data")
}

func main() {
	flag.Parse()

	if path == "" {
		fmt.Fprintf(os.Stderr, "Unix socket path not set (use -unix <path>)\n")
		os.Exit(255)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start listening in %s : %s\n", path, err.Error())
		os.Exit(255)
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while accepting connection: %s \n", err.Error())
		os.Exit(255)
	}
	defer conn.Close()

	rd := bufio.NewReader(conn)

	for line, err := rd.ReadString('\n'); err == nil; line, err = rd.ReadString('\n') {

		//fmt.Printf("Received \"%s\"", line)
		if ruuviMode {
			ruuvi := struct {
				Device hci.BtAddress `json:"device"`
				Rssi   int8          `json:"rssi"`
				Values *ruuvi.Data   `json:"sensors"`
			}{Device: hci.BtAddress{}, Values: &ruuvi.Data{}}
			jerr := json.Unmarshal([]byte(line), &ruuvi)
			if jerr != nil {
				fmt.Fprintf(os.Stderr, "Error while reading ruuvi data: %s\n", jerr.Error())
				continue
			}
			fmt.Printf("Got: %+v\n", ruuvi)
			fmt.Printf("Values is: %+v\n", ruuvi.Values)
		} else {
			devices := []struct {
				Structures []*hci.AdStructure `json:"data"`
				LastSeen   time.Time          `json:"last"`
				Rssi       int8               `json:"rssi"`
				Types      []hci.AdvType      `json:"types"`
				Device     hci.BtAddress      `json:"device"`
			}{}
			if jerr := json.Unmarshal([]byte(line), &devices); err != nil {
				fmt.Fprintf(os.Stderr, "Error while reading found devices data: %s\n", jerr.Error())
				continue
			}
			fmt.Printf("Got: %+v\n", devices)
		}
	}
	fmt.Printf("Shutting down")
}
