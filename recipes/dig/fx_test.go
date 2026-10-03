// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dig_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
	"github.com/cratis/fundamentals.go/dependencyinjection/ditest"
	"github.com/cratis/fundamentals.go/recipes/dig"
	"go.uber.org/fx"
)

type lifecycle struct{ hooks []fx.Hook }

func (l *lifecycle) Append(hook fx.Hook) { l.hooks = append(l.hooks, hook) }

func TestHostedProvider(t *testing.T) {
	// Qualification applies to the real Fundamentals provider returned by Provide,
	// not to native dig/fx resolution. Actual fx lifecycle is tested separately.
	ditest.RunProvider(t, func(cfg ditest.Config) (di.Provider, error) {
		var registry container.Registry
		for _, b := range cfg.Bindings {
			if err := registry.Register(b); err != nil {
				return nil, err
			}
		}
		if cfg.BindScopeFactory {
			if err := registry.BindScopeFactory(); err != nil {
				return nil, err
			}
		}
		return dig.Provide(&lifecycle{}, &registry)
	})
}

type connection struct{ closed *int }

func (c *connection) Close() error { (*c.closed)++; return nil }

func fxApp(closed *int, provider *di.Provider, startError error) *fx.App {
	var registry container.Registry
	err := di.Bind(&registry, di.Singleton, func(context.Context, di.Resolver) (*connection, error) {
		return &connection{closed: closed}, nil
	})
	if err != nil {
		panic(err)
	}
	return fx.New(fx.NopLogger, fx.Supply(&registry), fx.Provide(dig.Provide),
		fx.Invoke(func(lc fx.Lifecycle, p di.Provider) {
			*provider = p
			lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
				s, err := p.NewScope(ctx)
				if err != nil {
					return err
				}
				_, err = di.Resolve[*connection](ctx, s)
				return errors.Join(err, s.Close(ctx), startError)
			}})
		}))
}

func ExampleProvide() {
	closed := 0
	var provider di.Provider
	app := fxApp(&closed, &provider, nil)
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}
	fmt.Println("running, closed:", closed)
	if err := app.Stop(ctx); err != nil {
		panic(err)
	}
	fmt.Println("stopped, closed:", closed)
	// Output:
	// running, closed: 0
	// stopped, closed: 1
}

func TestFXStopsProviderAndRollsBackFailedStart(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint("failed_start=", fail), func(t *testing.T) {
			closed := 0
			var provider di.Provider
			var cause error
			if fail {
				cause = errors.New("start failed")
			}
			app := fxApp(&closed, &provider, cause)
			if err := app.Start(t.Context()); !errors.Is(err, cause) {
				t.Fatalf("start = %v", err)
			}
			if fail && closed != 1 {
				t.Fatal("failed startup did not roll back provider")
			}
			if err := app.Stop(t.Context()); err != nil {
				t.Fatal(err)
			}
			if closed != 1 {
				t.Fatalf("closed %d times", closed)
			}
			if _, err := provider.NewScope(t.Context()); !errors.Is(err, di.ErrClosed) {
				t.Fatalf("provider remained open: %v", err)
			}
		})
	}
}
