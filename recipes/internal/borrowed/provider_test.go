// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package borrowed

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

type engine map[di.Key]func() (any, error)

func (e engine) Register(key di.Key, factory func() (any, error)) error { e[key] = factory; return nil }
func (e engine) Resolve(key di.Key) (any, error)                        { return e[key]() }

func binding(t *testing.T, deps ...di.Key) di.Binding {
	t.Helper()
	b, err := di.NewBinding(di.Singleton, di.Borrowed, func(context.Context, di.Resolver) (string, error) {
		t.Fatal("validation ran factory")
		return "", nil
	}, deps...)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInvalidGraphsFailBeforeRegistration(t *testing.T) {
	b := binding(t)
	for _, tc := range []struct {
		name     string
		bindings []di.Binding
		want     error
	}{
		{"zero", []di.Binding{{}}, di.ErrInvalidRegistration},
		{"duplicate", []di.Binding{b, b}, di.ErrDuplicate},
		{"missing", []di.Binding{binding(t, di.KeyFor[int]())}, di.ErrMissing},
		{"cycle", []di.Binding{binding(t, di.KeyFor[string]())}, di.ErrCycle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := engine{}
			if _, err := New(e, tc.bindings); !errors.Is(err, tc.want) || len(e) != 0 {
				t.Fatalf("New = %v; registrations = %d", err, len(e))
			}
		})
	}
}

func TestUndeclaredFactoryResolutionFails(t *testing.T) {
	b, err := di.NewBinding(di.Singleton, di.Borrowed, func(ctx context.Context, r di.Resolver) (string, error) {
		_, err := di.Resolve[int](ctx, r)
		return "", err
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := New(engine{}, []di.Binding{b})
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.NewScope(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := di.Resolve[string](t.Context(), s); !errors.Is(err, di.ErrUndeclaredDependency) {
		t.Fatal(err)
	}
	if err := p.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestWaitingCancellationAndCloseJoin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		key := di.KeyFor[string]()
		p := Wrap(engine{key: func() (any, error) { <-release; return "value", nil }}, map[di.Key]bool{key: true})
		s, err := p.NewScope(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		creator := make(chan error, 1)
		go func() { _, err := s.Resolve(t.Context(), key); creator <- err }()
		synctest.Wait()
		waitCtx, cancel := context.WithCancel(t.Context())
		waiter := make(chan error, 1)
		go func() { _, err := s.Resolve(waitCtx, key); waiter <- err }()
		synctest.Wait()
		cancel()
		if err := <-waiter; !errors.Is(err, context.Canceled) {
			t.Error("waiter:", err)
		}
		closeCtx, cancelClose := context.WithCancel(t.Context())
		closing := make(chan error, 1)
		go func() { closing <- p.Close(closeCtx) }()
		synctest.Wait()
		select {
		case err := <-closing:
			t.Error("Close did not join admitted call:", err)
			closing <- err
		default:
		}
		cancelClose()
		if err := <-closing; !errors.Is(err, context.Canceled) {
			t.Error("close:", err)
		}
		if _, err := s.Resolve(t.Context(), key); !errors.Is(err, di.ErrClosed) {
			t.Error("new resolution admitted while closing:", err)
		}
		close(release)
		if err := <-creator; err != nil {
			t.Error("admitted call:", err)
		}
		if err := p.Close(t.Context()); err != nil {
			t.Error("resumed close:", err)
		}
	})
}
