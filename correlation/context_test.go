// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package correlation_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/correlation"
)

const fixture = "00112233-4455-6677-8899-aabbccddeeff"

// These signatures require ID to be an alias, not a second named UUID type.
var (
	_ func(context.Context, concepts.UUID) context.Context = correlation.WithID
	_ func(context.Context) concepts.UUID                  = correlation.FromContext
)

func fixtureID(t *testing.T) concepts.UUID {
	t.Helper()
	id, err := concepts.ParseUUID(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestFromContextWhenAbsent(t *testing.T) {
	if got := correlation.FromContext(context.Background()); got != (correlation.ID{}) {
		t.Errorf("absent ID = %v, want zero", got)
	}
}

func TestWithIDRoundTripPreservesRFCBytes(t *testing.T) {
	id := fixtureID(t)
	ctx := correlation.WithID(context.Background(), id)
	got := correlation.FromContext(ctx)
	if got != id || got.String() != fixture {
		t.Errorf("round trip = %v, want %s", got, fixture)
	}
	want := [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	if [16]byte(got) != want {
		t.Errorf("RFC bytes = %x, want %x", [16]byte(got), want)
	}
}

func TestWithIDExplicitZeroShadowsInheritedID(t *testing.T) {
	id := fixtureID(t)
	parent := correlation.WithID(context.Background(), id)
	child := correlation.WithID(parent, correlation.ID{})
	if got := correlation.FromContext(child); got != (correlation.ID{}) {
		t.Errorf("child ID = %v, want zero", got)
	}
	if got := correlation.FromContext(parent); got != id {
		t.Errorf("parent ID = %v, want %v", got, id)
	}
}

func TestWithIDNestedOverrideLeavesParentUnmodified(t *testing.T) {
	root := context.Background()
	first := fixtureID(t)
	second := correlation.ID{15: 1}
	parent := correlation.WithID(root, first)
	child := correlation.WithID(parent, second)
	nested := context.WithValue(child, unrelatedKey{}, "value")
	if got := correlation.FromContext(nested); got != second {
		t.Errorf("nested ID = %v, want %v", got, second)
	}
	if got := correlation.FromContext(parent); got != first {
		t.Errorf("parent ID = %v, want %v", got, first)
	}
	if got := correlation.FromContext(root); got != (correlation.ID{}) {
		t.Errorf("root ID = %v, want zero", got)
	}
}

type unrelatedKey struct{}

func TestWithIDPreservesContextLifetimeAndValues(t *testing.T) {
	deadline := time.Now().Add(time.Hour)
	parent, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	parent = context.WithValue(parent, unrelatedKey{}, "unrelated")
	child := correlation.WithID(parent, fixtureID(t))
	if got, ok := child.Deadline(); !ok || !got.Equal(deadline) {
		t.Errorf("deadline = %v, %v, want %v, true", got, ok, deadline)
	}
	if child.Done() != parent.Done() {
		t.Error("cancellation channel changed")
	}
	if err := child.Err(); err != nil {
		t.Errorf("active context error = %v, want nil", err)
	}
	if got := child.Value(unrelatedKey{}); got != "unrelated" {
		t.Errorf("unrelated value = %v, want unrelated", got)
	}
	cancel()
	select {
	case <-child.Done():
	default:
		t.Error("parent cancellation did not reach child")
	}
	if err := child.Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled context error = %v, want context.Canceled", err)
	}
}

func TestWithIDPreservesExpiredDeadline(t *testing.T) {
	parent, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	defer cancel()
	id := fixtureID(t)
	child := correlation.WithID(parent, id)
	select {
	case <-child.Done():
	default:
		t.Error("expired deadline did not reach child")
	}
	if err := child.Err(); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expired context error = %v, want context.DeadlineExceeded", err)
	}
	if got := correlation.FromContext(child); got != id {
		t.Errorf("expired context ID = %v, want %v", got, id)
	}
}

func TestFromContextReadsNeverGenerate(t *testing.T) {
	id := fixtureID(t)
	parent := correlation.WithID(context.Background(), id)
	cases := []struct {
		name string
		ctx  context.Context
		want correlation.ID
	}{
		{name: "absent", ctx: context.Background()},
		{name: "installed", ctx: parent, want: id},
		{name: "explicit zero", ctx: correlation.WithID(parent, correlation.ID{})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for range 100 {
				if got := correlation.FromContext(tc.ctx); got != tc.want {
					t.Fatalf("read ID = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestWithIDConcurrentContextsDoNotLeak(t *testing.T) {
	parentID := fixtureID(t)
	parent := correlation.WithID(context.Background(), parentID)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index := range 32 {
		workers.Go(func() {
			<-start
			id := correlation.ID{15: byte(index + 1)}
			child := correlation.WithID(parent, id)
			for range 100 {
				if got := correlation.FromContext(child); got != id {
					t.Errorf("operation %d ID = %v, want %v", index, got, id)
					return
				}
				if got := correlation.FromContext(parent); got != parentID {
					t.Errorf("operation %d parent ID = %v, want %v", index, got, parentID)
					return
				}
			}
		})
	}
	close(start)
	workers.Wait()
}

func TestWithIDCopiesValue(t *testing.T) {
	id := fixtureID(t)
	want := id
	ctx := correlation.WithID(context.Background(), id)
	id[0] = 0xff
	got := correlation.FromContext(ctx)
	got[0] = 0xee
	if stored := correlation.FromContext(ctx); stored != want {
		t.Errorf("stored ID = %v, want %v", stored, want)
	}
}

func TestWithIDNilContextPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("WithID(nil) did not panic")
		}
	}()
	var ctx context.Context
	correlation.WithID(ctx, correlation.ID{})
}

func TestFromContextNilContextPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("FromContext(nil) did not panic")
		}
	}()
	var ctx context.Context
	correlation.FromContext(ctx)
}
