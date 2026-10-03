---
title: Plan constructor bindings
description: Type-check source, normalize lifetime directives and analyze exact constructor bindings before a product renderer emits registrations.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

If you maintain a generator, `dependencyinjection/bindingtypes` in **v0.2.0**
lets you share constructor-selection rules without sharing an emitter. It
accepts type-checked packages and returns a deterministic plan. It does **not**
load packages, execute constructors, register services, write files or provide
a generator CLI. Use [plain constructors or typed bindings](dependency-injection.md)
when you do not need generation.

## Type-check, read directives and analyze

Use Go 1.26 or later with `github.com/cratis/fundamentals.go@v0.2.0`. Save this
complete program as `main.go` and run `go run .`. It is derived from the
[executable planner example](../dependencyinjection/bindingtypes/documentation_examples_test.go).

```go
package main

import (
 "fmt"
 "go/ast"
 "go/parser"
 "go/token"
 "go/types"

 di "github.com/cratis/fundamentals.go/dependencyinjection"
 "github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
)

func main() {
 const source = `package reports
type Title string
// Report is prepared for one operation.
//cratis:scoped
type Report struct{ Title Title }
func NewReport(title Title) *Report { panic("analysis must not call me") }
`
 fset := token.NewFileSet()
 file, err := parser.ParseFile(fset, "reports.go", source, parser.ParseComments)
 if err != nil {
  panic(err)
 }
 info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
 checker := types.Config{}
 pkg, err := checker.Check("example/reports", fset, []*ast.File{file}, info)
 if err != nil {
  panic(err)
 }
 policies, diagnostics := bindingtypes.ReadDirectives([]*ast.File{file}, info)
 for _, diagnostic := range diagnostics {
  fmt.Println(diagnostic.Code, diagnostic.Message)
  if diagnostic.Severity == bindingtypes.Error {
   return
  }
 }
 plan := bindingtypes.Analyze([]*types.Package{pkg}, bindingtypes.Config{
  Policies: policies,
  Existing: []bindingtypes.Registration{{
   Service:  pkg.Scope().Lookup("Title").Type(),
   Lifetime: di.Singleton,
  }},
  RequireAllDependencies: true,
 })
 for _, diagnostic := range plan.Diagnostics {
  fmt.Println(diagnostic.Code, diagnostic.Message)
  if diagnostic.Severity == bindingtypes.Error {
   return
  }
 }
 for _, binding := range plan.Bindings {
  fmt.Println(binding.Service, "scoped:", binding.Lifetime == di.Scoped)
  fmt.Println("constructor:", binding.Constructor.Name(), "arguments:", len(binding.Arguments))
 }
}
```

Expected output:

```text
*example/reports.Report scoped: true
constructor: NewReport arguments: 1
```

The constructor body would panic if called, but analysis reads only metadata.
The directive selects `Scoped`; `Existing` attests that composition supplies a
singleton `Title`. It does **not** register a title value. Remove that entry to
see `BT012` (`MissingConfiguration`) instead of a usable plan. In a real generator,
return a failure to your caller on diagnostics with severity `Error` rather than
emitting output; this demonstration prints diagnostics and stops.

## Supply one type-checking universe

All input packages, policy types, explicit constructors, interface selections and
existing keys must come from the **same coherent importer/type-checking universe**.
Do not manufacture look-alike `types.Named` values or combine separate loads of
the same import path. The planner uses exact type identity, not printed names.

