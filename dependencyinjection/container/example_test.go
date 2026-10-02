// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	"fmt"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

type title string
type report struct{ title title }

func newReport(_ context.Context, value title) (*report, error) { return &report{title: value}, nil }

func ExampleRegistry() {
	var registry container.Registry
	if err := di.BindValue(&registry, title("Quarterly report")); err != nil {
		panic(err)
	}
	// The parameter type derives the title edge; Build checks it before use.
	if err := di.BindFunc1(&registry, di.Scoped, newReport); err != nil {
		panic(err)
	}
	p, err := registry.Build()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	s, err := p.NewScope(ctx)
	if err != nil {
		panic(err)
	}
	value, err := di.Resolve[*report](ctx, s)
	if err != nil {
		panic(err)
	}
	fmt.Println(value.title)
	if err := s.Close(ctx); err != nil {
		panic(err)
	}
	if err := p.Close(ctx); err != nil {
		panic(err)
	}
	// Output: Quarterly report
}

func ExampleRegistry_clientLifetime() {
	var registry container.Registry
	if err := di.BindValue(&registry, title("Client report")); err != nil {
		panic(err)
	}
	if err := di.BindFunc1(&registry, di.Singleton, newReport); err != nil {
		panic(err)
	}
	p, err := registry.Build()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	// Resolve the Singleton at client startup through a short-lived scope.
	startupScope, err := p.NewScope(ctx)
	if err != nil {
		panic(err)
	}
	value, err := di.Resolve[*report](ctx, startupScope)
	if err != nil {
		panic(err)
	}
	if err := startupScope.Close(ctx); err != nil {
		panic(err)
	}
	// The provider still owns value after the startup scope closes.
	fmt.Println(value.title)
	// The application coordinates client/provider shutdown ordering.
	if err := p.Close(ctx); err != nil {
		panic(err)
	}
	// Output: Client report
}

type reporter interface{ Title() string }
type ownedReporter struct{ closed *int }

func (*ownedReporter) Title() string                 { return "Shared report" }
func (r *ownedReporter) Close(context.Context) error { (*r.closed)++; return nil }

func ExampleRegistry_interfaceForwarding() {
	var registry container.Registry
	closed := 0
	if err := di.Bind(&registry, di.Scoped, func(context.Context, di.Resolver) (*ownedReporter, error) {
		return &ownedReporter{closed: &closed}, nil
	}); err != nil {
		panic(err)
	}
	// Preserve the concrete lifetime and borrow the forwarded interface result.
	if err := di.BindBorrowed(&registry, di.Scoped, func(ctx context.Context, resolver di.Resolver) (reporter, error) {
		return di.Resolve[*ownedReporter](ctx, resolver)
	}, di.KeyFor[*ownedReporter]()); err != nil {
		panic(err)
	}
	p, err := registry.Build()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	s, err := p.NewScope(ctx)
	if err != nil {
		panic(err)
	}
	concrete, err := di.Resolve[*ownedReporter](ctx, s)
	if err != nil {
		panic(err)
	}
	forwarded, err := di.Resolve[reporter](ctx, s)
	if err != nil {
		panic(err)
	}
	fmt.Println(forwarded == concrete, forwarded.Title())
	if err := p.Close(ctx); err != nil {
		panic(err)
	}
	fmt.Println("closed", closed)
	// Output:
	// true Shared report
	// closed 1
}

func run(ctx context.Context) error {
	var registry container.Registry
	if err := di.BindValue(&registry, title("Quarterly report")); err != nil {
		return err
	}
	if err := di.BindFunc1(&registry, di.Scoped, newReport); err != nil {
		return err
	}
	p, err := registry.Build()
	if err != nil {
		return err
	}
	s, err := p.NewScope(ctx)
	if err != nil {
		return errors.Join(err, p.Close(context.Background()))
	}
	value, resolveErr := di.Resolve[*report](ctx, s)
	if resolveErr == nil {
		fmt.Println(value.title)
	}
	return errors.Join(resolveErr, s.Close(context.Background()), p.Close(context.Background()))
}

func ExampleRegistry_cleanupErrors() {
	if err := run(context.Background()); err != nil {
		panic(err)
	}
	// Output: Quarterly report
}
