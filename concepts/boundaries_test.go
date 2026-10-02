// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"encoding/hex"
	"math"
	"testing"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
)

func TestUUIDAsymmetricRFCBytes(t *testing.T) {
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-AABBCCDDEEFF")
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(id[:]); got != "00112233445566778899aabbccddeeff" {
		t.Fatalf("RFC bytes = %s", got)
	}
	if got := id.String(); got != "00112233-4455-6677-8899-aabbccddeeff" {
		t.Fatalf("canonical UUID = %s", got)
	}
	for _, text := range []string{
		"00112233445566778899aabbccddeeff",
		"{00112233-4455-6677-8899-aabbccddeeff}",
		"(00112233-4455-6677-8899-aabbccddeeff)",
		"{0x00112233,0x4455,0x6677,{0x88,0x99,0xaa,0xbb,0xcc,0xdd,0xee,0xff}}",
		" 00112233-4455-6677-8899-aabbccddeeff ",
		"00112233-4455-6677-8899-aabbccddeefg",
	} {
		if _, err := concepts.ParseUUID(text); err == nil {
			t.Errorf("accepted UUID %q", text)
		}
	}
}

func TestDateOnlyConstructorBoundaries(t *testing.T) {
	for _, tc := range []struct {
		year  int
		month time.Month
		day   int
		text  string
	}{
		{1, time.January, 1, "0001-01-01"},
		{9999, time.December, 31, "9999-12-31"},
		{2000, time.February, 29, "2000-02-29"},
		{2024, time.February, 29, "2024-02-29"},
	} {
		t.Run(tc.text, func(t *testing.T) {
			date, err := concepts.NewDateOnly(tc.year, tc.month, tc.day)
			if err != nil {
				t.Fatal(err)
			}
			if date.String() != tc.text {
				t.Fatalf("date = %s, want %s", date, tc.text)
			}
			parsed, err := concepts.ParseDateOnly(tc.text)
			if err != nil || parsed != date {
				t.Fatalf("parsed date = %v (%v), want %v", parsed, err, date)
			}
		})
	}
	for _, tc := range []struct {
		year  int
		month time.Month
		day   int
	}{
		{0, time.January, 1}, {10000, time.January, 1},
		{2024, 0, 1}, {2024, 13, 1},
		{2024, time.January, 0}, {2024, time.January, 32},
		{1900, time.February, 29}, {2023, time.February, 29},
		{2024, time.April, 31},
	} {
		if _, err := concepts.NewDateOnly(tc.year, tc.month, tc.day); err == nil {
			t.Errorf("accepted date %d-%d-%d", tc.year, tc.month, tc.day)
		}
	}
	if _, err := concepts.ParseDateOnly("2024-02-29T12:00:00Z"); err == nil {
		t.Fatal("accepted timestamp as date")
	}
	var zero concepts.DateOnly
	if year, month, day := zero.Date(); year != 1 || month != time.January || day != 1 {
		t.Fatalf("zero date = %d-%d-%d", year, month, day)
	}
}

func TestTimeOnlyTickBoundaries(t *testing.T) {
	for _, tc := range []struct {
		ticks int64
		text  string
	}{
		{0, "00:00:00.0000000"},
		{1, "00:00:00.0000001"},
		{37_231_234_567, "01:02:03.1234567"},
		{863_999_999_999, "23:59:59.9999999"},
	} {
		t.Run(tc.text, func(t *testing.T) {
			clock, err := concepts.NewTimeOnly(tc.ticks)
			if err != nil {
				t.Fatal(err)
			}
			if clock.Ticks() != tc.ticks || clock.String() != tc.text {
				t.Fatalf("time = %d / %s", clock.Ticks(), clock)
			}
			parsed, err := concepts.ParseTimeOnly(tc.text)
			if err != nil || parsed != clock {
				t.Fatalf("parsed time = %v (%v), want %v", parsed, err, clock)
			}
		})
	}
	clock, err := concepts.ParseTimeOnly("00:00:00")
	if err != nil || clock != (concepts.TimeOnly{}) {
		t.Fatalf("midnight = %v (%v)", clock, err)
	}
	for _, ticks := range []int64{math.MinInt64, -1, 864_000_000_000, math.MaxInt64} {
		if _, err := concepts.NewTimeOnly(ticks); err == nil {
			t.Errorf("accepted time ticks %d", ticks)
		}
	}
	if _, err := concepts.ParseTimeOnly("12:00 PM"); err == nil {
		t.Fatal("accepted culture-dependent time")
	}
}

func TestTimeSpanSignedTickBoundaries(t *testing.T) {
	for _, tc := range []struct {
		ticks int64
		text  string
	}{
		{math.MinInt64, "-10675199.02:48:05.4775808"},
		{math.MaxInt64, "10675199.02:48:05.4775807"},
		{-1, "-00:00:00.0000001"},
		{-937_841_234_567, "-1.02:03:04.1234567"},
		{-864_000_000_000, "-1.00:00:00"},
		{0, "00:00:00"},
	} {
		t.Run(tc.text, func(t *testing.T) {
			span, err := concepts.ParseTimeSpan(tc.text)
			if err != nil {
				t.Fatal(err)
			}
			if span.Ticks() != tc.ticks || span.String() != tc.text {
				t.Fatalf("duration = %d / %s, want %d / %s", span.Ticks(), span, tc.ticks, tc.text)
			}
		})
	}
}
