package parseutil

import (
	"encoding/json"
	"testing"
	"time"
)

func Test_ParseCapacityString(t *testing.T) {
	testCases := []struct {
		name     string
		inp      interface{}
		valid    bool
		expected uint64
	}{
		{
			"bare number as an int",
			5,
			true,
			uint64(5),
		},
		{
			"bare number as a float",
			5.0,
			true,
			uint64(5),
		},
		{
			"bare number as a string",
			"5",
			true,
			uint64(5),
		},
		{
			"string",
			"haha",
			false,
			uint64(0),
		},
		{
			"random data structure",
			struct{}{},
			false,
			uint64(0),
		},
		{
			"kb",
			"5kb",
			true,
			uint64(5000),
		},
		{
			"kib",
			"5kib",
			true,
			uint64(5120),
		},
		{
			"KB",
			"5KB",
			true,
			uint64(5000),
		},
		{
			"KIB",
			"5KIB",
			true,
			uint64(5120),
		},
		{
			"kB",
			"5kB",
			true,
			uint64(5000),
		},
		{
			"Kb",
			"5Kb",
			true,
			uint64(5000),
		},
		{
			"space kb",
			"5 kb",
			true,
			uint64(5000),
		},
		{
			"space KB",
			"5 KB",
			true,
			uint64(5000),
		},
		{
			"kb surrounding spaces",
			" 5 kb ",
			true,
			uint64(5000),
		},
		{
			"mb",
			"5mb",
			true,
			uint64(5000000),
		},
		{
			"mib",
			"5mib",
			true,
			uint64(5242880),
		},
		{
			"gb",
			"5gb",
			true,
			uint64(5000000000),
		},
		{
			"gib",
			"5gib",
			true,
			uint64(5368709120),
		},
		{
			"tb",
			"5tb",
			true,
			uint64(5000000000000),
		},
		{
			"tib",
			"5tib",
			true,
			uint64(5497558138880),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outp, err := ParseCapacityString(tc.inp)
			if tc.valid && err != nil {
				t.Errorf("failed to parse: %v. err: %v", tc.inp, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("no error for: %v", tc.inp)
			}
			if outp != tc.expected {
				t.Errorf("input %v parsed as %v, expected %v", tc.inp, outp, tc.expected)
			}
		})
	}
}

func Test_ParseDurationSecond(t *testing.T) {
	type Test struct {
		in      interface{}
		out     time.Duration
		invalid bool
	}

	tests := []Test{
		// Numeric inputs
		{in: 9876, out: 9876 * time.Second},
		{in: 5.5, out: 5 * time.Second},
		{in: 5, out: 5 * time.Second},
		{in: 0.9, out: 0 * time.Second},
		{in: -5, out: -5 * time.Second},

		// String inputs
		{in: "9876", out: 9876 * time.Second},
		{in: "9876s", out: 9876 * time.Second},
		{in: "50ms", out: 50 * time.Millisecond},
		{in: "0.5m", out: 30 * time.Second},
		{in: "0.5s", out: 500 * time.Millisecond},
		{in: "5m", out: 5 * time.Minute},
		{in: "6h", out: 6 * time.Hour},
		{in: "5d", out: 5 * 24 * time.Hour},
		{in: "-5d", out: -5 * 24 * time.Hour},
		{in: "05d", out: 5 * 24 * time.Hour},
		{in: "500d", out: 500 * 24 * time.Hour},

		// JSON Number inputs
		{in: json.Number("4352s"), out: 4352 * time.Second},
	}

	// Invalid inputs
	for _, s := range []string{
		"5 s",
		"5sa",
		" 5m",
		"5h ",
		"5days",
		"9876q",
		"s20ms",
		"10S",
		"ad",
		"0.5d",
		"1.5d",
		"d",
		"4世",
		"s",
		"m",
	} {
		tests = append(tests, Test{
			in:      s,
			invalid: true,
		})
	}

	for _, test := range tests {
		out, err := ParseDurationSecond(test.in)
		if test.invalid {
			if err == nil {
				t.Fatalf("%q: expected error, got nil", test.in)
			}
			continue
		}

		if err != nil {
			t.Fatal(err)
		}

		if out != test.out {
			t.Fatalf("%q: expected: %q, got: %q", test.in, test.out, out)
		}
	}
}

func Test_ParseAbsoluteTime(t *testing.T) {
	testCases := []struct {
		inp      interface{}
		valid    bool
		expected time.Time
	}{
		{
			"2020-12-11T09:08:07.654321Z",
			true,
			time.Date(2020, 12, 11, 9, 8, 7, 654321000, time.UTC),
		},
		{
			"2020-12-11T09:08:07+02:00",
			true,
			time.Date(2020, 12, 11, 7, 8, 7, 0, time.UTC),
		},
		{
			"2021-12-11T09:08:07Z",
			true,
			time.Date(2021, 12, 11, 9, 8, 7, 0, time.UTC),
		},
		{
			"2021-12-11T09:08:07",
			false,
			time.Time{},
		},
		{
			"1670749687",
			true,
			time.Date(2022, 12, 11, 9, 8, 7, 0, time.UTC),
		},
		{
			1670749687,
			true,
			time.Date(2022, 12, 11, 9, 8, 7, 0, time.UTC),
		},
		{
			uint32(1670749687),
			true,
			time.Date(2022, 12, 11, 9, 8, 7, 0, time.UTC),
		},
		{
			json.Number("1670749687"),
			true,
			time.Date(2022, 12, 11, 9, 8, 7, 0, time.UTC),
		},
		{
			nil,
			true,
			time.Time{},
		},
		{
			struct{}{},
			false,
			time.Time{},
		},
		{
			true,
			false,
			time.Time{},
		},
	}
	for _, tc := range testCases {
		outp, err := ParseAbsoluteTime(tc.inp)
		if err != nil {
			if tc.valid {
				t.Errorf("failed to parse: %v", tc.inp)
			}
			continue
		}
		if err == nil && !tc.valid {
			t.Errorf("no error for: %v", tc.inp)
			continue
		}
		if !outp.Equal(tc.expected) {
			t.Errorf("input %v parsed as %v, expected %v", tc.inp, outp, tc.expected)
		}
	}
}

