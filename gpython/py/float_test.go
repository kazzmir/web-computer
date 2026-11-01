package py

import (
	"math"
	"math/big"
	"testing"
)

func TestFloatRound(t *testing.T) {
	for _, test := range []struct {
		value  Float
		digits Int
		want   Float
	}{
		{2.675, 2, 2.67}, {1.25, 1, 1.2}, {1.35, 1, 1.4}, {-1.25, 1, -1.2},
		{1.23456, 3, 1.235}, {15, -1, 20}, {25, -1, 20}, {-15, -1, -20}, {-25, -1, -20},
		{1.2, 7, 1.2}, {1e308, -309, 0},
	} {
		got, err := test.value.M__round__(test.digits)
		if err != nil || got != test.want {
			t.Errorf("round(%v, %v) = %v, %v; want %v", test.value, test.digits, got, err, test.want)
		}
	}
	for _, test := range []struct {
		value Float
		want  Int
	}{{2.5, 2}, {3.5, 4}, {-2.5, -2}, {-3.5, -4}} {
		got, err := test.value.M__round__(None)
		if err != nil || got != test.want {
			t.Errorf("round(%v) = %v, %v; want int %d", test.value, got, err, test.want)
		}
	}
	large, err := Float(math.Exp2(63)).M__round__(None)
	if err != nil {
		t.Fatal(err)
	}
	if integer, ok := large.(*BigInt); !ok || (*big.Int)(integer).Cmp(new(big.Int).Lsh(big.NewInt(1), 63)) != 0 {
		t.Errorf("large round = %v, want 2**63", large)
	}
	for _, digits := range []Object{Int(0), Int(-309)} {
		got, err := Float(-0.1).M__round__(digits)
		if err != nil || !math.Signbit(float64(got.(Float))) || got != Float(0) {
			t.Errorf("negative zero = %v, %v", got, err)
		}
	}
	if _, err := Float(1.2).M__round__(Float(1.0)); !IsException(TypeError, err) {
		t.Errorf("non-integer digits returned %v", err)
	}
	if _, err := Float(1.7e308).M__round__(Int(-308)); !IsException(OverflowError, err) {
		t.Errorf("overflow returned %v", err)
	}
	for _, test := range []struct {
		value Float
		want  *Type
	}{{Float(math.NaN()), ValueError}, {Float(math.Inf(1)), OverflowError}} {
		if _, err := test.value.M__round__(None); !IsException(test.want, err) {
			t.Errorf("round(%v) returned %v", test.value, err)
		}
		got, err := test.value.M__round__(Int(2))
		if err != nil {
			t.Fatal(err)
		}
		if actual := float64(got.(Float)); !((math.IsNaN(actual) && math.IsNaN(float64(test.value))) || actual == float64(test.value)) {
			t.Errorf("non-finite round = %v", actual)
		}
	}
	huge := (*BigInt)(new(big.Int).Lsh(big.NewInt(1), 100))
	if got, err := Float(1.25).M__round__(huge); err != nil || got != Float(1.25) {
		t.Errorf("huge digits = %v, %v", got, err)
	}
	negative := (*BigInt)(new(big.Int).Neg((*big.Int)(huge)))
	if got, err := Float(1.25).M__round__(negative); err != nil || got != Float(0) {
		t.Errorf("huge negative digits = %v, %v", got, err)
	}
}
