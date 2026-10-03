// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dig

import (
	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
	"go.uber.org/fx"
)

// Provide builds a Fundamentals provider and registers its Close with fx.
// Use fx.Provide(Provide) and supply a configured *container.Registry. Construct
// service resources in OnStart, not during fx graph construction: fx cannot stop
// hooks that have not started. Hooks using these services must be appended after
// this provider's hook, so their OnStop runs before provider disposal.
func Provide(lifecycle fx.Lifecycle, registry *container.Registry) (di.Provider, error) {
	p, err := registry.Build()
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStop: p.Close})
	return p, nil
}