func Test_ParseBool(t *testing.T) {
	outp, err := ParseBool("true")
	if err != nil {
		t.Fatal(err)
	}
	if !outp {
		t.Fatal("wrong output")
	}
	outp, err = ParseBool(1)
	if err != nil {
		t.Fatal(err)
	}
	if !outp {
		t.Fatal("wrong output")
	}
	outp, err = ParseBool(true)
	if err != nil {
		t.Fatal(err)
	}
	if !outp {
		t.Fatal("wrong output")
	}
}

func equalInt64Slice(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func Test_ParseIntSlice(t *testing.T) {
	// Handles testing of ParseInt, ParseDirectIntSlice, and ParseIntSlice.
	testCases := []struct {
		inp      interface{}
		valid    bool
		expected []int64
	}{
		// ParseInt
		{
			int(-1),
			true,
			[]int64{-1},
		},
		{
			int32(-1),
			true,
			[]int64{-1},
		},
		{
			int64(-1),
			true,
			[]int64{-1},
		},
		{
			uint(1),
			true,
			[]int64{1},
		},
		{
			uint32(1),
			true,
			[]int64{1},
		},
		{
			uint64(1),
			true,
			[]int64{1},
		},
		{
			json.Number("1"),
			true,
			[]int64{1},
		},
		{
			"1",
			true,
			[]int64{1},
		},
		// ParseDirectIntSlice
		{
			[]int{1, -2, 3},
			true,
			[]int64{1, -2, 3},
		},
		{
			[]int32{1, -2, 3},
			true,
			[]int64{1, -2, 3},
		},
		{
			[]int64{1, -2, 3},
			true,
			[]int64{1, -2, 3},
		},
		{
			[]uint{1, 2, 3},
			true,
			[]int64{1, 2, 3},
		},
		{
			[]uint32{1, 2, 3},
			true,
			[]int64{1, 2, 3},
		},
		{
			[]uint64{1, 2, 3},
			true,
			[]int64{1, 2, 3},
		},
		{
			[]json.Number{json.Number("1"), json.Number("2"), json.Number("3")},
			true,
			[]int64{1, 2, 3},
		},
		{
			[]string{"1", "2", "3"},
			true,
			[]int64{1, 2, 3},
		},
		// Comma separated list
		{
			"1",
			true,
			[]int64{1},
		},
		{
			"1,",
			true,
			[]int64{1},
		},
		{
			",1",
			true,
			[]int64{1},
		},
		{
			",1,",
			true,
			[]int64{1},
		},
		{
			"1,2",
			true,
			[]int64{1,2},
		},
		{
			"1,2,3",
			true,
			[]int64{1,2,3},
		},
	}

	for _, tc := range testCases {
		outp, err := ParseIntSlice(tc.inp)
		if err != nil {
			if tc.valid {
				t.Errorf("failed to parse: %v", tc.inp)
			}
			continue
		}
		if err == nil && !tc.valid {
			t.Errorf("no error for: %v", tc.inp)
			continue
		}
		if !equalInt64Slice(outp, tc.expected) {
			t.Errorf("input %v parsed as %v, expected %v", tc.inp, outp, tc.expected)
		}
	}
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func Test_ParseCommaStringSlice(t *testing.T) {
	cases := []struct{
		name     string
		inp       interface{}
		expected []string
		valid    bool
	}{
		{
			"empty string",
			"",
			[]string{},
			true,
		},
		{
			"single value",
			"foo",
			[]string{"foo"},
			true,
		},
		{
			"multiple values",
			"foo,bar,baz",
			[]string{"foo", "bar", "baz"},
			true,
		},
		{
			"multiple values with trim",
			"  foo ,    bar   ,baz  ",
			[]string{"foo", "bar", "baz"},
			true,
		},
		{
			"json number",
			json.Number("123"),
			nil,
			false,
		},
		{
			"string slice",
			[]string{"foo", "bar", "baz"},
			nil,
			false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outp, err := ParseCommaStringSlice(tc.inp)
			if tc.valid && err != nil {
				t.Errorf("failed to parse: %v. err: %v", tc.inp, err)
			}

			if !tc.valid && err == nil {
				t.Errorf("no error for: %v", tc.inp)
			}

			if !equalStringSlice(outp, tc.expected) {
				t.Errorf("input %v parsed as %v, expected %v", tc.inp, outp, tc.expected)
			}

		})
	}
}
