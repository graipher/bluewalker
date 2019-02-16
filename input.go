package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
)

//parseAddressFilters parses one or more address filters from given
//input from command line options
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
			parts[1] = strings.TrimSpace(parts[1])
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
		addr = strings.TrimSpace(addr)
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

func parseByteArray(input string, length int) ([]byte, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "0x") {
		input = input[2:]
	}
	bytes, err := hex.DecodeString(input)
	if err != nil {
		return nil, err
	}
	if length > 0 && len(bytes) != length {
		return nil, fmt.Errorf("Expected %d bytes, got %d", length, len(bytes))
	}
	return bytes, nil
}

//parseIrkFilter parses IRK filter from IRK given as command line parameter
func parseIrkFilter(data string) (filter.AdFilter, error) {
	bytes, err := parseByteArray(data, hci.IrkLength)
	if err != nil {
		return nil, fmt.Errorf("Invalid IRK data (%v)", err)
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

//parseVendorSpecFilter parses filter for vendor specific data from
//command line parameter
func parseVendorSpecFilter(data string) (filter.AdFilter, error) {

	bytes, err := parseByteArray(data, -1)
	if err != nil {
		return nil, fmt.Errorf("Invalid vendor specific data specification (%v)", err)
	}
	return filter.ByVendor(bytes), nil
}

//parseAdTypeFilters parses one or more filters for AD types from command
//line parameters
func parseAdTypeFilters(types string) ([]filter.AdFilter, error) {

	parts := strings.Split(types, ",")
	filters := make([]filter.AdFilter, len(parts))
	for i, part := range parts {
		data, err := parseByteArray(part, 1)
		if err != nil {
			return nil, fmt.Errorf("Invalid Ad Type value %q (%v)", part, err)
		}
		filters[i] = filter.ByAdType(hci.AdType(data[0]))
	}
	return filters, nil
}
