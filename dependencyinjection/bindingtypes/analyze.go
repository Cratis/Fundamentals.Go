// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes

import (
	"cmp"
	"go/types"
	"slices"
	"strings"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

type analyzer struct {
	cfg  Config
	pkgs []*types.Package
	plan Plan
}

// Analyze plans bindings without loading packages, executing code or accessing files.
// Inputs must be complete, coherently type-checked packages. A nil constructor list
// discovers exact NewX declarations; a non-nil empty list discovers nothing.
// Errors invalidate the entire plan. Missing non-scalar dependencies are normally
// informational; scalar configuration always needs an explicit registration.
func Analyze(pkgs []*types.Package, cfg Config) Plan {
	a := analyzer{cfg: cfg, pkgs: slices.Clone(pkgs)}
	if !a.normalize() {
		return finish(a.plan)
	}
	a.constructors()
	a.interfaces()
	a.existing()
	a.dependencies()
	return finish(a.plan)
}

func (a *analyzer) add(code Code, subject types.Object, t types.Type, message string) {
	a.plan.Diagnostics = append(a.plan.Diagnostics, diagnostic(code, subject, t, message))
}

func (a *analyzer) normalize() bool {
	a.cfg.Policies = slices.Clone(a.cfg.Policies)
	slices.SortFunc(a.cfg.Policies, func(x, y TypePolicy) int {
		return cmp.Or(cmp.Compare(objectIdentity(x.Type), objectIdentity(y.Type)), cmp.Compare(x.Lifetime, y.Lifetime), cmp.Compare(boolOrder(x.Ignore), boolOrder(y.Ignore)), cmp.Compare(x.Pos, y.Pos))
	})
	a.cfg.Existing = slices.Clone(a.cfg.Existing)
	slices.SortFunc(a.cfg.Existing, func(x, y Registration) int {
		return cmp.Or(cmp.Compare(identity(x.Service), identity(y.Service)), cmp.Compare(x.Lifetime, y.Lifetime))
	})
	if len(a.pkgs) == 0 {
		a.add(InvalidInput, nil, nil, "supply at least one complete package")
		return false
	}
	// Imports participate in identity validation, but not constructor discovery.
	universe := make(map[string]*types.Package)
	var visit func(*types.Package) bool
	visit = func(p *types.Package) bool {
		if p == nil || p.Path() == "" {
			return false
		}
		if prior := universe[p.Path()]; prior != nil {
			return prior == p
		}
		universe[p.Path()] = p
		for _, imported := range p.Imports() {
			if !visit(imported) {
				return false
			}
		}
		return true
	}
	seen := make(map[*types.Package]bool)
	for _, p := range a.pkgs {
		if p == nil || !p.Complete() || seen[p] || !visit(p) {
			a.add(InvalidInput, nil, nil, "packages must be distinct and share one importer universe")
			return false
		}
		seen[p] = true
	}
	if a.cfg.EmitPackage == nil {
		if len(a.pkgs) != 1 {
			a.add(MissingConfiguration, nil, nil, "EmitPackage is required for multiple input packages")
			return false
		}
		a.cfg.EmitPackage = a.pkgs[0]
	}
	if !a.cfg.EmitPackage.Complete() || !visit(a.cfg.EmitPackage) {
		a.add(InvalidInput, nil, nil, "invalid EmitPackage identity")
		return false
	}
	if a.cfg.Duplicates > KeepExisting {
		a.add(InvalidInput, nil, nil, "unknown duplicate policy")
	}
	// Validate package identities in configured types without loading anything.
	checkType := func(t types.Type) bool {
		return t != nil && closed(t) && coherentType(t, universe, make(map[types.Type]bool))
	}
	for i, p := range a.cfg.Policies {
		if p.Type == nil || named(p.Type.Type()) == nil || !coherentType(p.Type.Type(), universe, make(map[types.Type]bool)) || (p.Lifetime != 0 && !validLifetime(p.Lifetime)) {
			a.add(InvalidInput, p.Type, nil, "invalid type policy")
			continue
		}
		for _, previous := range a.cfg.Policies[:i] {
			if previous.Type != nil && sameFamily(previous.Type.Type(), p.Type.Type()) {
				d := diagnostic(ConflictingPolicy, p.Type, p.Type.Type(), "duplicate type policy")
				d.Pos = p.Pos
				d.Related = []types.Object{previous.Type}
				a.plan.Diagnostics = append(a.plan.Diagnostics, d)
				break
			}
		}
	}
	for i, r := range a.cfg.Existing {
		if !checkType(r.Service) || !validLifetime(r.Lifetime) {
			a.add(InvalidInput, nil, r.Service, "existing registrations require a closed key and explicit lifetime")
			continue
		}
		if registrationIndex(a.cfg.Existing[:i], r.Service) >= 0 {
			a.add(DuplicateBinding, nil, r.Service, "duplicate existing registration")
		}
	}
	for _, pair := range a.cfg.Interfaces {
		if !checkType(pair.Service) || !checkType(pair.Implementation) {
			a.add(InvalidInput, nil, pair.Service, "interface selections require closed coherent types")
		}
	}
	slices.SortFunc(a.pkgs, func(x, y *types.Package) int { return cmp.Compare(x.Path(), y.Path()) })
	for _, d := range a.plan.Diagnostics {
		if d.Severity == Error {
			return false
		}
	}
	return true
}

func coherentType(t types.Type, universe map[string]*types.Package, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return t != nil
	}
	seen[t] = true
	var obj types.Object
	switch n := t.(type) {
	case *types.Named:
		obj = n.Obj()
	case *types.Alias:
		obj = n.Obj()
	}
	if obj != nil && obj.Pkg() != nil {
		pkg := obj.Pkg()
		if prior := universe[pkg.Path()]; prior != nil && prior != pkg {
			return false
		}
		universe[pkg.Path()] = pkg
	}
	switch t := t.(type) {
	case *types.Alias:
		return coherentType(types.Unalias(t), universe, seen)
	case *types.Named:
		for i := 0; i < t.TypeArgs().Len(); i++ {
			if !coherentType(t.TypeArgs().At(i), universe, seen) {
				return false
			}
		}
		return coherentType(t.Underlying(), universe, seen)
	case *types.Pointer:
		return coherentType(t.Elem(), universe, seen)
	case *types.Slice:
		return coherentType(t.Elem(), universe, seen)
	case *types.Array:
		return coherentType(t.Elem(), universe, seen)
	case *types.Map:
		return coherentType(t.Key(), universe, seen) && coherentType(t.Elem(), universe, seen)
	case *types.Chan:
		return coherentType(t.Elem(), universe, seen)
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if !coherentType(t.Field(i).Type(), universe, seen) {
				return false
			}
		}
	case *types.Interface:
		for i := 0; i < t.NumMethods(); i++ {
			if !coherentType(t.Method(i).Type(), universe, seen) {
				return false
			}
		}
	case *types.Signature:
		for _, tuple := range []*types.Tuple{t.Params(), t.Results()} {
			for i := 0; i < tuple.Len(); i++ {
				if !coherentType(tuple.At(i).Type(), universe, seen) {
					return false
				}
			}
		}
	}
	return true
}

