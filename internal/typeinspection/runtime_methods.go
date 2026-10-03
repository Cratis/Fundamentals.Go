// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package typeinspection supplies non-mutating views of checked Go types.
package typeinspection

import "go/types"

// RuntimeMethodSet excludes generic methods BEFORE resolving promotion, matching
// the runtime's method lists. A generic method neither shadows nor collides with
// a promoted ordinary method; fields and ordinary methods still do. Filtering a
// completed method set would lose the ordinary methods hidden by go/types.
//
// A temporary view copies only receiver structure and embedded type graphs.
// Method signatures retain their original parameter/result types, preserving
// nominal type identity. The caller's type information is never modified.
// Calls are safe concurrently when callers do not mutate the supplied types.
func RuntimeMethodSet(t types.Type) *types.MethodSet {
	clones := make(map[*types.Named]*types.Named)
	instances := make(map[*types.Named][]*types.Named)
	var visible func(types.Type) types.Type
	visible = func(t types.Type) types.Type {
		t = types.Unalias(t)
		switch t := t.(type) {
		case *types.Named:
			if clone := clones[t]; clone != nil {
				return clone
			}
			// Independently created instantiations may be identical but not equal.
			// Keep one view per identity so repeated embedding stays ambiguous and
			// recursive embeddings terminate just as in types.NewMethodSet.
			origin := t.Origin()
			for _, instance := range instances[origin] {
				if types.Identical(t, instance) {
					return clones[instance]
				}
			}
			obj := t.Obj()
			clone := types.NewNamed(types.NewTypeName(obj.Pos(), obj.Pkg(), obj.Name(), nil), types.Typ[types.Invalid], nil)
			clones[t] = clone
			instances[origin] = append(instances[origin], t)
			clone.SetUnderlying(visible(t.Underlying()))
			for i := 0; i < t.NumMethods(); i++ {
				method := t.Method(i)
				if method.Type().(*types.Signature).TypeParams().Len() == 0 {
					clone.AddMethod(method)
				}
			}
			return clone
		case *types.Pointer:
			return types.NewPointer(visible(t.Elem()))
		case *types.Struct:
			fields := make([]*types.Var, t.NumFields())
			tags := make([]string, t.NumFields())
			for i := range fields {
				field := t.Field(i)
				fieldType := field.Type()
				if field.Embedded() {
					fieldType = visible(fieldType)
				}
				fields[i] = types.NewField(field.Pos(), field.Pkg(), field.Name(), fieldType, field.Embedded())
				tags[i] = t.Tag(i)
			}
			return types.NewStruct(fields, tags)
		default:
			return t
		}
	}
	return types.NewMethodSet(visible(t))
}
