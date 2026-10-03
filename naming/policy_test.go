// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package naming_test

import (
	"testing"

	"github.com/cratis/fundamentals.go/naming"
)

type policyFixture struct {
	Label, Policy, Namespace, Name, Property   string
	Separator, Prefix, WantModel, WantProperty string
	Source, Deviation                          string
	Skip                                       int
	Camel                                      bool
}

type policyFixtures struct {
	Revision string
	Cases    []policyFixture
}

func TestPolicyGolden(t *testing.T) {
	fixtures := loadFixture[policyFixtures](t, "testdata/policy.json")
	if fixtures.Revision != "d2accc4a79b6bcf2708213c97093ab5ba6c06381" || len(fixtures.Cases) != 17 {
		t.Fatalf("unexpected authority or fixture count: %s, %d", fixtures.Revision, len(fixtures.Cases))
	}
	for _, fixture := range fixtures.Cases {
		t.Run(fixture.Label, func(t *testing.T) {
			checkSource(t, fixture.Source)
			var policy naming.Policy
			switch fixture.Policy {
			case "default":
				policy = naming.Default{}
			case "camel":
				policy = naming.CamelCasePolicy{}
			case "namespaced-zero":
				policy = naming.Namespaced{}
			case "namespaced":
				separator := []rune(fixture.Separator)
				if len(separator) != 1 {
					t.Fatal("fixture separator must be one rune")
				}
				policy = naming.NewNamespaced(fixture.Skip, separator[0], fixture.Prefix, fixture.Camel)
			default:
				t.Fatalf("unknown fixture policy: %q", fixture.Policy)
			}
			if got := policy.GetPropertyName(fixture.Property); got != fixture.WantProperty {
				t.Errorf("GetPropertyName(%q) = %q, want %q (%s)", fixture.Property, got, fixture.WantProperty, fixture.Source)
			}
			if got := policy.GetReadModelName(fixture.Namespace, fixture.Name); got != fixture.WantModel {
				t.Errorf("GetReadModelName(%q, %q) = %q, want %q (%s)", fixture.Namespace, fixture.Name, got, fixture.WantModel, fixture.Source)
			}
		})
	}
}

func TestNamespacedInvalidSeparator(t *testing.T) {
	for _, separator := range []rune{-1, 0xd800, 0x110000} {
		policy := naming.NewNamespaced(0, separator, "", false)
		if got := policy.GetReadModelName("Cratis.Models", "Person"); got != "cratis�models-Person" {
			t.Errorf("separator %U: got %q, want replacement rune", separator, got)
		}
	}
}
