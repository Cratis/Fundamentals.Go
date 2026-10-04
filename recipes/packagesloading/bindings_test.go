// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package packagesloading_test

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
)

func TestConstructorPlanUsesOneTypeUniverseDeterministically(t *testing.T) {
	patterns := []string{"./testdata/bindings/repository", "./testdata/bindings/service"}
	forward, err := planConstructorBindings(t.Context(), ".", bindingsFixturePath+"/service", patterns...)
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := planConstructorBindings(t.Context(), ".", bindingsFixturePath+"/service", patterns[1], patterns[0])
	if err != nil {
		t.Fatal(err)
	}
	if got, want := normalizedConstructorPlan(reverse), normalizedConstructorPlan(forward); got != want {
		t.Fatalf("reversed patterns changed semantics:\n%s\nwant:\n%s", got, want)
	}
	if len(forward.Bindings) != 2 || len(forward.Diagnostics) != 0 {
		t.Fatalf("plan = %s, want two bindings without diagnostics", normalizedConstructorPlan(forward))
	}
	store, service := forward.Bindings[0], forward.Bindings[1]
	if store.Constructor.Name() != "NewStore" || store.Lifetime != di.Singleton || store.Ownership != di.Owned || store.Action != bindingtypes.Register || store.PassContext || store.ReturnsError || len(store.Arguments) != 0 || len(store.Dependencies) != 0 || store.Forward != nil {
		t.Fatalf("unexpected store metadata: %s", normalizedConstructorPlan(forward))
	}
	if service.Constructor.Name() != "NewService" || service.Lifetime != di.Scoped || service.Ownership != di.Owned || service.Action != bindingtypes.Register || service.PassContext || !service.ReturnsError || len(service.Arguments) != 2 || len(service.Dependencies) != 1 || service.Forward != nil {
		t.Fatalf("unexpected service metadata: %s", normalizedConstructorPlan(forward))
	}
	for _, argument := range service.Arguments {
		if !types.Identical(argument, store.Service) {
			t.Fatalf("argument %s does not share the repository binding's exact type identity", argument)
		}
	}
	if !types.Identical(service.Dependencies[0], store.Service) {
		t.Fatal("unique dependency does not share the repository binding's exact type identity")
	}
}

func TestConstructorPlanOperationalFailures(t *testing.T) {
	const emit = bindingsFixturePath + "/service"
	valid := []string{"./testdata/bindings/repository", "./testdata/bindings/service"}
	cases := []struct {
		name     string
		dir      string
		emit     string
		patterns []string
		source   string
	}{
		{name: "empty patterns", dir: ".", emit: emit},
		{name: "blank pattern", dir: ".", emit: emit, patterns: []string{""}},
		{name: "invalid directory", dir: filepath.Join(t.TempDir(), "absent"), emit: emit, patterns: valid},
		{name: "valid plus missing root", dir: ".", emit: emit, patterns: append(append([]string{}, valid...), "./testdata/absent")},
		{name: "no matches", dir: ".", emit: emit, patterns: []string{"./testdata/absent/..."}},
		{name: "syntax", source: "package fixture\ntype Broken struct {\n"},
		{name: "type checking", source: "package fixture\nvar Broken int = \"not an int\"\n"},
		{name: "broken import", source: "package fixture\nimport _ \"missing.invalid/constructor-recipe\"\n"},
		{name: "missing emit", dir: ".", emit: bindingsFixturePath + "/absent", patterns: valid},
		{name: "empty emit", dir: ".", patterns: valid},
		{name: "import is not an emit root", dir: ".", emit: bindingsFixturePath + "/repository", patterns: []string{"./testdata/bindings/service"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.source != "" {
				tc.dir, tc.emit, tc.patterns = constructorSourceQuery(t, tc.source)
			}
			plan, err := planConstructorBindings(t.Context(), tc.dir, tc.emit, tc.patterns...)
			if err == nil || !reflect.DeepEqual(plan, bindingtypes.Plan{}) {
				t.Fatalf("operational failure = %s, %v; want empty plan and error", normalizedConstructorPlan(plan), err)
			}
		})
	}
}

