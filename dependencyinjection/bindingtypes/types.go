// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes

import (
	"cmp"
	"go/types"
	"slices"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

func identity(t types.Type) string {
	if t == nil {
		return ""
	}
	return types.TypeString(types.Unalias(t), func(p *types.Package) string { return p.Path() })
}

func objectNil(o types.Object) bool {
	if o == nil {
		return true
	}
	switch o := o.(type) {
	case *types.Func:
		return o == nil
	case *types.TypeName:
		return o == nil
	}
	return false
}

func sortPolicies(policies []TypePolicy) {
	slices.SortFunc(policies, func(a, b TypePolicy) int {
		return cmp.Compare(objectIdentity(a.Type), objectIdentity(b.Type))
	})
}

func objectIdentity(o types.Object) string {
	if objectNil(o) {
		return ""
	}
	prefix := ""
	if o.Pkg() != nil {
		prefix = o.Pkg().Path() + "."
	}
	return prefix + o.Name() + ":" + identity(o.Type())
}

func finish(p Plan) Plan {
	slices.SortFunc(p.Bindings, func(a, b Binding) int { return cmp.Compare(identity(a.Service), identity(b.Service)) })
	for i := range p.Diagnostics {
		slices.SortFunc(p.Diagnostics[i].Related, func(a, b types.Object) int {
			return cmp.Compare(objectIdentity(a), objectIdentity(b))
		})
	}
	slices.SortFunc(p.Diagnostics, func(a, b Diagnostic) int {
		return cmp.Or(cmp.Compare(identity(a.Type), identity(b.Type)),
			cmp.Compare(objectIdentity(a.Subject), objectIdentity(b.Subject)),
			cmp.Compare(a.Code, b.Code), cmp.Compare(a.Severity, b.Severity),
			cmp.Compare(a.Message, b.Message), cmp.Compare(a.Pos, b.Pos))
	})
	for _, d := range p.Diagnostics {
		if d.Severity == Error {
			p.Bindings = nil
			break
		}
	}
	return p
}

func diagnostic(code Code, subject types.Object, t types.Type, message string) Diagnostic {
	d := Diagnostic{Code: code, Severity: Error, Subject: subject, Type: t, Message: message}
	if !objectNil(subject) {
		d.Pos = subject.Pos()
	} else {
		d.Subject = nil
	}
	return d
}

func validLifetime(lt di.Lifetime) bool { return lt >= di.Singleton && lt <= di.Transient }

func boolOrder(b bool) int {
	if b {
		return 1
	}
	return 0
}

func named(t types.Type) *types.Named {
	if t == nil {
		return nil
	}
	t = types.Unalias(t)
	if p, ok := t.(*types.Pointer); ok {
		t = types.Unalias(p.Elem())
	}
	n, _ := t.(*types.Named)
	return n
}

func ordinaryInterface(t types.Type) *types.Interface {
	if t == nil {
		return nil
	}
	i, _ := t.Underlying().(*types.Interface)
	if i != nil && i.Complete().IsMethodSet() {
		return i
	}
	return nil
}

func bindingIndex(bindings []Binding, t types.Type) int {
	for i, b := range bindings {
		if types.Identical(b.Service, t) {
			return i
		}
	}
	return -1
}

func registrationIndex(regs []Registration, t types.Type) int {
	for i, r := range regs {
		if r.Service != nil && types.Identical(r.Service, t) {
			return i
		}
	}
	return -1
}

func appendUnique(ts []types.Type, t types.Type) []types.Type {
	for _, prior := range ts {
		if types.Identical(prior, t) {
			return ts
		}
	}
	return append(ts, t)
}

func accessibleObject(o types.Object, emit *types.Package) bool {
	return o != nil && (o.Pkg() == nil || o.Pkg() == emit || o.Exported())
}

// spelling keeps usable aliases and replaces private alias spellings when their
// canonical target can be named. Keys are still compared by types.Identical.
func spelling(t types.Type, emit *types.Package) types.Type {
	switch t := t.(type) {
	case *types.Alias:
		if !accessibleObject(t.Obj(), emit) {
			return spelling(t.Rhs(), emit)
		}
		return instantiatedSpelling(t, t.TypeArgs(), emit)
	case *types.Named:
		return instantiatedSpelling(t, t.TypeArgs(), emit)
	case *types.Pointer:
		return types.NewPointer(spelling(t.Elem(), emit))
	case *types.Slice:
		return types.NewSlice(spelling(t.Elem(), emit))
	case *types.Array:
		return types.NewArray(spelling(t.Elem(), emit), t.Len())
	case *types.Map:
		return types.NewMap(spelling(t.Key(), emit), spelling(t.Elem(), emit))
	case *types.Chan:
		return types.NewChan(t.Dir(), spelling(t.Elem(), emit))
	case *types.Signature:
		if t.TypeParams().Len() == 0 && t.RecvTypeParams().Len() == 0 {
			return types.NewSignatureType(t.Recv(), nil, nil, spellingTuple(t.Params(), emit), spellingTuple(t.Results(), emit), t.Variadic())
		}
	case *types.Struct:
		fields := make([]*types.Var, t.NumFields())
		tags := make([]string, t.NumFields())
		for i := range fields {
			f := t.Field(i)
			fields[i] = types.NewField(f.Pos(), f.Pkg(), f.Name(), spelling(f.Type(), emit), f.Embedded())
			tags[i] = t.Tag(i)
		}
		return types.NewStruct(fields, tags)
	case *types.Interface:
		methods := make([]*types.Func, t.NumExplicitMethods())
		for i := range methods {
			m := t.ExplicitMethod(i)
			methods[i] = types.NewFunc(m.Pos(), m.Pkg(), m.Name(), spelling(m.Type(), emit).(*types.Signature))
		}
		embedded := make([]types.Type, t.NumEmbeddeds())
		for i := range embedded {
			embedded[i] = spelling(t.EmbeddedType(i), emit)
		}
		return types.NewInterfaceType(methods, embedded).Complete()
	}
	return t
}

func spellingTuple(tuple *types.Tuple, emit *types.Package) *types.Tuple {
	vars := make([]*types.Var, tuple.Len())
	for i := range vars {
		v := tuple.At(i)
		vars[i] = types.NewVar(v.Pos(), v.Pkg(), v.Name(), spelling(v.Type(), emit))
	}
	return types.NewTuple(vars...)
}

func instantiatedSpelling(t types.Type, args *types.TypeList, emit *types.Package) types.Type {
	if args.Len() == 0 {
		return t
	}
	arguments := make([]types.Type, args.Len())
	for i := range arguments {
		arguments[i] = spelling(args.At(i), emit)
	}
	var origin types.Type
	switch t := t.(type) {
	case *types.Named:
		origin = t.Origin()
	case *types.Alias:
		origin = t.Origin()
	}
	// Rebuild the spelling without altering borrowed types or canonical keys.
	instance, err := types.Instantiate(nil, origin, arguments, false)
	if err != nil {
		return t // Accessibility validation will reject an unusable spelling.
	}
	return instance
}

// Named declarations need only their spelling and type arguments, not private fields.
// Anonymous composites recursively require every name the renderer must spell.
func accessible(t types.Type, emit *types.Package) bool {
	switch t := t.(type) {
	case *types.Alias:
		if accessibleObject(t.Obj(), emit) && t.TypeParams().Len() == t.TypeArgs().Len() {
			for i := 0; i < t.TypeArgs().Len(); i++ {
				if !accessible(t.TypeArgs().At(i), emit) {
					return false
				}
			}
			return true
		}
		return false
	case *types.Named:
		if !accessibleObject(t.Obj(), emit) {
			return false
		}
		for i := 0; i < t.TypeArgs().Len(); i++ {
			if !accessible(t.TypeArgs().At(i), emit) {
				return false
			}
		}
		return true
	case *types.Basic:
		return t.Kind() != types.Invalid && t.Info()&types.IsUntyped == 0
	case *types.Pointer:
		return accessible(t.Elem(), emit)
	case *types.Slice:
		return accessible(t.Elem(), emit)
	case *types.Array:
		return accessible(t.Elem(), emit)
	case *types.Map:
		return accessible(t.Key(), emit) && accessible(t.Elem(), emit)
	case *types.Chan:
		return accessible(t.Elem(), emit)
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if !accessibleObject(t.Field(i), emit) || !accessible(t.Field(i).Type(), emit) {
				return false
			}
		}
		return true
	case *types.Interface:
		if !t.Complete().IsMethodSet() {
			return false
		}
		for i := 0; i < t.NumEmbeddeds(); i++ {
			if !accessible(t.EmbeddedType(i), emit) {
				return false
			}
		}
		for i := 0; i < t.NumMethods(); i++ {
			if !accessibleObject(t.Method(i), emit) || !accessible(t.Method(i).Type(), emit) {
				return false
			}
		}
		return true
	case *types.Signature:
		return t.TypeParams().Len() == 0 && accessibleTuple(t.Params(), emit) && accessibleTuple(t.Results(), emit)
	}
	return false
}

