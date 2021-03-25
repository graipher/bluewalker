package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
)

func parseAddress(addr string) (hci.BtAddress, error) {
	atype := hci.LePublicAddress
	if strings.Contains(addr, ",") {
		parts := strings.Split(addr, ",")
		if len(parts) != 2 {
			return hci.BtAddress{}, fmt.Errorf("invalid address specification %q", addr)
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
			return hci.BtAddress{}, fmt.Errorf("invalid address type %q", parts[1])
		}
		addr = parts[0]
	}
	addr = strings.TrimSpace(addr)
	baddr, err := hci.BtAddressFromString(addr)
	if err != nil {
		return hci.BtAddress{}, err
	}
	baddr.Atype = atype
	return baddr, nil
}

//parseAddressFilters parses one or more address filters from given
//input from command line options
func parseAddressFilters(addresses string) (filter.AdFilter, error) {

	addrs := strings.Split(addresses, ";")
	parsed := make([]filter.AdFilter, len(addrs))
	for i, addr := range addrs {
		baddr, err := parseAddress(addr)
		if err != nil {
			return nil, err
		}
		parsed[i] = filter.ByAddress(baddr)
	}
	return filter.Any(parsed), nil
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
		return nil, fmt.Errorf("expected %d bytes, got %d", length, len(bytes))
	}
	return bytes, nil
}

func parsePartialAddrFilter(data string) (filter.AdFilter, error) {

	var bytes []byte
	data = strings.TrimSpace(data)
	if strings.HasSuffix(data, ":") {
		// remove trailing ':' first
		data = data[0 : len(data)-1]
	}
	if strings.Contains(data, ":") {
		// Assuming the data is in BD_ADDR format
		parts := strings.Split(data, ":")
		if len(parts) > 6 {
			return nil, fmt.Errorf("invalid partial address filter %s", data)
		}
		bytes = make([]byte, len(parts))
		for i := 0; i < len(parts); i++ {
			bb, err := hex.DecodeString(parts[i])
			if err != nil {
				return nil, fmt.Errorf("invalid partial address filter %s", data)
			}
			if len(bb) != 1 {
				return nil, fmt.Errorf("invalid partial address filter %s", data)
			}
			bytes[i] = bb[0]
		}
	} else {
		// assuming just byte array in hex
		var err error
		bytes, err = parseByteArray(data, -1)
		if err != nil {
			return nil, fmt.Errorf("invalid partial address filter %s (%v)", data, err)
		}
	}

	if len(bytes) > 6 {
		return nil, fmt.Errorf("too long prefix %d bytes, max is 6", len(bytes))
	}
	return filter.ByPartialAddress(bytes), nil
}

//parseIrkFilter parses IRK filter from IRK given as command line parameter
func parseIrkFilter(data string) (filter.AdFilter, error) {
	bytes, err := parseByteArray(data, hci.IrkLength)
	if err != nil {
		return nil, fmt.Errorf("invalid IRK data (%v)", err)
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
		return nil, fmt.Errorf("invalid vendor specific data specification (%v)", err)
	}
	return filter.ByVendor(bytes), nil
}

//parseAdTypeFilters parses one or more filters for AD types from command
//line parameters
func parseAdTypeFilters(types string) (filter.AdFilter, error) {

	parts := strings.Split(types, ",")
	filters := make([]filter.AdFilter, len(parts))
	for i, part := range parts {
		data, err := parseByteArray(part, 1)
		if err != nil {
			return nil, fmt.Errorf("invalid Ad Type value %q (%v)", part, err)
		}
		filters[i] = filter.ByAdType(hci.AdType(data[0]))
	}
	return filter.Any(filters), nil
}

func parseAdDataFilters(ads string) (filter.AdFilter, error) {
	parts := strings.Split(ads, ";")
	filters := make([]filter.AdFilter, len(parts))
	for i, part := range parts {
		t, d, err := parseAdStructure(part)
		if err != nil {
			return nil, err
		}
		filters[i] = filter.ByAdData(t, d)
	}
	return filter.All(filters), nil
}

func parseAdStructure(adstruct string) (hci.AdType, []byte, error) {
	parts := strings.Split(adstruct, ",")
	if len(parts) != 2 {
		return 0, nil, fmt.Errorf("expected Ad Structure as \"<type>,<data>\"")
	}
	typ, err := parseByteArray(parts[0], 1)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid Ad Structure type %s (%s)", parts[0], err.Error())
	}
	data, err := parseByteArray(parts[1], -1)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid value for Ad Structure data (%s)", err.Error())
	}
	return hci.AdType(typ[0]), data, nil
}

//parseAdStructures parses one or more Ad Structures from command line parameters
func parseAdStructures(structs string) ([]*hci.AdStructure, error) {

	ads := strings.Split(structs, ";")
	ret := make([]*hci.AdStructure, 0)
	for _, ad := range ads {
		t, d, err := parseAdStructure(ad)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &hci.AdStructure{Typ: t, Data: d})
	}
	return ret, nil
}
