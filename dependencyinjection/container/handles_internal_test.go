// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	"errors"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// Keep Arc's invalid concrete-zero-handle assertions even though the extracted
// provider and scope types are now private and inaccessible to external tests.
func TestZeroImplementationHandles(t *testing.T) {
	ctx := context.Background()
	for _, p := range []*provider{nil, {}} {
		if err := p.Close(ctx); !errors.Is(err, di.ErrInvalidScope) {
			t.Fatal(err)
		}
		if _, err := p.NewScope(ctx); !errors.Is(err, di.ErrInvalidScope) {
			t.Fatal(err)
		}
		if p.Owns(&scope{}) || p.Contains(di.KeyFor[int]()) {
			t.Fatal("invalid provider introspection")
		}
	}
	for _, s := range []*scope{nil, {}} {
		if _, err := s.Resolve(ctx, di.KeyFor[int]()); !errors.Is(err, di.ErrInvalidScope) {
			t.Fatal(err)
		}
		if err := s.CheckContext(ctx); !errors.Is(err, di.ErrInvalidScope) {
			t.Fatal(err)
		}
		if err := s.Close(ctx); !errors.Is(err, di.ErrInvalidScope) {
			t.Fatal(err)
		}
	}
}