The example source has no imports and needs no importer. For real modules, your
loader must resolve imports and report all type-checking errors before analysis.
The [package-loading recipe](recipes.md#load-types-for-a-generator) demonstrates
`go/packages` outside the standard-library-only root. It is a recipe, not a
runtime dependency or loader supplied by `bindingtypes`.

`ReadDirectives(files []*ast.File, info *types.Info) ([]TypePolicy, []Diagnostic)`
needs AST comments (`parser.ParseComments`) and matching `types.Info.Defs`.
Export data preserves types, **not comments**. Supply source metadata or explicit
`TypePolicy` values for external packages; never infer a lifetime from export
data alone. Directives belong on type doc comments, with per-type comments in
grouped declarations:

- `//cratis:singleton`: one cached result per provider.
- `//cratis:scoped`: one cached result per scope.
- `//cratis:ignore-convention`: omit DI convention bindings, not product artifacts.
- No directive: transient. Unknown, misplaced, repeated or conflicting directives
  are errors, not ignored hints.

## Configure the analysis

`Analyze(pkgs []*types.Package, cfg Config) Plan` uses these fields:

| `Config` field | Type and default | Meaning |
| --- | --- | --- |
| `EmitPackage` | `*types.Package`, defaults to sole input | Rendering destination for accessibility; specify for multiple inputs |
| `Constructors` | `[]*types.Func`, nil discovers `NewX` | Non-nil is a whitelist, including an empty whitelist |
| `Policies` | `[]TypePolicy`, none | `Type *types.TypeName`, `Lifetime di.Lifetime` (zero means transient), `Ignore bool`, `Pos token.Pos`; aliases share target policy |
| `MatchIFoo` | `bool`, false | Opt into same-package `IFoo` → `Foo` matching |
| `Interfaces` | `[]InterfaceBinding`, none | Explicit `Service types.Type` interface and exact `Implementation types.Type` |
| `Existing` | `[]Registration`, none | Already registered `Service types.Type` and actual nonzero `Lifetime di.Lifetime` |
| `Duplicates` | `DuplicatePolicy`, `RejectDuplicates` | `KeepExisting` retains only explicitly attested existing keys |
| `RequireAllDependencies` | `bool`, false | Promote missing ordinary dependencies from obligations to errors |

Structural interface uniqueness is limited to supplied packages. Ignored and
constructorless competitors count; aliases and value/pointer forms belong to one
implementation family. Explicit mappings resolve ambiguity. Interface-returning
constructors register that interface directly; pointer and value keys stay distinct.

`KeepExisting` produces `RetainExisting` actions and `BT013` information for
overlapping keys, using their attested lifetimes. It does not excuse multiple
generated candidates. Preflight the manifest through runtime `Catalog` when
available; otherwise composition must attest it. One product should own shared
registrations and the other list them in `Existing`. Reapplying unchanged emitted
wiring is not idempotent: refresh the manifest and replan. Never swallow duplicate
registration errors.

## Consume a plan without changing its meaning

`Plan.Bindings` and `Plan.Diagnostics` are deterministically ordered, independently
owned slices. Referenced `go/types` objects are borrowed: treat them as immutable.
Concurrent analysis is safe only while callers do not mutate the inputs.

Each `Binding` supplies exact `Service`, `Lifetime`, `Ownership` and `Action`.
For `Register`, exactly one of `Constructor` and `Forward` is set. Constructor
`Arguments` preserve order and repetitions, excluding leading context;
`Dependencies` contain unique direct keys in first-argument order. `PassContext`
and `ReturnsError` tell your renderer which signature adaptation is needed.
`RetainExisting` has neither constructor nor forwarder and emits no registration.

Render zero-argument constructors through `di.Bind`; use `BindFunc1` through
`BindFunc4` for one through four arguments, adapting missing context/error
parameters. There is **no `BindFunc0`**. Larger arities require `di.Bind`, ordered
argument resolution and unique declared keys. Disposable value results with more
than four arguments are rejected: failed dependency resolution must not transfer
ownership of a never-constructed zero value. Always preserve a non-nil constructor
result returned alongside an error so owned cleanup can run.

Forward interfaces with `di.BindBorrowed`, resolving the exact concrete key and
preserving its lifetime. The concrete binding owns cleanup; forwarding must not
create a second owner. The [render-equivalence tests](../dependencyinjection/bindingtypes/render_test.go)
exercise this contract, not a public emitter. Products must compile and execute
their own generated output. Runtime `container.Registry.Build` remains responsible
for the final graph, cycles and captive lifetimes.

## Diagnose an unusable plan

Any `Error` diagnostic makes `Bindings` nil. Check directive diagnostics as well
as analysis diagnostics; never render a partial plan. `Diagnostic` exposes
`Code`, `Severity`, `Pos`, `Subject`, `Type`, `Related` and `Message`. Use stable
codes, not message text, for tooling; source positions need the original file set.

| Code and name | Corrective action |
| --- | --- |
| `BT001 InvalidInput` | Supply valid packages, configuration and metadata |
| `BT002 InvalidDirective` | Correct directive spelling, syntax or placement |
| `BT003 ConflictingPolicy` | Remove duplicate or conflicting policies |
| `BT004 ConstructorNotFound` | Supply a supported constructor for the requested key |
| `BT005 AmbiguousConstructor` | Select one constructor per exact key |
| `BT006 UnsupportedSignature` | Use a supported nongeneric, nonvariadic wrapper or a safe result shape |
| `BT007 InaccessibleDeclaration` | Choose a destination that can name the required declarations |
| `BT008 InvalidInterfaceBinding` | Select an ordinary named interface and compatible exact implementation |
| `BT009 AmbiguousImplementation` | Provide an explicit interface mapping |
| `BT010 DuplicateBinding` | Remove conflicting registrations or attest existing keys deliberately |
| `BT011 MissingDependency` | Supply the external key; informational unless all dependencies are required |
| `BT012 MissingConfiguration` | Explicitly provide or attest the scalar configuration key |
| `BT013 ExistingRegistrationRetained` | Information: emit nothing for the retained key |

Supported constructors return `T` or `(T, error)`, optionally accepting exact
`context.Context` first. Nongeneric wrappers may return closed generic types.
Open generic constructors, variadics and extra results are unsupported. There is
no richest-constructor selection, implicit `T{}` fallback, field injection or
runtime scanning. The [normative convention rules](dependency-injection.md#convention-based-bindings)
and [parity map](parity.md#dependency-injection-surfaces) describe the remaining
compatibility boundaries.
