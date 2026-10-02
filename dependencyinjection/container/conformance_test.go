// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
	"github.com/cratis/fundamentals.go/dependencyinjection/ditest"
)

func buildConformanceProvider(cfg ditest.Config) (di.Provider, error) {
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
	var options []container.Option
	for _, guard := range cfg.ContextGuards {
		options = append(options, container.WithContextGuard(guard))
	}
	return registry.Build(options...)
}

func TestProviderConformance(t *testing.T) {
	ditest.RunProvider(t, buildConformanceProvider)
}

func TestDeclaredGraphConformance(t *testing.T) {
	ditest.RunDeclaredGraph(t, buildConformanceProvider)
}
