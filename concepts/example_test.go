// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
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

func ExampleUUID_Scan() {
	var id concepts.UUID
	if err := id.Scan("00112233-4455-6677-8899-AABBCCDDEEFF"); err != nil {
		fmt.Println(err)
		return
	}
	value, err := id.Value()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(value)
	// Output: 00112233-4455-6677-8899-aabbccddeeff
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

// AuthorID is a nominal UUID value; defined types do not inherit codecs.
type AuthorID concepts.UUID

var _ concepts.Concept[concepts.UUID] = AuthorID{}

func (id AuthorID) ConceptValue() concepts.UUID  { return concepts.UUID(id) }
func (id AuthorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id AuthorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *AuthorID) UnmarshalText(data []byte) error {
	var value concepts.UUID
	if err := value.UnmarshalText(data); err != nil {
		return err
	}
	*id = AuthorID(value)
	return nil
}
func (id *AuthorID) UnmarshalJSON(data []byte) error {
	var value concepts.UUID
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	*id = AuthorID(value)
	return nil
}

func (id AuthorID) Value() (driver.Value, error) { return concepts.UUID(id).Value() }
func (id *AuthorID) Scan(src any) error {
	var value concepts.UUID
	if err := value.Scan(src); err != nil {
		return err
	}
	*id = AuthorID(value)
	return nil
}

// Birthday forwards a calendar scalar with no exposed storage fields.
type Birthday concepts.DateOnly

var _ concepts.Concept[concepts.DateOnly] = Birthday{}

func (d Birthday) ConceptValue() concepts.DateOnly { return concepts.DateOnly(d) }
func (d Birthday) MarshalText() ([]byte, error)    { return concepts.DateOnly(d).MarshalText() }
func (d Birthday) MarshalJSON() ([]byte, error)    { return concepts.DateOnly(d).MarshalJSON() }
func (d *Birthday) UnmarshalText(data []byte) error {
	var value concepts.DateOnly
	if err := value.UnmarshalText(data); err != nil {
		return err
	}
	*d = Birthday(value)
	return nil
}
func (d *Birthday) UnmarshalJSON(data []byte) error {
	var value concepts.DateOnly
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	*d = Birthday(value)
	return nil
}

func ExampleConcept() {
	value, err := concepts.ParseUUID("00112233-4455-6677-8899-aabbccddeeff")
	if err != nil {
		fmt.Println(err)
		return
	}
	id := AuthorID(value)
	data, err := json.Marshal(id)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
	// Output: "00112233-4455-6677-8899-aabbccddeeff"
}

func ExampleUnderlying() {
	r, ok, err := concepts.Underlying(reflect.TypeFor[**AuthorID]())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(ok, r.Type, r.Declared, r.PointerDepth)
	// Output: true concepts.UUID concepts_test.AuthorID 2
}

func ExampleCheckJSON() {
	r, ok, err := concepts.Underlying(reflect.TypeFor[AuthorID]())
	if err != nil || !ok {
		fmt.Println(err)
		return
	}
	data, err := json.Marshal(AuthorID{})
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := concepts.CheckJSON(r, data); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
	// Output: "00000000-0000-0000-0000-000000000000"
}
