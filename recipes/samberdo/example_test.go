// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package samberdo_test

import (
	"context"
	"fmt"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/recipes/samberdo"
)

func ExampleNew() {
	binding, err := di.NewBinding(di.Singleton, di.Borrowed, func(context.Context, di.Resolver) (string, error) {
		return "cached by samber/do", nil
	})
	if err != nil {
		panic(err)
	}
	p, err := samberdo.New(binding)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	s, err := p.NewScope(ctx)
	if err != nil {
		panic(err)
	}
	message, err := di.Resolve[string](ctx, s)
	if err != nil {
		panic(err)
	}
	fmt.Println(message)
	if err := p.Close(ctx); err != nil {
		panic(err)
	}
	// Output: cached by samber/do
}
