// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"testing"

	"github.com/cratis/fundamentals.go/concepts"
)

func FuzzDateOnlyRoundTrip(f *testing.F) {
	for _, text := range []string{"0001-01-01", "9999-12-31", "2024-02-29", "2000-02-29", "1900-02-29", "2023-02-29", "invalid"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		date, err := concepts.ParseDateOnly(text)
		if err != nil {
			return
		}
		parsed, err := concepts.ParseDateOnly(date.String())
		if err != nil || parsed != date {
			t.Fatalf("date round trip %q = %v (%v), want %v", text, parsed, err, date)
		}
		year, month, day := date.Date()
		constructed, err := concepts.NewDateOnly(year, month, day)
		if err != nil || constructed != date {
			t.Fatalf("date construction = %v (%v), want %v", constructed, err, date)
		}
	})
}

func FuzzTimeOnlyRoundTrip(f *testing.F) {
	for _, text := range []string{"00:00:00", "00:00:00.0000001", "23:59:59.9999999", "01:02:03.4", "24:00:00", "12:00:00.12345678", "invalid"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		clock, err := concepts.ParseTimeOnly(text)
		if err != nil {
			return
		}
		parsed, err := concepts.ParseTimeOnly(clock.String())
		if err != nil || parsed != clock {
			t.Fatalf("time round trip %q = %v (%v), want %v", text, parsed, err, clock)
		}
		constructed, err := concepts.NewTimeOnly(clock.Ticks())
		if err != nil || constructed != clock {
			t.Fatalf("time construction = %v (%v), want %v", constructed, err, clock)
		}
	})
}
