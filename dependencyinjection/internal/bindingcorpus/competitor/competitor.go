// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package competitor adds constructorless, ignored structural competition.
package competitor

// Other must still count as an implementation even when ignored.
//
//cratis:ignore-convention
type Other struct{}

// Work structurally implements unique.IFoo without importing it.
func (Other) Work() { panic("analysis executed method") }
