package hci

import "testing"

func TestFromString(t *testing.T) {

	testdata := []struct {
		name  string
		val   string
		valid bool
	}{
		{
			"valid", "00:11:22:33:44:55", true,
		},
		{
			"valid2", "11:aa:bb:dd:22:33", true,
		},
		{
			"short", "00:11", false,
		},
		{
			"empty", "", false,
		},
		{
			"invalid number", "00:11:2:33:44:55", false,
		},
		{
			"invalid char", "00:aa:bb:vv:33:44", false,
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			addr, err := BtAddressFromString(test.val)
			if test.valid {
				if err != nil {
					t.Errorf("Unexpected error while parsing address: %s", err.Error())
				} else if addr.String() != test.val {
					t.Errorf("Parsed address did not match, expected %s, got %s", addr.String(), test.val)
				}
			} else {
				if err == nil {
					t.Errorf("Parsed invalid address")
				}
			}
		})
	}
}
