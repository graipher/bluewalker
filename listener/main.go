package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

var path string
var ruuviMode bool
var observer bool

func init() {
	flag.StringVar(&path, "unix", "", "Unix socket path to listen for connections")
	flag.BoolVar(&ruuviMode, "ruuvi", false, "Expect to receive ruuvi data")
	flag.BoolVar(&observer, "observer", false, "Start listener in observer mode")
}

type device struct {
	Structures []*hci.AdStructure `json:"data"`
	LastSeen   time.Time          `json:"last"`
	Rssi       int8               `json:"rssi"`
	Types      []hci.AdvType      `json:"types"`
	Device     hci.BtAddress      `json:"device"`
}

func (dev *device) String() string {

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Device: %s RSSI: %d \n", dev.Device.String(), dev.Rssi))
	sb.WriteString("Adv types: ")
	for _, t := range dev.Types {
		sb.WriteString(fmt.Sprintf("%s ", t.String()))
	}
	sb.WriteString("\n")
	sb.WriteString("Adv Data:\n")
	for _, s := range dev.Structures {
		sb.WriteString(fmt.Sprintf("\t%s\n", s.String()))
	}
	sb.WriteString("\n")
	return sb.String()
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

			var data interface{}
			var printer func(interface{})
			if !observer {
				data = &[]*device{}
				printer = func(in interface{}) {
					list := in.(*[]*device)
					for _, l := range *list {
						fmt.Printf("%s", l.String())
					}
				}
			} else {
				data = &device{}
				printer = func(in interface{}) {
					d := in.(*device)
					fmt.Printf("%s", d.String())
				}
			}
			if jerr := json.Unmarshal([]byte(line), data); err != nil {
				fmt.Fprintf(os.Stderr, "Error while reading found devices data: %s\n", jerr.Error())
				continue
			}
			printer(data)
		}
	}
	fmt.Printf("Shutting down")
}
