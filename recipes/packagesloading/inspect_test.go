// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package packagesloading_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

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