func accessibleTuple(t *types.Tuple, emit *types.Package) bool {
	for i := 0; i < t.Len(); i++ {
		if !accessible(t.At(i).Type(), emit) {
			return false
		}
	}
	return true
}

func closed(t types.Type) bool {
	return closedType(t, make(map[types.Type]bool))
}

func closedType(t types.Type, seen map[types.Type]bool) bool {
	if t == nil {
		return false
	}
	if seen[t] {
		return true
	}
	seen[t] = true
	switch t := types.Unalias(t).(type) {
	case *types.TypeParam, *types.Union:
		return false
	case *types.Named:
		if t.TypeParams().Len() != t.TypeArgs().Len() {
			return false
		}
		for i := 0; i < t.TypeArgs().Len(); i++ {
			if !closedType(t.TypeArgs().At(i), seen) {
				return false
			}
		}
		if i, ok := t.Underlying().(*types.Interface); ok && !i.Complete().IsMethodSet() {
			return false
		}
		return true
	case *types.Pointer:
		return closedType(t.Elem(), seen)
	case *types.Slice:
		return closedType(t.Elem(), seen)
	case *types.Array:
		return closedType(t.Elem(), seen)
	case *types.Map:
		return closedType(t.Key(), seen) && closedType(t.Elem(), seen)
	case *types.Chan:
		return closedType(t.Elem(), seen)
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if !closedType(t.Field(i).Type(), seen) {
				return false
			}
		}
	case *types.Interface:
		if !t.Complete().IsMethodSet() {
			return false
		}
		for i := 0; i < t.NumMethods(); i++ {
			if !closedType(t.Method(i).Type(), seen) {
				return false
			}
		}
	case *types.Signature:
		if t.TypeParams().Len() != 0 {
			return false
		}
		for _, tuple := range []*types.Tuple{t.Params(), t.Results()} {
			for i := 0; i < tuple.Len(); i++ {
				if !closedType(tuple.At(i).Type(), seen) {
					return false
				}
			}
		}
	case *types.Basic:
		return t.Kind() != types.Invalid && t.Info()&types.IsUntyped == 0
	default:
		return false
	}
	return true
}

func isContext(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == "context" && n.Obj().Name() == "Context"
}

func disposableValue(t types.Type) bool {
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface, *types.Map, *types.Chan, *types.Slice, *types.Signature:
		return false
	}
	method, _, _ := types.LookupFieldOrMethod(t, false, nil, "Close")
	fn, ok := method.(*types.Func)
	if !ok {
		return false
	}
	sig := fn.Type().(*types.Signature)
	return !sig.Variadic() && sig.Results().Len() == 1 && types.Identical(sig.Results().At(0).Type(), types.Universe.Lookup("error").Type()) &&
		(sig.Params().Len() == 0 || (sig.Params().Len() == 1 && isContext(sig.Params().At(0).Type())))
}
