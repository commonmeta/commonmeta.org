package main

import (
	"testing"
)

func TestGetDateFromDateParts(t *testing.T) {
	t.Parallel()

	type testCase struct {
		got  []DateParts
		want string
		err  error
	}

	testCases := []testCase{
		{got: [[2012, 1, 1]], want: "2012-01-01", err: nil},
		{got: [[2012, 1]], want: "2012-01", err: nil},
		{got: [[2012]], want: "2012", err: nil},
	}
	for _, tc := range testCases {
		got, err := main.GetDateFromDateParts(tc.got)
		if tc.want != got{
			t.Errorf("Get date from date parts(%v): want %v, error %v",
				tc.got, tc.want, err)
		}
	}
}

func TestGetDateFromParts(t *testing.T) {
	t.Parallel()

	type testCase struct {
		got  []int
		want string
		err  error
	}

	testCases := []testCase{
		{got: [2012, 1, 1], want: "2012-01-01", err: nil},
		{got: [2012, 1], want: "2012-01", err: nil},
		{got: [2012], want: "2012", err: nil},
	}
	for _, tc := range testCases {
		got, err := main.GetDateFromParts(tc.got)
		if tc.want != got{
			t.Errorf("Get date from date parts(%v): want %v, error %v",
				tc.got, tc.want, err)
		}
	}
}
