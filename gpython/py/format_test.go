package py

import (
	"math"
	"testing"
)

func TestFormat(t *testing.T) {
	for _, tc := range []struct {
		value      Object
		spec, want string
	}{
		{Float(1.234567), ".4f", "1.2346"},
		{Float(math.Copysign(0, -1)), ".2f", "-0.00"},
		{Float(1), "#", "1.0"},
		{Float(1), ".1", "1e+00"},
		{Float(1), ".2", "1.0"},
		{Float(1), "#.0", "1.e+00"},
		{Float(1), "08,", "00,001.0"},
		{Float(1234), "010,", "0,001,234.0"},
		{Float(0.125), ".1%", "12.5%"},
		{Float(12.5), "+.2f", "+12.50"},
		{Float(12.5), ".2e", "1.25e+01"},
		{Int(-12), "05d", "-0012"},
		{Int(255), "#06x", "0x00ff"},
		{Int(0xdeadbeef), "_x", "dead_beef"},
		{Int(1234), "010,d", "00,001,234"},
		{Int(1234), "08,d", "0,001,234"},
		{Int(65), "c", "A"},
		{True, "04d", "0001"},
		{String("é"), ">3s", "  é"},
		{String("ééé"), ".2s", "éé"},
		{String("cat"), "*^7", "**cat**"},
	} {
		got, err := Format(tc.value, String(tc.spec))
		if err != nil || got != String(tc.want) {
			t.Errorf("format(%v, %q) = %v, %v; want %q", tc.value, tc.spec, got, err, tc.want)
		}
	}
}

func TestInvalidFormat(t *testing.T) {
	for _, tc := range []struct {
		value Object
		spec  string
	}{
		{Int(3), ".2d"}, {Int(3), "q"}, {Int(3), ",x"},
		{Float(3), "d"}, {Float(3), ".f"},
		{String("x"), "+s"}, {String("x"), "=3"}, {String("x"), "#s"},
		{Int(3), "999999999999999999999d"},
	} {
		if _, err := Format(tc.value, String(tc.spec)); !IsException(ValueError, err) {
			t.Errorf("format(%v, %q): want ValueError, got %v", tc.value, tc.spec, err)
		}
	}
}
