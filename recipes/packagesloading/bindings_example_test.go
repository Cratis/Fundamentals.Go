// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package packagesloading_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"go/types"
	"os"
	"slices"
	"strings"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
	"golang.org/x/tools/go/packages"
)

const bindingsFixturePath = "github.com/cratis/fundamentals.go/recipes/packagesloading/testdata/bindings"

// planConstructorBindings is a private recipe, not a supported loading API.
// Load all roots together: separate loads produce incompatible go/types identities.
// Operational failures return an empty plan; structural rejections stay diagnostics.
func planConstructorBindings(ctx context.Context, dir, emitPackagePath string, patterns ...string) (bindingtypes.Plan, error) {
	if err := ctx.Err(); err != nil {
		return bindingtypes.Plan{}, err
	}
	if len(patterns) == 0 || slices.ContainsFunc(patterns, func(pattern string) bool { return strings.TrimSpace(pattern) == "" }) {
		return bindingtypes.Plan{}, errors.New("supply nonempty package patterns")
	}
	loaded, err := packages.Load(&packages.Config{
		Context: ctx, Dir: dir, Mode: packages.LoadAllSyntax, Tests: false,
		// Match Inspect's offline environment, including disabling external drivers.
		Env: append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GONOPROXY=none", "GOSUMDB=off", "GOTOOLCHAIN=local", "GOFLAGS=-mod=readonly", "GOPACKAGESDRIVER=off"),
	}, patterns...)
	if contextErr := ctx.Err(); contextErr != nil {
		return bindingtypes.Plan{}, contextErr
	}
	if err != nil {
		return bindingtypes.Plan{}, fmt.Errorf("load constructor packages: %w", err)
	}
	if len(loaded) == 0 {
		return bindingtypes.Plan{}, errors.New("no packages matched")
	}
	var loadErrors []error
	packages.Visit(loaded, nil, func(pkg *packages.Package) {
		for _, diagnostic := range pkg.Errors {
			loadErrors = append(loadErrors, errors.New(diagnostic.Error()))
		}
		if pkg.IllTyped || pkg.Types == nil || !pkg.Types.Complete() {
			loadErrors = append(loadErrors, fmt.Errorf("incomplete type-checking for %s", pkg.ID))
		}
	})
	for _, pkg := range loaded {
		if pkg.TypesInfo == nil || pkg.TypesInfo.Types == nil || pkg.TypesInfo.Defs == nil || pkg.TypesInfo.Uses == nil || pkg.TypesInfo.Scopes == nil || len(pkg.Syntax) == 0 || len(pkg.Syntax) != len(pkg.CompiledGoFiles) || slices.Contains(pkg.Syntax, nil) {
			loadErrors = append(loadErrors, fmt.Errorf("incomplete source metadata for %s", pkg.ID))
		}
	}
	if err := errors.Join(loadErrors...); err != nil {
		return bindingtypes.Plan{}, err
	}
	slices.SortFunc(loaded, func(a, b *packages.Package) int {
		return cmp.Or(cmp.Compare(a.PkgPath, b.PkgPath), cmp.Compare(a.ID, b.ID))
	})
	var roots []*types.Package
	var policies []bindingtypes.TypePolicy
	var diagnostics []bindingtypes.Diagnostic
	for _, pkg := range loaded {
		rootPolicies, rootDiagnostics := bindingtypes.ReadDirectives(pkg.Syntax, pkg.TypesInfo)
		policies = append(policies, rootPolicies...)
		diagnostics = append(diagnostics, rootDiagnostics...)
		roots = append(roots, pkg.Types)
	}
	var emit *types.Package
	for _, pkg := range loaded {
		if pkg.PkgPath == emitPackagePath {
			if emit != nil {
				return bindingtypes.Plan{}, fmt.Errorf("ambiguous emit package %q", emitPackagePath)
			}
			emit = pkg.Types
		}
	}
	if emitPackagePath == "" || emit == nil {
		return bindingtypes.Plan{}, fmt.Errorf("emit package %q must match exactly one root import path", emitPackagePath)
	}
	if err := ctx.Err(); err != nil {
		return bindingtypes.Plan{}, err
	}
	if slices.ContainsFunc(diagnostics, func(d bindingtypes.Diagnostic) bool { return d.Severity == bindingtypes.Error }) {
		// Do not analyze rejected policies or expose any bindings from other roots.
		return bindingtypes.Plan{Diagnostics: diagnostics}, nil
	}
	// Imported packages share identities, but only requested roots supply constructors.
	// Nil Constructors selects Analyze's exact NewX convention, not arbitrary factories.
	plan := bindingtypes.Analyze(roots, bindingtypes.Config{
		EmitPackage: emit, Constructors: nil, Policies: policies,
	})
	// Analyze is synchronous and cannot be preempted by context cancellation.
	if err := ctx.Err(); err != nil {
		return bindingtypes.Plan{}, err
	}
	return plan, nil
}

func Example_constructorBindings() {
	plan, err := planConstructorBindings(context.Background(), ".", bindingsFixturePath+"/service",
		"./testdata/bindings/repository", "./testdata/bindings/service")
	if err != nil {
		panic(err)
	}
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Severity == bindingtypes.Error {
			panic(fmt.Sprintf("%s: %s", diagnostic.Code, diagnostic.Message))
		}
	}
	lifetimes := map[di.Lifetime]string{di.Singleton: "singleton", di.Scoped: "scoped", di.Transient: "transient"}
	ownerships := map[di.Ownership]string{di.Owned: "owned", di.Borrowed: "borrowed"}
	fmt.Println("constructors:", len(plan.Bindings))
	for _, binding := range plan.Bindings {
		action := "register"
		if binding.Action != bindingtypes.Register {
			panic("expected constructor registration")
		}
		fmt.Printf("%s: lifetime=%s ownership=%s action=%s arguments=%d dependencies=%d returns-error=%t\n",
			binding.Constructor.Name(), lifetimes[binding.Lifetime], ownerships[binding.Ownership], action,
			len(binding.Arguments), len(binding.Dependencies), binding.ReturnsError)
	}
	// Output:
	// constructors: 2
	// NewStore: lifetime=singleton ownership=owned action=register arguments=0 dependencies=0 returns-error=false
	// NewService: lifetime=scoped ownership=owned action=register arguments=2 dependencies=1 returns-error=true
}
