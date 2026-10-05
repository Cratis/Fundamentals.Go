// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package documentationcheck

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Reuse the unpublished module's YAML parser; the root stays stdlib-only.
func TestRecipeDependabotPolicy(t *testing.T) {
	data, err := os.ReadFile("../../.github/dependabot.yml")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkRecipeUpdates(data); err != nil {
		t.Fatal(err)
	}
}

func checkRecipeUpdates(data []byte) error {
	var config struct {
		Version int `yaml:"version"`
		Updates []struct {
			Ecosystem string                    `yaml:"package-ecosystem"`
			Directory string                    `yaml:"directory"`
			Schedule  struct{ Interval string } `yaml:"schedule"`
			Labels    []string                  `yaml:"labels"`
			Limit     int                       `yaml:"open-pull-requests-limit"`
			Groups    map[string]struct {
				UpdateTypes []string `yaml:"update-types"`
			} `yaml:"groups"`
			Ignore []struct {
				Name        string   `yaml:"dependency-name"`
				Versions    []string `yaml:"versions"`
				UpdateTypes []string `yaml:"update-types"`
			} `yaml:"ignore"`
		} `yaml:"updates"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	if config.Version != 2 {
		return fmt.Errorf("Dependabot configuration must use version 2")
	}
	found := 0
	for _, update := range config.Updates {
		if update.Ecosystem != "gomod" || update.Directory != "/recipes" {
			continue
		}
		found++
		if update.Schedule.Interval != "weekly" || update.Limit < 1 || update.Limit > 5 {
			return fmt.Errorf("recipe updates must be weekly with 1–5 open PRs")
		}
		labels := slices.Clone(update.Labels)
		slices.Sort(labels)
		if !slices.Equal(labels, []string{"dependencies", "no-release"}) {
			return fmt.Errorf("recipe updates need dependencies and only no-release intent")
		}
		group, ok := update.Groups["go-non-major"]
		kinds := slices.Clone(group.UpdateTypes)
		slices.Sort(kinds)
		if !ok || !slices.Equal(kinds, []string{"minor", "patch"}) {
			return fmt.Errorf("recipe minor/patch updates must be grouped")
		}
		rootIgnored := false
		for _, ignored := range update.Ignore {
			if ignored.Name == "github.com/cratis/fundamentals.go" && len(ignored.Versions) == 0 && len(ignored.UpdateTypes) == 0 {
				rootIgnored = true
			}
		}
		if !rootIgnored {
			return fmt.Errorf("all root-module version updates must be ignored to preserve the local recipe placeholder")
		}
	}
	if found != 1 {
		return fmt.Errorf("want exactly one /recipes Go update entry, got %d", found)
	}
	return nil
}

func TestRecipeDependabotPolicyRejectsRegressions(t *testing.T) {
	const valid = `version: 2
updates:
  - package-ecosystem: gomod
    directory: /recipes
    schedule: {interval: weekly}
    labels: [dependencies, no-release]
    open-pull-requests-limit: 5
    groups:
      go-non-major:
        update-types: [minor, patch]
    ignore:
      - dependency-name: github.com/cratis/fundamentals.go
`
	if err := checkRecipeUpdates([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, old, replacement string }{
		{"missing recipes", "directory: /recipes", "directory: /"},
		{"wrong ecosystem", "package-ecosystem: gomod", "package-ecosystem: npm"},
		{"disabled updates", "limit: 5", "limit: 0"},
		{"unbounded policy", "limit: 5", "limit: 6"},
		{"daily updates", "interval: weekly", "interval: daily"},
		{"release label", "no-release", "patch"},
		{"extra release label", "dependencies, no-release", "dependencies, no-release, minor"},
		{"major grouping", "[minor, patch]", "[major, minor, patch]"},
		{"missing root ignore", "dependency-name: github.com/cratis/fundamentals.go", "dependency-name: another/module"},
		{"partial root ignore", "dependency-name: github.com/cratis/fundamentals.go", "dependency-name: github.com/cratis/fundamentals.go\n        update-types: [version-update:semver-major]"},
		{"version-specific root ignore", "dependency-name: github.com/cratis/fundamentals.go", "dependency-name: github.com/cratis/fundamentals.go\n        versions: [v0.0.0]"},
		{"duplicate YAML key", "version: 2", "version: 2\nversion: 2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Count(valid, tc.old) != 1 {
				t.Fatal("regression must change exactly one target")
			}
			broken := strings.Replace(valid, tc.old, tc.replacement, 1)
			if err := checkRecipeUpdates([]byte(broken)); err == nil {
				t.Fatal("accepted broken update policy")
			}
		})
	}
	t.Run("duplicate recipes", func(t *testing.T) {
		duplicate := valid + strings.TrimPrefix(valid, "version: 2\nupdates:\n")
		err := checkRecipeUpdates([]byte(duplicate))
		if err == nil || err.Error() != "want exactly one /recipes Go update entry, got 2" {
			t.Fatalf("error = %v, want duplicate recipe entry rejection", err)
		}
	})
}
