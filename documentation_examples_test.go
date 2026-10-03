// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/correlation"
)

// AuthorID identifies an author without making arbitrary UUIDs interchangeable.
type AuthorID concepts.UUID

var _ concepts.Concept[concepts.UUID] = AuthorID{}

func (id AuthorID) ConceptValue() concepts.UUID  { return concepts.UUID(id) }
func (id AuthorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id AuthorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *AuthorID) UnmarshalText(data []byte) error {
	return (*concepts.UUID)(id).UnmarshalText(data)
}
func (id *AuthorID) UnmarshalJSON(data []byte) error {
	return (*concepts.UUID)(id).UnmarshalJSON(data)
}

func Example_domainRoundTrip() {
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-AABBCCDDEEFF")
	if err != nil {
		panic(err)
	}
	type author struct {
		ID   AuthorID `json:"id"`
		Name string   `json:"name"`
	}
	original := author{ID: AuthorID(id), Name: "Ada"}
	data, err := json.Marshal(original)
	if err != nil {
		panic(err)
	}
	var restored author
	if err := json.Unmarshal(data, &restored); err != nil {
		panic(err)
	}
	fmt.Println(string(data))
	fmt.Println("same author:", restored == original)
	// Output:
	// {"id":"00112233-4455-6677-8899-aabbccddeeff","name":"Ada"}
	// same author: true
}

func Example_correlationPropagation() {
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-aabbccddeeff")
	if err != nil {
		panic(err)
	}
	operation, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := correlation.WithID(operation, id)
	// Pass ctx to the next operation; the parent remains unchanged.
	fmt.Println("save author:", correlation.FromContext(ctx))
	fmt.Println("parent unset:", correlation.FromContext(operation).IsZero())
	cleared := correlation.WithID(ctx, correlation.ID{})
	fmt.Println("child unset:", correlation.FromContext(cleared).IsZero())
	cancel()
	fmt.Println("canceled:", cleared.Err() == context.Canceled)
	// Output:
	// save author: 00112233-4455-6677-8899-aabbccddeeff
	// parent unset: true
	// child unset: true
	// canceled: true
}