func sameFamily(x, y types.Type) bool {
	nx, ny := named(x), named(y)
	return nx != nil && ny != nil && types.Identical(nx.Origin(), ny.Origin())
}

func (a *analyzer) policy(t types.Type) TypePolicy {
	for _, p := range a.cfg.Policies {
		if p.Type != nil && sameFamily(p.Type.Type(), t) {
			if p.Lifetime == 0 {
				p.Lifetime = di.Transient
			}
			return p
		}
	}
	return TypePolicy{Lifetime: di.Transient}
}

func (a *analyzer) constructors() {
	selected := slices.Clone(a.cfg.Constructors)
	if a.cfg.Constructors == nil {
		for _, pkg := range a.pkgs {
			for _, name := range pkg.Scope().Names() {
				if !strings.HasPrefix(name, "New") {
					continue
				}
				fn, ok := pkg.Scope().Lookup(name).(*types.Func)
				typeName, typeOK := pkg.Scope().Lookup(strings.TrimPrefix(name, "New")).(*types.TypeName)
				if !ok || !typeOK || named(typeName.Type()) == nil {
					continue
				}
				if a.policy(typeName.Type()).Ignore {
					continue
				}
				sig := fn.Type().(*types.Signature)
				if sig.Results().Len() > 0 && !sameFamily(sig.Results().At(0).Type(), typeName.Type()) {
					a.add(UnsupportedSignature, fn, nil, "NewX must return the associated local named X or its pointer")
					continue
				}
				selected = append(selected, fn)
			}
		}
	}
	slices.SortFunc(selected, func(x, y *types.Func) int { return cmp.Compare(objectIdentity(x), objectIdentity(y)) })
	for _, fn := range selected {
		if fn != nil && fn.Type().(*types.Signature).Recv() != nil {
			a.add(UnsupportedSignature, fn, nil, "methods are not package-level constructors")
			continue
		}
		if fn == nil || fn.Pkg() == nil || fn.Parent() != fn.Pkg().Scope() || !slices.Contains(a.pkgs, fn.Pkg()) {
			a.add(InvalidInput, fn, nil, "constructors must be package-level functions in the supplied universe")
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Recv() != nil || sig.Variadic() || sig.TypeParams().Len() != 0 || sig.Results().Len() < 1 || sig.Results().Len() > 2 || !closed(sig) {
			a.add(UnsupportedSignature, fn, nil, "expected a nongeneric nonvariadic constructor returning T or (T, error)")
			continue
		}
		service := spelling(sig.Results().At(0).Type(), a.cfg.EmitPackage)
		policy := a.policy(service)
		if policy.Ignore {
			continue
		}
		n := named(service)
		if n == nil || (ordinaryInterface(service) == nil && isInterface(service)) ||
			(sig.Results().Len() == 2 && !types.Identical(sig.Results().At(1).Type(), types.Universe.Lookup("error").Type())) {
			a.add(UnsupportedSignature, fn, service, "service must be named and second result must be the predeclared error")
			continue
		}
		// Only T and *T are supported, not pointers to ordinary interfaces.
		if _, pointer := types.Unalias(service).(*types.Pointer); pointer && isInterface(n) {
			a.add(UnsupportedSignature, fn, service, "pointer-to-interface services are unsupported")
			continue
		}
		b := Binding{Service: service, Lifetime: policy.Lifetime, Ownership: di.Owned, Action: Register, Constructor: fn, ReturnsError: sig.Results().Len() == 2}
		valid := true
		for i := 0; i < sig.Params().Len(); i++ {
			param := spelling(sig.Params().At(i).Type(), a.cfg.EmitPackage)
			if isContext(param) {
				if i != 0 {
					a.add(UnsupportedSignature, fn, service, "context.Context is allowed only once, in leading position")
					valid = false
				}
				b.PassContext = true
				continue
			}
			b.Arguments = append(b.Arguments, param)
			b.Dependencies = appendUnique(b.Dependencies, param)
		}
		accessibleArguments := true
		for _, argument := range b.Arguments {
			accessibleArguments = accessibleArguments && accessible(argument, a.cfg.EmitPackage)
		}
		if !accessibleObject(fn, a.cfg.EmitPackage) || !accessible(service, a.cfg.EmitPackage) || !accessibleArguments {
			a.add(InaccessibleDeclaration, fn, service, "constructor or type cannot be named from EmitPackage")
			valid = false
		}
		if len(b.Arguments) > 4 && disposableValue(service) {
			a.add(UnsupportedSignature, fn, service, "disposable value results with more than four dependencies require a safe runtime adapter")
			valid = false
		}
		if !valid {
			continue
		}
		if i := bindingIndex(a.plan.Bindings, service); i >= 0 {
			d := diagnostic(AmbiguousConstructor, fn, service, "multiple selected constructors return the same exact key")
			d.Related = []types.Object{a.plan.Bindings[i].Constructor, fn}
			a.plan.Diagnostics = append(a.plan.Diagnostics, d)
			continue
		}
		a.plan.Bindings = append(a.plan.Bindings, b)
	}
}

func isInterface(t types.Type) bool {
	_, ok := t.Underlying().(*types.Interface)
	return ok
}

func (a *analyzer) interfaces() {
	pairs := slices.Clone(a.cfg.Interfaces)
	if a.cfg.MatchIFoo {
		for _, pkg := range a.pkgs {
			for _, name := range pkg.Scope().Names() {
				obj, ok := pkg.Scope().Lookup(name).(*types.TypeName)
				if !ok || !strings.HasPrefix(name, "I") || len(name) < 2 || ordinaryInterface(obj.Type()) == nil || !closed(obj.Type()) || a.policy(obj.Type()).Ignore {
					continue
				}
				if slices.ContainsFunc(pairs, func(p InterfaceBinding) bool { return types.Identical(p.Service, obj.Type()) }) {
					continue // Explicit selections resolve convention ambiguity.
				}
				concrete, ok := pkg.Scope().Lookup(name[1:]).(*types.TypeName)
				if !ok || named(concrete.Type()) == nil || isInterface(concrete.Type()) || a.policy(concrete.Type()).Ignore {
					continue
				}
				contract := ordinaryInterface(obj.Type())
				// The declaration may be generic. Match exact closed constructor and
				// existing keys before considering the open family's method set.
				var keys []types.Type
				for _, b := range a.plan.Bindings {
					if sameFamily(b.Service, concrete.Type()) && types.Implements(b.Service, contract) {
						keys = appendUnique(keys, b.Service)
					}
				}
				for _, r := range a.cfg.Existing {
					if sameFamily(r.Service, concrete.Type()) && types.Implements(r.Service, contract) {
						keys = appendUnique(keys, r.Service)
					}
				}
				if len(keys) == 0 && (!closed(concrete.Type()) ||
					(!types.Implements(concrete.Type(), contract) && !types.Implements(types.NewPointer(concrete.Type()), contract))) {
					continue
				}
				competitors := a.implementations(contract)
				if len(competitors) > 1 {
					d := diagnostic(AmbiguousImplementation, obj, obj.Type(), "multiple named implementation families in supplied packages")
					d.Related = competitors
					a.plan.Diagnostics = append(a.plan.Diagnostics, d)
					continue
				}
				if len(keys) == 0 {
					a.add(ConstructorNotFound, concrete, concrete.Type(), "convention implementation has no selected constructor or existing exact key")
					continue
				}
				if len(keys) > 1 {
					a.add(AmbiguousImplementation, obj, obj.Type(), "both value and pointer keys are available; select an exact key explicitly")
					continue
				}
				pairs = append(pairs, InterfaceBinding{Service: obj.Type(), Implementation: keys[0]})
			}
		}
	}
	slices.SortFunc(pairs, func(x, y InterfaceBinding) int {
		return cmp.Or(cmp.Compare(identity(x.Service), identity(y.Service)), cmp.Compare(identity(x.Implementation), identity(y.Implementation)))
	})
	for _, pair := range pairs {
		pair.Service = spelling(pair.Service, a.cfg.EmitPackage)
		pair.Implementation = spelling(pair.Implementation, a.cfg.EmitPackage)
		i := ordinaryInterface(pair.Service)
		if named(pair.Service) == nil || i == nil || isInterface(pair.Implementation) || !types.Implements(pair.Implementation, i) {
			a.add(InvalidInterfaceBinding, nil, pair.Service, "selection must map a named ordinary interface to a compatible concrete exact key")
			continue
		}
		if a.policy(pair.Service).Ignore || a.policy(pair.Implementation).Ignore {
			continue
		}
		if !accessible(pair.Service, a.cfg.EmitPackage) || !accessible(pair.Implementation, a.cfg.EmitPackage) {
			a.add(InaccessibleDeclaration, nil, pair.Service, "interface selection cannot be named from EmitPackage")
			continue
		}
		if bindingIndex(a.plan.Bindings, pair.Service) >= 0 {
			a.add(DuplicateBinding, nil, pair.Service, "multiple generated registrations for an interface key")
			continue
		}
		lifetime := di.Lifetime(0)
		if index := bindingIndex(a.plan.Bindings, pair.Implementation); index >= 0 {
			lifetime = a.plan.Bindings[index].Lifetime
		}
		if index := registrationIndex(a.cfg.Existing, pair.Implementation); index >= 0 && (lifetime == 0 || a.cfg.Duplicates == KeepExisting) {
			lifetime = a.cfg.Existing[index].Lifetime
		}
		if lifetime == 0 {
			a.add(ConstructorNotFound, nil, pair.Implementation, "implementation has no selected constructor or existing registration")
			continue
		}
		a.plan.Bindings = append(a.plan.Bindings, Binding{Service: pair.Service, Lifetime: lifetime, Ownership: di.Borrowed, Action: Register, Forward: pair.Implementation, Dependencies: []types.Type{pair.Implementation}})
	}
}

func (a *analyzer) implementations(contract *types.Interface) []types.Object {
	var families []types.Type
	var result []types.Object
	add := func(t types.Type) {
		n := named(t)
		if n == nil || isInterface(t) || slices.ContainsFunc(families, func(prior types.Type) bool { return sameFamily(prior, t) }) {
			return
		}
		if types.Implements(t, contract) || types.Implements(types.NewPointer(t), contract) {
			families = append(families, t)
			result = append(result, n.Origin().Obj())
		}
	}
	// Declaration types include constructorless/ignored instantiations in
	// parameters, variables, fields and methods, not just function results.
	// Walk each exact shape once; recursive declarations must terminate.
	seen := make(map[types.Type]bool)
	var visit func(types.Type)
	visit = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		if n := named(t); n != nil && slices.Contains(a.pkgs, n.Obj().Pkg()) {
			add(t)
		}
		switch t := t.(type) {
		case *types.Alias:
			for i := 0; i < t.TypeArgs().Len(); i++ {
				visit(t.TypeArgs().At(i))
			}
			visit(t.Rhs())
		case *types.Named:
			for i := 0; i < t.TypeArgs().Len(); i++ {
				visit(t.TypeArgs().At(i))
			}
			visit(t.Underlying())
			for i := 0; i < t.NumMethods(); i++ {
				visit(t.Method(i).Type())
			}
		case *types.Pointer:
			visit(t.Elem())
		case *types.Slice:
			visit(t.Elem())
		case *types.Array:
			visit(t.Elem())
		case *types.Map:
			visit(t.Key())
			visit(t.Elem())
		case *types.Chan:
			visit(t.Elem())
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				visit(t.Field(i).Type())
			}
		case *types.Interface:
			for i := 0; i < t.NumEmbeddeds(); i++ {
				visit(t.EmbeddedType(i))
			}
			for i := 0; i < t.NumExplicitMethods(); i++ {
				visit(t.ExplicitMethod(i).Type())
			}
		case *types.Signature:
			for _, tuple := range []*types.Tuple{t.Params(), t.Results()} {
				for i := 0; i < tuple.Len(); i++ {
					visit(tuple.At(i).Type())
				}
			}
		}
	}
	for _, pkg := range a.pkgs {
		for _, name := range pkg.Scope().Names() {
			visit(pkg.Scope().Lookup(name).Type())
		}
	}
	for _, r := range a.cfg.Existing {
		if n := named(r.Service); n != nil && slices.Contains(a.pkgs, n.Obj().Pkg()) {
			add(r.Service)
		}
	}
	return result
}

