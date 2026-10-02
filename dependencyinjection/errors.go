// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection

import (
	"errors"
	"slices"
)

var (
	// ErrWrongType identifies a result incompatible with the exact requested key.
	ErrWrongType = errors.New("wrong service type")
	// ErrFactoryFailed identifies a factory error; its cause remains inspectable.
	ErrFactoryFailed = errors.New("service factory failed")
	// ErrInvalidRegistration identifies invalid binding input.
	ErrInvalidRegistration = errors.New("invalid service registration")
	// ErrDuplicate identifies duplicate bindings or declared dependencies.
	ErrDuplicate = errors.New("duplicate service registration")
	// ErrMissing identifies a missing exact type binding.
	ErrMissing = errors.New("service missing")
	// ErrCycle identifies a dependency cycle.
	ErrCycle = errors.New("service dependency cycle")
	// ErrCaptiveLifetime identifies a singleton path reaching a scoped service.
	ErrCaptiveLifetime = errors.New("captive service lifetime")
	// ErrUndeclaredDependency identifies an undeclared factory edge.
	ErrUndeclaredDependency = errors.New("undeclared service dependency")
	// ErrClosed identifies closing/closed providers or scopes.
	ErrClosed = errors.New("services closed")
	// ErrInvalidScope identifies zero, nil, or otherwise invalid scope handles.
	ErrInvalidScope = errors.New("invalid service scope")
	// ErrContextMismatch identifies guard rejection, including captured presence changes.
	ErrContextMismatch = errors.New("service context mismatch")
	// ErrResolverExpired identifies use after a factory returns.
	ErrResolverExpired = errors.New("factory scope expired")
	// ErrConcurrentFactoryUse identifies overlapping resolution on one factory view.
	ErrConcurrentFactoryUse = errors.New("concurrent factory scope use")
	// ErrNilValue identifies nil or typed-nil produced/registered values.
	ErrNilValue = errors.New("nil service value")
	// ErrCallbackPanicked identifies factory, cleanup or context-guard panics.
	ErrCallbackPanicked = errors.New("service callback panicked")
)

// Error is an inspectable service failure. Text is diagnostic-only, not a stable
// format, and never includes service values, causes or panic payloads. Constructors
// copy Path; adapters constructing Error themselves must also copy retained paths.
type Error struct {
	// Operation is the failing operation.
	Operation string
	// Key is the exact service key, if applicable.
	Key Key
	// Path is a copied dependency diagnostic path.
	Path []Key
	// Kind is the stable failure category.
	Kind error
	// Cause preserves callback or cancellation failures without displaying them.
	Cause error
	// Panic captures a callback panic without displaying it.
	Panic any
}

// Error returns only operation, type identity, and category.
func (e *Error) Error() string {
	text := "dependencyinjection: " + e.Operation + " " + e.Key.String()
	if e.Kind != nil {
		text += ": " + e.Kind.Error()
	}
	return text
}

// Unwrap preserves both category and cause for errors.Is/errors.As.
func (e *Error) Unwrap() []error {
	var result []error
	if e.Kind != nil {
		result = append(result, e.Kind)
	}
	if e.Cause != nil {
		result = append(result, e.Cause)
	}
	return result
}

func failure(operation string, key Key, path []Key, kind, cause error) *Error {
	return &Error{Operation: operation, Key: key, Path: slices.Clone(path), Kind: kind, Cause: cause}
}
