// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package uuidinterop_test

import (
	"encoding/json"
	"fmt"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/recipes/uuidinterop"
)

func Example() {
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-aabbccddeeff")
	if err != nil {
		panic(err)
	}
	for _, value := range []any{id, uuidinterop.Google(id), uuidinterop.Gofrs(id)} {
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s %s\n", value, data)
	}
	// Output:
	// 00112233-4455-6677-8899-aabbccddeeff "00112233-4455-6677-8899-aabbccddeeff"
	// 00112233-4455-6677-8899-aabbccddeeff "00112233-4455-6677-8899-aabbccddeeff"
	// 00112233-4455-6677-8899-aabbccddeeff "00112233-4455-6677-8899-aabbccddeeff"
}