func (a *analyzer) existing() {
	for index, b := range a.plan.Bindings {
		i := registrationIndex(a.cfg.Existing, b.Service)
		if i < 0 {
			continue
		}
		if a.cfg.Duplicates != KeepExisting {
			a.add(DuplicateBinding, b.Constructor, b.Service, "generated key overlaps an existing registration")
			continue
		}
		a.plan.Bindings[index] = Binding{Service: b.Service, Lifetime: a.cfg.Existing[i].Lifetime, Ownership: b.Ownership, Action: RetainExisting}
		d := diagnostic(ExistingRegistrationRetained, b.Constructor, b.Service, "explicitly listed existing registration retained")
		d.Severity = Information
		a.plan.Diagnostics = append(a.plan.Diagnostics, d)
	}
}

func (a *analyzer) dependencies() {
	for _, b := range a.plan.Bindings {
		for _, dep := range b.Dependencies {
			if bindingIndex(a.plan.Bindings, dep) >= 0 || registrationIndex(a.cfg.Existing, dep) >= 0 {
				continue
			}
			d := diagnostic(MissingDependency, b.Constructor, dep, "declare an external provider for this exact dependency")
			d.Severity = Information
			if a.cfg.RequireAllDependencies {
				d.Severity = Error
			}
			if _, scalar := dep.Underlying().(*types.Basic); scalar {
				d.Code = MissingConfiguration
				d.Severity = Error
				d.Message = "scalar configuration requires an explicit provider or existing registration"
			}
			a.plan.Diagnostics = append(a.plan.Diagnostics, d)
		}
	}
}
