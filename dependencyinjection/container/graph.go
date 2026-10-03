// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"slices"
)

func orderedKeys(bindings map[di.Key]binding) []di.Key {
	keys := make([]di.Key, 0, len(bindings))
	for key := range bindings {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b di.Key) int {
		if a.String() < b.String() {
			return -1
		}
		if a.String() > b.String() {
			return 1
		}
		return 0
	})
	return keys
}
func orderedDependencies(b binding) []di.Key {
	deps := slices.Clone(b.dependencies)
	slices.SortFunc(deps, func(a, b di.Key) int {
		if a.String() < b.String() {
			return -1
		}
		if a.String() > b.String() {
			return 1
		}
		return 0
	})
	return deps
}
func validateGraph(bindings map[di.Key]binding) error {
	keys := orderedKeys(bindings)
	for _, key := range keys {
		for _, dep := range orderedDependencies(bindings[key]) {
			if _, exists := bindings[dep]; !exists {
				return failure("build", dep, []di.Key{key, dep}, di.ErrMissing, nil)
			}
		}
	}
	visited := map[di.Key]bool{}
	var visit func(di.Key, []di.Key) error
	visit = func(key di.Key, path []di.Key) error {
		if slices.Contains(path, key) {
			return failure("build", key, append(slices.Clone(path), key), di.ErrCycle, nil)
		}
		if visited[key] {
			return nil
		}
		path = append(slices.Clone(path), key)
		for _, dep := range orderedDependencies(bindings[key]) {
			if err := visit(dep, path); err != nil {
				return err
			}
		}
		visited[key] = true
		return nil
	}
	for _, key := range keys {
		if err := visit(key, nil); err != nil {
			return err
		}
	}
	var captive func(di.Key, []di.Key, map[di.Key]bool) error
	captive = func(key di.Key, path []di.Key, seen map[di.Key]bool) error {
		path = append(slices.Clone(path), key)
		if bindings[key].lifetime == di.Scoped {
			return failure("build", key, path, di.ErrCaptiveLifetime, nil)
		}
		if seen[key] {
			return nil
		}
		seen[key] = true
		for _, dep := range orderedDependencies(bindings[key]) {
			if err := captive(dep, path, seen); err != nil {
				return err
			}
		}
		return nil
	}
	for _, key := range keys {
		if bindings[key].lifetime == di.Singleton {
			if err := captive(key, nil, map[di.Key]bool{}); err != nil {
				return err
			}
		}
	}
	return nil
}
