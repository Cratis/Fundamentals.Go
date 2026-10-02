// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

func TestSingletonTransitiveContextRetainsCancellationAndExactDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(withPrincipal(context.Background(), "request"), time.Minute)
		defer cancel()
		deadline, _ := ctx.Deadline()
		var r container.Registry
		entered := make(chan struct{})
		register(t, &r, di.Transient, func(rootCtx context.Context, _ di.Resolver) (*beta, error) {
			got, ok := rootCtx.Deadline()
			if !ok || got != deadline || rootCtx.Done() != ctx.Done() {
				t.Error("context lifetime lost")
			}
			if rootCtx.Value(principalKey{}) != nil {
				t.Error("value leaked")
			}
			close(entered)
			<-rootCtx.Done()
			return nil, rootCtx.Err()
		})
		register(t, &r, di.Singleton, func(rootCtx context.Context, resolver di.Resolver) (*alpha, error) {
			_, err := di.Resolve[*beta](rootCtx, resolver)
			return nil, err
		}, di.KeyFor[*beta]())
		p := provider(t, &r)
		s := scope(t, p, ctx)
		done := make(chan error, 1)
		go func() { _, err := di.Resolve[*alpha](ctx, s); done <- err }()
		<-entered
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}

func TestStartupScopeClosePreservesSingletonUntilProviderClose(t *testing.T) {
	var registry container.Registry
	closed := 0
	register(t, &registry, di.Singleton, func(context.Context, di.Resolver) (*resource, error) {
		return &resource{close: func(context.Context) error { closed++; return nil }}, nil
	})
	p := provider(t, &registry)
	ctx := context.Background()
	startup := scope(t, p, ctx)
	artifact := resolved[*resource](t, ctx, startup)
	if err := startup.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if closed != 0 {
		t.Fatal("startup scope disposed singleton")
	}
	later := scope(t, p, ctx)
	if got := resolved[*resource](t, ctx, later); got != artifact {
		t.Fatal("singleton identity lost")
	}
	if err := later.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if closed != 0 {
		t.Fatal("later scope disposed singleton")
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Fatal("provider did not own singleton", closed)
	}
}

func TestUseAfterClosingStarts(t *testing.T) {
	for _, providerClose := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			var r container.Registry
			entered, release := make(chan struct{}), make(chan struct{})
			register(t, &r, di.Scoped, func(context.Context, di.Resolver) (*item, error) { close(entered); <-release; return &item{}, nil })
			p := provider(t, &r)
			ctx := context.Background()
			s := scope(t, p, ctx)
			resolving := make(chan error, 1)
			go func() { _, err := di.Resolve[*item](ctx, s); resolving <- err }()
			<-entered
			closing := make(chan error, 1)
			closeCall := s.Close
			if providerClose {
				closeCall = p.Close
			}
			go func() { closing <- closeCall(ctx) }()
			synctest.Wait()
			if _, err := di.Resolve[*item](ctx, s); !errors.Is(err, di.ErrClosed) {
				t.Fatal(err)
			}
			if err := s.CheckContext(ctx); !errors.Is(err, di.ErrClosed) {
				t.Fatal(err)
			}
			close(release)
			if err := <-resolving; err != nil {
				t.Fatal(err)
			}
			if err := <-closing; err != nil {
				t.Fatal(err)
			}
		})
	}
}
