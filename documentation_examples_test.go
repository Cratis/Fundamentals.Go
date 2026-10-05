// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"encoding/json"
	"fmt"

	"github.com/cratis/fundamentals.go/concepts"
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

// Keep this getting-started file to one Example and no Test functions: go/doc's
// whole-file mode preserves the blank Concept forwarding assertion above.
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
