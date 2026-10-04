// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package repository supplies a constructor-planning fixture, not an active store.
package repository

// Store is the shared dependency in the planning example.
//
//cratis:singleton
type Store struct{}

// NewStore must never execute during package loading or planning.
func NewStore() *Store {
	panic("constructor planning must not activate Store")
}