func TestConstructorPlanStructuralDiagnostics(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		code     bindingtypes.Code
		severity bindingtypes.Severity
		bindings int
	}{
		{name: "unsupported constructor", source: "package fixture\ntype Thing struct{}\nfunc NewThing() (*Thing, bool) { panic(\"not activation\") }\n", code: bindingtypes.UnsupportedSignature, severity: bindingtypes.Error},
		// The invalid constructor must not be analyzed after directive rejection.
		{name: "invalid directive", source: "package fixture\n//cratis:unknown\ntype Thing struct{}\nfunc NewThing() (*Thing, bool) { panic(\"not activation\") }\n", code: bindingtypes.InvalidDirective, severity: bindingtypes.Error},
		{name: "ordinary external obligation", source: "package fixture\ntype Dependency struct{}\ntype Thing struct{}\nfunc NewThing(dep *Dependency) *Thing { panic(\"not activation\") }\n", code: bindingtypes.MissingDependency, severity: bindingtypes.Information, bindings: 1},
		{name: "scalar configuration", source: "package fixture\ntype Thing struct{}\nfunc NewThing(value string) *Thing { panic(\"not activation\") }\n", code: bindingtypes.MissingConfiguration, severity: bindingtypes.Error},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, emit, patterns := constructorSourceQuery(t, tc.source)
			plan, err := planConstructorBindings(t.Context(), dir, emit, patterns...)
			if err != nil {
				t.Fatalf("structural rejection returned operational error: %v", err)
			}
			if len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != tc.code || plan.Diagnostics[0].Severity != tc.severity || len(plan.Bindings) != tc.bindings {
				t.Fatalf("plan = %s, want %s/%s and %d bindings", normalizedConstructorPlan(plan), tc.code, tc.severity, tc.bindings)
			}
			if tc.severity == bindingtypes.Error && plan.Bindings != nil {
				t.Fatal("error diagnostics must leave bindings nil")
			}
		})
	}
}

func TestConstructorPlanPreCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	plan, err := planConstructorBindings(ctx, ".", bindingsFixturePath+"/service", "./testdata/bindings/service")
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(plan, bindingtypes.Plan{}) {
		t.Fatalf("canceled planning = %s, %v; want empty plan and context.Canceled", normalizedConstructorPlan(plan), err)
	}
}

// A file query uses the existing recipes module without creating another go.mod.
func constructorSourceQuery(t *testing.T, source string) (string, string, []string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "fixture.go")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return ".", "command-line-arguments", []string{"file=" + file}
}

// Normalize semantic metadata only: type pointers and token positions differ per load.
func normalizedConstructorPlan(plan bindingtypes.Plan) string {
	var result strings.Builder
	identity := func(typ types.Type) string {
		if typ == nil {
			return ""
		}
		return types.TypeString(typ, func(pkg *types.Package) string { return pkg.Path() })
	}
	for _, binding := range plan.Bindings {
		constructor := ""
		if binding.Constructor != nil {
			constructor = binding.Constructor.Pkg().Path() + "." + binding.Constructor.Name()
		}
		fmt.Fprintf(&result, "%s constructor=%s lifetime=%d ownership=%d action=%d context=%t error=%t forward=%s\n",
			identity(binding.Service), constructor, binding.Lifetime, binding.Ownership, binding.Action, binding.PassContext, binding.ReturnsError, identity(binding.Forward))
		for _, argument := range binding.Arguments {
			fmt.Fprintf(&result, " argument=%s\n", identity(argument))
		}
		for _, dependency := range binding.Dependencies {
			fmt.Fprintf(&result, " dependency=%s\n", identity(dependency))
		}
	}
	for _, diagnostic := range plan.Diagnostics {
		fmt.Fprintf(&result, "%s/%s type=%s\n", diagnostic.Code, diagnostic.Severity, identity(diagnostic.Type))
	}
	return result.String()
}
