// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes

import (
	"go/token"
	"go/types"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// Config declares the analysis universe and product-owned composition decisions.
// All packages and types must belong to one coherent type-checking universe.
// Slices are read, not retained; referenced go/types objects remain borrowed.
type Config struct {
	// EmitPackage is the rendering destination; nil defaults to the sole input package.
	EmitPackage *types.Package
	// Constructors is an explicit whitelist; nil discovers exact package-level NewX functions.
	Constructors []*types.Func
	// Policies supplies source directives or explicit external-type policies.
	Policies []TypePolicy
	// MatchIFoo enables same-package IFoo/Foo convention matching.
	MatchIFoo bool
	// Interfaces explicitly selects exact interface/implementation keys.
	Interfaces []InterfaceBinding
	// Existing declares already registered keys and their actual lifetimes.
	Existing []Registration
	// Duplicates defaults to RejectDuplicates.
	Duplicates DuplicatePolicy
	// RequireAllDependencies promotes missing external dependencies to errors.
	RequireAllDependencies bool
}

// TypePolicy applies to a named type and constructors returning it or its pointer.
type TypePolicy struct {
	// Type identifies the declaration; aliases share the target's policy.
	Type *types.TypeName
	// Lifetime defaults to transient when zero.
	Lifetime di.Lifetime
	// Ignore excludes this type from DI conventions, not product artifact discovery.
	Ignore bool
	// Pos locates the policy in source, if available.
	Pos token.Pos
}

// InterfaceBinding selects a forwarding binding without naming heuristics.
type InterfaceBinding struct {
	// Service is a named ordinary interface key.
	Service types.Type
	// Implementation is the exact concrete key to resolve, including pointer shape.
	Implementation types.Type
}

// Registration attests an existing exact key and its effective lifetime.
type Registration struct {
	// Service is an already registered exact type.
	Service types.Type
	// Lifetime is required; zero is not an attestation of a lifetime.
	Lifetime di.Lifetime
}

// DuplicatePolicy determines how explicitly listed existing keys are handled.
type DuplicatePolicy uint8

const (
	// RejectDuplicates rejects generated/existing overlaps (the default).
	RejectDuplicates DuplicatePolicy = iota
	// KeepExisting retains only keys explicitly listed in Config.Existing.
	KeepExisting
)

// Plan contains independently owned slices. Any error makes Bindings nil.
// Referenced go/types objects are borrowed and must be treated as immutable.
type Plan struct {
	// Bindings is deterministically sorted by package-qualified service identity.
	Bindings []Binding
	// Diagnostics is deterministically ordered; messages are not stable identifiers.
	Diagnostics []Diagnostic
}

// Action tells a renderer whether to register or explicitly skip a key.
type Action uint8

const (
	// Register emits a strict registration whose errors must propagate.
	Register Action = iota + 1
	// RetainExisting emits no registration; the product verifies the existing manifest.
	RetainExisting
)

// Binding describes constructor invocation or borrowed interface forwarding.
// Register has exactly one of Constructor and Forward. RetainExisting has neither.
type Binding struct {
	// Service is the exact registration key; aliases retain usable source spellings.
	Service types.Type
	// Lifetime is the effective concrete lifetime, also used by forwarders.
	Lifetime di.Lifetime
	// Ownership is Owned for constructors and Borrowed for forwarders.
	Ownership di.Ownership
	// Action controls whether registration is emitted.
	Action Action
	// Constructor is invoked directly, never by analysis.
	Constructor *types.Func
	// Arguments preserves parameter order and repetition, excluding leading context.
	Arguments []types.Type
	// PassContext passes the construction context as the first parameter.
	PassContext bool
	// ReturnsError indicates a second result of the predeclared error type.
	ReturnsError bool
	// Forward is the exact concrete key to resolve for an interface binding.
	Forward types.Type
	// Dependencies contains unique direct exact keys in first-argument order.
	Dependencies []types.Type
}

// Severity distinguishes fatal plans from informational obligations.
type Severity string

const (
	// Error prevents emission of any bindings.
	Error Severity = "error"
	// Information reports a decision or external obligation.
	Information Severity = "information"
)

// Code is a stable diagnostic identifier; Message is explanatory only.
type Code string

const (
	// InvalidInput identifies invalid packages, configuration or metadata.
	InvalidInput Code = "BT001"
	// InvalidDirective identifies unknown, malformed or misplaced directives.
	InvalidDirective Code = "BT002"
	// ConflictingPolicy identifies duplicate or conflicting policies.
	ConflictingPolicy Code = "BT003"
	// ConstructorNotFound identifies a requested key without a selected constructor.
	ConstructorNotFound Code = "BT004"
	// AmbiguousConstructor identifies multiple constructors for one exact key.
	AmbiguousConstructor Code = "BT005"
	// UnsupportedSignature identifies unsupported constructor forms or unsafe adapters.
	UnsupportedSignature Code = "BT006"
	// InaccessibleDeclaration identifies declarations the destination cannot name.
	InaccessibleDeclaration Code = "BT007"
	// InvalidInterfaceBinding identifies invalid or incompatible interface selections.
	InvalidInterfaceBinding Code = "BT008"
	// AmbiguousImplementation identifies competing structural implementations.
	AmbiguousImplementation Code = "BT009"
	// DuplicateBinding identifies conflicting registrations.
	DuplicateBinding Code = "BT010"
	// MissingDependency identifies an external dependency obligation.
	MissingDependency Code = "BT011"
	// MissingConfiguration identifies a scalar dependency needing explicit configuration.
	MissingConfiguration Code = "BT012"
	// ExistingRegistrationRetained records an explicit keep-existing decision.
	ExistingRegistrationRetained Code = "BT013"
)

// Diagnostic carries stable classification and borrowed source/type metadata.
type Diagnostic struct {
	// Code is stable across message changes.
	Code Code
	// Severity determines whether the plan can be rendered.
	Severity Severity
	// Pos locates the diagnostic if source positions are available.
	Pos token.Pos
	// Subject is the relevant declaration, if any.
	Subject types.Object
	// Type is the relevant exact key, if any.
	Type types.Type
	// Related lists competing declarations in deterministic order.
	Related []types.Object
	// Message explains the problem or decision without a stable text contract.
	Message string
}
