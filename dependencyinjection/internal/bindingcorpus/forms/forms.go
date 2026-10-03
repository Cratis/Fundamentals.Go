// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package forms supplies constructor shapes. Constructors panic deliberately:
// a binding planner must inspect declarations, never execute them.
package forms

import "context"

// Dependency is an external non-scalar obligation.
type Dependency struct{}

// Value is a value-result service with private representation.
type Value struct{ private int }

// NewValue is a zero-dependency constructor.
func NewValue() Value {
	_ = Value{}.private
	panic("analysis executed constructor")
}

// Pointer returns a pointer service.
func Pointer() *Value { panic("analysis executed constructor") }

// WithError returns the predeclared error.
func WithError() (*Value, error) { panic("analysis executed constructor") }

// ErrorAlias preserves error identity.
type ErrorAlias = error

// AliasError returns an alias of error.
func AliasError() (*Value, ErrorAlias) { panic("analysis executed constructor") }

// ContextAlias preserves context identity.
type ContextAlias = context.Context

// Repeated tests context and repeated argument keys.
func Repeated(ContextAlias, *Dependency, *Dependency) (*Value, error) {
	panic("analysis executed constructor")
}

// Alias is an alternate spelling, not a new key.
type Alias = Value

// AliasPointer returns an alias spelling.
func AliasPointer() *Alias { panic("analysis executed constructor") }

// Contract is a named ordinary interface.
type Contract interface{ Work() }

// InterfaceResult directly registers its interface result.
func InterfaceResult() Contract { panic("analysis executed constructor") }

// ErrorLike is not the predeclared error.
type ErrorLike interface{ Error() string }

// WrongError has an incompatible second result.
func WrongError() (Value, ErrorLike) { panic("analysis executed constructor") }

// Variadic is unsupported.
func Variadic(...*Dependency) Value { panic("analysis executed constructor") }

// Extra has too many results.
func Extra() (Value, int, error) { panic("analysis executed constructor") }

// Empty has no results.
func Empty() { panic("analysis executed constructor") }

// Misplaced has a non-leading context.
func Misplaced(*Dependency, context.Context) Value { panic("analysis executed constructor") }

// DoubleContext has two contexts.
func DoubleContext(context.Context, context.Context) Value { panic("analysis executed constructor") }

// Scalar requires explicit configuration.
func Scalar(string) Value { panic("analysis executed constructor") }

// Port is named scalar configuration.
type Port uint16

// NamedScalar also requires configuration.
func NamedScalar(Port) Value { panic("analysis executed constructor") }

type private struct{}

func newPrivate() *private { panic("analysis executed constructor") }

// PrivateResult cannot be rendered into another package.
func PrivateResult() *private { return newPrivate() }

// PrivateArgument names an inaccessible dependency.
func PrivateArgument(*private) Value { panic("analysis executed constructor") }

// Box is generic; only closed instantiations are services.
type Box[T any] struct{ Value T }

// NewBox is open and requires a nongeneric wrapper.
func NewBox[T any](T) *Box[T] { panic("analysis executed constructor") }

// Closed returns a closed generic result.
func Closed() *Box[int] { panic("analysis executed constructor") }

// ClosedDependency uses a closed generic dependency.
func ClosedDependency(*Box[int]) Value { panic("analysis executed constructor") }

// Disposable is a disposable value result.
type Disposable struct{}

// Close implements io.Closer on the value method set.
func (Disposable) Close() error { panic("analysis executed method") }

// LargeDisposable requires a safe adapter and is rejected in v1.
func LargeDisposable(*Dependency, *Dependency, *Dependency, *Dependency, *Dependency) Disposable {
	panic("analysis executed constructor")
}

// LargePointer is safe to adapt with a nil pointer on dependency failure.
func LargePointer(*Dependency, *Dependency, *Dependency, *Dependency, *Dependency) *Disposable {
	panic("analysis executed constructor")
}

// LargeValue is a non-disposable value result.
func LargeValue(*Dependency, *Dependency, *Dependency, *Dependency, *Dependency) Value {
	panic("analysis executed constructor")
}

// NewValueWithDependency is never discovered by a richest-constructor heuristic.
func NewValueWithDependency(*Dependency) Value { panic("analysis executed constructor") }

// Build is a method, not a constructor.
func (Value) Build() Value { panic("analysis executed method") }

type privateAlias = Value

// PrivateAliasResult can be spelled through its accessible canonical target.
func PrivateAliasResult() *privateAlias { panic("analysis executed constructor") }

// PublicAlias exposes an otherwise private target by a usable spelling.
type PublicAlias = private

// PublicAliasResult preserves that usable alias across packages.
func PublicAliasResult() *PublicAlias { panic("analysis executed constructor") }

// ContextLike is structurally identical but is not the exact context type.
type ContextLike context.Context

// ContextDependency declares ContextLike as an ordinary dependency.
func ContextDependency(ContextLike) Value { panic("analysis executed constructor") }

// ContextDisposable prefers context-aware disposal.
type ContextDisposable struct{}

// Close has the context-aware cleanup contract.
func (ContextDisposable) Close(context.Context) error { panic("analysis executed method") }

// LargeContextDisposable is also unsupported with a value result.
func LargeContextDisposable(*Dependency, *Dependency, *Dependency, *Dependency, *Dependency) ContextDisposable {
	panic("analysis executed constructor")
}

// SmallDisposable is safe through BindFunc1.
func SmallDisposable(*Dependency) Disposable { panic("analysis executed constructor") }

type aliasChain = PublicAlias

// AliasChainResult retains the accessible intermediate alias.
func AliasChainResult() *aliasChain { panic("analysis executed constructor") }

// AliasGenericResult normalizes a closed generic argument spelling.
func AliasGenericResult() *Box[aliasChain] { panic("analysis executed constructor") }

type hidden interface{ Work() }

// EmbeddedPrivateArgument cannot spell its anonymous interface externally.
func EmbeddedPrivateArgument(interface{ hidden }) Value { panic("analysis executed constructor") }

// PublicInterface hides an inaccessible embedded declaration.
type PublicInterface interface{ hidden }

// PublicInterfaceArgument can name the exported interface itself.
func PublicInterfaceArgument(PublicInterface) Value { panic("analysis executed constructor") }
