// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package packagesloading_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/recipes/packagesloading"
)

func TestLoadedExportedTypes(t *testing.T) {
	got, err := packagesloading.Inspect(t.Context(), ".", "./testdata/model")
	if err != nil {
		t.Fatal(err)
	}
	want := []packagesloading.Concept{{Name: "ID", Kind: concepts.KindUUID}, {Name: "Name", Kind: concepts.KindString}}
	if !slices.Equal(got, want) {
		t.Fatalf("concepts = %v, want %v", got, want)
	}
}

func TestExternalPackageDriversNeverExecute(t *testing.T) {
	dir := t.TempDir()
	name := "gopackagesdriver"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	driver := filepath.Join(dir, name)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", driver, "./testdata/driver")
	build.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GONOPROXY=none", "GOSUMDB=off", "GOTOOLCHAIN=local", "GOENV=off", "GOFLAGS=-mod=readonly")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build driver probe: %v\n%s", err, output)
	}

	cases := []struct {
		name   string
		driver string
	}{
		{name: "explicit environment", driver: driver},
		{name: "PATH discovery"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "invoked")
			t.Setenv("PACKAGESLOADING_DRIVER_MARKER", marker)
			t.Setenv("GOPACKAGESDRIVER", tc.driver)
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("GOENV", "off")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			got, err := packagesloading.Inspect(ctx, ".", "./testdata/model")
			plan, planErr := planConstructorBindings(ctx, ".", bindingsFixturePath+"/service",
				"./testdata/bindings/repository", "./testdata/bindings/service")
			if planErr != nil || len(plan.Bindings) != 2 || len(plan.Diagnostics) != 0 {
				t.Errorf("constructor planning = %s, %v; want two bindings without diagnostics", normalizedConstructorPlan(plan), planErr)
			}
			if _, markerErr := os.Stat(marker); !errors.Is(markerErr, os.ErrNotExist) {
				t.Errorf("external driver must not execute; marker stat = %v", markerErr)
			}
			if err != nil {
				t.Fatal(err)
			}
			want := []packagesloading.Concept{{Name: "ID", Kind: concepts.KindUUID}, {Name: "Name", Kind: concepts.KindString}}
			if !slices.Equal(got, want) {
				t.Fatalf("concepts = %v, want %v", got, want)
			}
		})
	}
}

func TestMalformedConceptFails(t *testing.T) {
	got, err := packagesloading.Inspect(t.Context(), ".", "./testdata/invalid")
	if !errors.Is(err, concepts.ErrInvalidConcept) || got != nil {
		t.Fatalf("malformed concept = %v, %v", got, err)
	}
}

func TestMissingPackageFails(t *testing.T) {
	got, err := packagesloading.Inspect(t.Context(), ".", "./testdata/absent")
	if err == nil || got != nil {
		t.Fatalf("missing package = %v, %v", got, err)
	}
}

func TestUncachedPrivateDependencyFailsWithoutFetching(t *testing.T) {
	const dependency = "private.invalid/dependency"
	// Synthetic checksums allow lookup instead of an earlier missing-go.sum error.
	// No dependency bytes may be fetched or reach checksum verification.
	const checksum = "h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="
	cases := []struct {
		name    string
		noProxy string
		private string
	}{
		{name: "GONOPROXY", noProxy: dependency},
		{name: "GOPRIVATE default", private: dependency},
		{name: "both", noProxy: dependency, private: dependency},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.NotFound(w, r)
			}))
			defer server.Close()

			// Route any attempted module discovery to loopback, never the network.
			t.Setenv("HTTP_PROXY", server.URL)
			t.Setenv("HTTPS_PROXY", server.URL)
			t.Setenv("NO_PROXY", "")
			t.Setenv("GOINSECURE", dependency)
			t.Setenv("GONOPROXY", tc.noProxy)
			t.Setenv("GOPRIVATE", tc.private)
			t.Setenv("GOPROXY", server.URL)
			t.Setenv("GOSUMDB", "sum.golang.org")
			t.Setenv("GOENV", "off")
			t.Setenv("GOMODCACHE", t.TempDir())

			dir := t.TempDir()
			files := map[string]string{
				"go.mod":   "module offline.test/model\n\ngo 1.26.0\n\nrequire " + dependency + " v1.0.0\n",
				"go.sum":   dependency + " v1.0.0 " + checksum + "\n" + dependency + " v1.0.0/go.mod " + checksum + "\n",
				"model.go": "package model\n\nimport _ \"" + dependency + "\"\n",
			}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			got, err := packagesloading.Inspect(ctx, dir, ".")
			if got != nil || err == nil {
				t.Fatalf("uncached private dependency = %v, %v; want nil result and an error", got, err)
			}
			if count := requests.Load(); count != 0 {
				t.Errorf("attempted %d HTTP requests while loading offline", count)
			}
			if !strings.Contains(err.Error(), "module lookup disabled by GOPROXY=off") {
				t.Errorf("error = %v, want an offline module lookup diagnostic", err)
			}
		})
	}
}

func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if got, err := packagesloading.Inspect(ctx, ".", "./testdata/model"); err == nil || got != nil {
		t.Fatalf("canceled load = %v, %v", got, err)
	}
}

func ExampleInspect() {
	found, err := packagesloading.Inspect(context.Background(), ".", "./testdata/model")
	if err != nil {
		panic(err)
	}
	for _, concept := range found {
		fmt.Println(concept.Name, concept.Kind)
	}
	// Output:
	// ID uuid
	// Name string
}
