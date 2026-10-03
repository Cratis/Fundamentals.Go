// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

//go:build providerprobe

package samberdo

import (
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/ditest"
	"github.com/cratis/fundamentals.go/recipes/internal/borrowed"
)

// This intentionally failing probe runs every level-1 subtest without skips.
// Run explicitly when reassessing qualification; it is not the CI pass target.
func TestProviderProbe(t *testing.T) {
	ditest.RunProvider(t, func(cfg ditest.Config) (di.Provider, error) {
		if cfg.BindScopeFactory || len(cfg.ContextGuards) != 0 {
			return nil, borrowed.ErrUnsupported
		}
		return New(cfg.Bindings...)
	})
}
