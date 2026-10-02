// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"fmt"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
)

func ExampleUUID() {
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-AABBCCDDEEFF")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(id)
	fmt.Printf("%x\n", [16]byte(id))
	// Output:
	// 00112233-4455-6677-8899-aabbccddeeff
	// 00112233445566778899aabbccddeeff
}

func ExampleDateOnly() {
	date, err := concepts.NewDateOnly(2024, time.February, 29)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(date)
	// Output: 2024-02-29
}

func ExampleTimeOnly() {
	clock, err := concepts.NewTimeOnly(37_231_234_567)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(clock)
	fmt.Println(clock.Ticks())
	// Output:
	// 01:02:03.1234567
	// 37231234567
}

func ExampleTimeSpan() {
	span, err := concepts.ParseTimeSpan("-1.02:03:04.1234567")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(span)
	fmt.Println(span.Ticks())
	// Output:
	// -1.02:03:04.1234567
	// -937841234567
}
