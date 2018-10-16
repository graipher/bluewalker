package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

var path string

func init() {
	flag.StringVar(&path, "unix", "", "Unix socket path to listen for connections")

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

		fmt.Printf("Received \"%s\"", line)
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
	}
	fmt.Printf("Shutting down")
}
