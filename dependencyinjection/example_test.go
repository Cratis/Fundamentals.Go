// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection_test

import (
	"context"
	"fmt"
	"sync"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

type report struct{ title string }

func newReport(title string) *report { return &report{title: title} }

func Example() {
	// Ordinary constructors are the first path. You own their resources explicitly.
	value := newReport("Quarterly report")
	fmt.Println(value.title)
	// Output: Quarterly report
}

// localFactory supplies application values without importing a container.
// Catalog is intentionally absent: consumers try resolution instead.
type localFactory struct{ title string }

func (f localFactory) NewScope(ctx context.Context) (di.Scope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &localScope{value: newReport(f.title)}, nil
}

type localScope struct {
	mu     sync.Mutex
	value  *report
	closed bool
}

func (s *localScope) Resolve(ctx context.Context, key di.Key) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, &di.Error{Operation: "resolve", Key: key, Kind: di.ErrClosed}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if key != di.KeyFor[*report]() {
		return nil, &di.Error{Operation: "resolve", Key: key, Kind: di.ErrMissing}
	}
	return s.value, nil
}
func (s *localScope) Close(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func ExampleScopeFactory() {
	var factory di.ScopeFactory = localFactory{title: "Quarterly report"}
	ctx := context.Background()
	scope, err := factory.NewScope(ctx)
	if err != nil {
		panic(err)
	}
	value, err := di.Resolve[*report](ctx, scope)
	if err != nil {
		panic(err)
	}
	fmt.Println(value.title)
	if err := scope.Close(ctx); err != nil {
		panic(err)
	}
	// Output: Quarterly report
}

func ExampleCatalog() {
	var factory di.ScopeFactory = localFactory{title: "Quarterly report"}
	key := di.KeyFor[*report]()
	resolvable := true // Without Catalog, try resolution rather than rejecting startup.
	if catalog, ok := factory.(di.Catalog); ok {
		resolvable = catalog.Contains(key)
	}
	fmt.Println(resolvable)
	// Output: true
}
