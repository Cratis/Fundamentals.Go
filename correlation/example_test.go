// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package correlation_test

import (
	"context"
	"fmt"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/correlation"
)

func ExampleWithID() {
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-aabbccddeeff")
	if err != nil {
		fmt.Println(err)
		return
	}
	parent := correlation.WithID(context.Background(), id)
	child := correlation.WithID(parent, correlation.ID{})
	fmt.Println(correlation.FromContext(parent))
	fmt.Println(correlation.FromContext(child).IsZero())
	// Output:
	// 00112233-4455-6677-8899-aabbccddeeff
	// true
}

func ExampleFromContext() {
	ctx := context.Background()
	fmt.Println(correlation.FromContext(ctx).IsZero())

	// Generation is an explicit caller decision, never a side effect of reading.
	id, err := concepts.NewUUID()
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx = correlation.WithID(ctx, id)
	fmt.Println(correlation.FromContext(ctx) == id)
	// Output:
	// true
	// true
}
