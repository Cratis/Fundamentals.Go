// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package service supplies a constructor-planning fixture, not an active service.
package service

import "github.com/cratis/fundamentals.go/recipes/packagesloading/testdata/bindings/repository"

// Service consumes the same exact store key twice.
//
//cratis:scoped
type Service struct{}

// NewService must never execute during package loading or planning.
func NewService(first, second *repository.Store) (*Service, error) {
	panic("constructor planning must not activate Service")
}
