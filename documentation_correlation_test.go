// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"context"
	"fmt"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/correlation"
)

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
