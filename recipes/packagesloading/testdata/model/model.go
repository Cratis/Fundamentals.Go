// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package model is local type-checker input, not a separate module or executable.
package model

import (
	"encoding/json"
	"errors"

	"github.com/cratis/fundamentals.go/concepts"
)

// ID aliases an exact shared scalar.
type ID = concepts.UUID

// Name is a string concept with scalar codecs.
type Name string

// ConceptValue returns the scalar representation.
func (n Name) ConceptValue() string { return string(n) }

// MarshalText encodes the scalar.
func (n Name) MarshalText() ([]byte, error) { return []byte(n), nil }

// UnmarshalText decodes the scalar.
func (n *Name) UnmarshalText(data []byte) error { *n = Name(data); return nil }

// MarshalJSON encodes a JSON string.
func (n Name) MarshalJSON() ([]byte, error) { return json.Marshal(string(n)) }

// UnmarshalJSON decodes a JSON string, rejecting null.
func (n *Name) UnmarshalJSON(data []byte) error {
	var text *string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	if text == nil {
		return errors.New("name must be a string")
	}
	*n = Name(*text)
	return nil
}

// Plain is intentionally not a concept.
type Plain string

// Version is ignored because it is not a type.
const Version = 1
