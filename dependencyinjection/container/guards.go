// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

type settings struct{ guards []di.CaptureContext }

// Option configures the default container. Options are applied in order at Build.
type Option func(*settings)

// WithContextGuard appends a guard captured at NewScope, never Build.
// Checks run on every scoped resolution, including restricted factory views.
// Singleton dependency paths hide all values and bypass request guard checks.
// Guards capture immutable metadata/presence, not contexts; they run outside locks
// and must support concurrent calls. Nil callbacks fail Build.
func WithContextGuard(capture di.CaptureContext) Option {
	return func(s *settings) { s.guards = append(s.guards, capture) }
}
func captureChecks(ctx context.Context, guards []di.CaptureContext) (checks []di.ContextCheck, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			checks = nil
			err = &di.Error{Operation: "capture context", Kind: di.ErrCallbackPanicked, Panic: recovered}
		}
	}()
	for _, capture := range guards {
		check, cause := capture(ctx)
		if cause != nil || check == nil {
			return nil, failure("capture context", di.Key{}, nil, di.ErrContextMismatch, cause)
		}
		checks = append(checks, check)
	}
	return checks, nil
}
func (s *scopeState) check(ctx context.Context) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = &di.Error{Operation: "check context", Kind: di.ErrCallbackPanicked, Panic: recovered}
		}
	}()
	for _, check := range s.checks {
		if cause := check(ctx); cause != nil {
			return failure("check context", di.Key{}, nil, di.ErrContextMismatch, cause)
		}
	}
	return nil
}
