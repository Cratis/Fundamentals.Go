// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package borrowed

import di "github.com/cratis/fundamentals.go/dependencyinjection"

func validate(bindings []di.Binding) error {
	byKey := make(map[di.Key]di.Binding, len(bindings))
	for _, b := range bindings {
		if err := b.Validate(); err != nil {
			return err
		}
		if b.Ownership() != di.Borrowed || b.Lifetime() != di.Singleton {
			return ErrUnsupported
		}
		if _, exists := byKey[b.Key()]; exists {
			return di.ErrDuplicate
		}
		byKey[b.Key()] = b
	}
	visited, active := map[di.Key]bool{}, map[di.Key]bool{}
	var visit func(di.Key) error
	visit = func(key di.Key) error {
		b, exists := byKey[key]
		if !exists {
			return di.ErrMissing
		}
		if active[key] {
			return di.ErrCycle
		}
		if visited[key] {
			return nil
		}
		active[key] = true
		for _, dep := range b.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}
		active[key], visited[key] = false, true
		return nil
	}
	for _, b := range bindings {
		if err := visit(b.Key()); err != nil {
			return err
		}
	}
	return nil
}
