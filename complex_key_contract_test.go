// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

// These tests validate historical evidence, not a Go complex-key codec.
// RawMessage keeps numbers out of float64 and leaves embedded JSON untouched.
type complexInventory struct {
	Count          int            `json:"count"`
	IDs            []string       `json:"ids"`
	Operations     map[string]int `json:"operations"`
	LookupOutcomes map[string]int `json:"lookupOutcomes"`
	SHA256         string         `json:"sha256"`
}

type complexManifest struct {
	SchemaVersion  int                         `json:"schemaVersion"`
	SourceRevision string                      `json:"sourceRevision"`
	Declarations   map[string]json.RawMessage  `json:"declarations"`
	Datasets       map[string]complexInventory `json:"datasets"`
	SourceFiles    map[string]string           `json:"sourceFiles"`
}

type complexRecord map[string]json.RawMessage

func complexDigest(data []byte) (string, error) {
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(compact.Bytes())), nil
}

func complexString(raw json.RawMessage) (string, error) {
	var value string
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", fmt.Errorf("missing string")
	}
	err := json.Unmarshal(raw, &value)
	return value, err
}

func complexOutcome(raw json.RawMessage, stage string, lookup bool) error {
	var value complexRecord
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return fmt.Errorf("missing %s outcome", stage)
	}
	gotStage, err := complexString(value["stage"])
	if err != nil || gotStage != stage {
		return fmt.Errorf("wrong %s stage", stage)
	}
	status, err := complexString(value["status"])
	if err != nil {
		return err
	}
	fields := 0
	switch status {
	case "accepted":
		fields = 3
		if len(value["value"]) == 0 {
			return fmt.Errorf("missing accepted value")
		}
	case "rejected":
		fields = 5
		category, err := complexString(value["exception"])
		if err != nil || category == "" {
			return fmt.Errorf("missing error category")
		}
		if _, err := complexString(value["message"]); err != nil {
			return err
		}
		if !bytes.Equal(value["innerException"], []byte("null")) {
			if _, err := complexString(value["innerException"]); err != nil {
				return err
			}
		}
	case "not-attempted":
		fields = 3
		reason, err := complexString(value["reason"])
		if err != nil || reason == "" {
			return fmt.Errorf("missing not-attempted reason")
		}
	default:
		return fmt.Errorf("unknown status %q", status)
	}
	if lookup {
		fields += 2
		mode, err := complexString(value["queryMode"])
		if err != nil || (mode != "fixed-key" && mode != "original-key") || len(value["lookupKey"]) == 0 {
			return fmt.Errorf("missing query mode/key")
		}
		if bytes.Equal(value["lookupKey"], []byte("null")) && (mode != "original-key" || status != "not-attempted") {
			return fmt.Errorf("invalid null lookup key")
		}
	}
	if len(value) != fields {
		return fmt.Errorf("unexpected outcome fields")
	}
	return nil
}

func loadComplexBundle(t *testing.T) (complexManifest, map[string][]byte) {
	t.Helper()
	raw, err := os.ReadFile("testdata/complex-key-contract/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := complexDigest(raw)
	if err != nil || digest != "8f0b94fab2ff2d57c040efa181535896ab291c8a64ca27a9bd8c85b65c42744c" {
		t.Fatal("source/type/ID inventory changed; recapture originals, never derive expectations from Go")
	}
	var manifest complexManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Declarations) != 6 || len(manifest.SourceFiles) != 49 || len(manifest.Datasets) != 4 {
		t.Fatal("invalid manifest")
	}
	files := make(map[string][]byte)
	for name := range manifest.Datasets {
		files[name], err = os.ReadFile("testdata/complex-key-contract/" + name)
		if err != nil {
			t.Fatal(err)
		}
	}
	return manifest, files
}

func validateComplexBundle(manifest complexManifest, files map[string][]byte) error {
	index := make(map[string]map[string]complexRecord)
	for name, inventory := range manifest.Datasets {
		raw := files[name]
		var rows []complexRecord
		if name == "inputs.json" {
			if err := json.Unmarshal(raw, &rows); err != nil {
				return err
			}
		} else {
			var capture struct {
				SchemaVersion  int             `json:"schemaVersion"`
				SourceRevision string          `json:"sourceRevision"`
				Count          int             `json:"count"`
				Observations   []complexRecord `json:"observations"`
			}
			if err := json.Unmarshal(raw, &capture); err != nil {
				return err
			}
			if capture.SchemaVersion != 1 || capture.SourceRevision != manifest.SourceRevision || capture.Count != inventory.Count {
				return fmt.Errorf("%s identity/count", name)
			}
			rows = capture.Observations
		}
		if len(rows) != inventory.Count || len(rows) != len(inventory.IDs) || len(rows) == 0 {
			return fmt.Errorf("%s truncated/empty rows", name)
		}
		index[name] = make(map[string]complexRecord)
		operations := make(map[string]int)
		lookups := make(map[string]int)
		for i, row := range rows {
			id, err := complexString(row["id"])
			if err != nil || id != inventory.IDs[i] || index[name][id] != nil {
				return fmt.Errorf("%s missing/duplicate/unexpected ID", name)
			}
			index[name][id] = row
			if name == "inputs.json" {
				operations["stimulus"]++
				continue
			}
			op, err := complexString(row["operation"])
			if err != nil {
				return err
			}
			operations[op]++
			if len(row["sourceCapture"]) == 0 || len(row["sourceCaseId"]) == 0 {
				return fmt.Errorf("missing explicit linkage")
			}
			stages := []string{}
			switch op {
			case "read":
				stages = []string{"read", "rewrite"}
				if string(row["returnedObject"]) != "true" && string(row["returnedObject"]) != "false" {
					return fmt.Errorf("missing returnedObject boolean")
				}
				if name == "javascript.json" {
					for _, field := range []string{"runtimeFieldAccess", "originalKeyLookup"} {
						if err := complexOutcome(row[field], "lookup", true); err != nil {
							return err
						}
						var lookup struct {
							Status string `json:"status"`
						}
						if err := json.Unmarshal(row[field], &lookup); err != nil {
							return err
						}
						lookups[field+"/"+lookup.Status]++
					}
				}
			case "write":
				stages = []string{"write"}
				if name == "csharp.json" {
					stages = append(stages, "construction")
				}
				if len(row["inputKey"]) == 0 || len(row["inputValue"]) == 0 {
					return fmt.Errorf("missing nullable write fields")
				}
			case "equality":
			default:
				return fmt.Errorf("unknown operation")
			}
			for _, stage := range stages {
				if err := complexOutcome(row[stage], stage, false); err != nil {
					return fmt.Errorf("%s/%s: %w", name, id, err)
				}
			}
		}
		if !reflect.DeepEqual(operations, inventory.Operations) || !reflect.DeepEqual(lookups, inventory.LookupOutcomes) {
			return fmt.Errorf("%s operation/lookup counts", name)
		}
	}
	for name, rows := range index {
		for id, row := range rows {
			if string(row["operation"]) != `"read"` {
				continue
			}
			sourceName, err := complexString(row["sourceCapture"])
			if err != nil {
				return err
			}
			sourceID, err := complexString(row["sourceCaseId"])
			if err != nil {
				return err
			}
			source := index[sourceName][sourceID]
			if source == nil || !bytes.Equal(row["kind"], source["kind"]) {
				return fmt.Errorf("%s/%s invalid source link", name, id)
			}
			var wire string
			if sourceName == "inputs.json" {
				wire, err = complexString(source["input"])
			} else {
				var write struct {
					Status string `json:"status"`
					Value  string `json:"value"`
				}
				err = json.Unmarshal(source["write"], &write)
				if err != nil || write.Status != "accepted" {
					return fmt.Errorf("link is not a successful write")
				}
				wire = write.Value
				if sourceName == "csharp.json" {
					wire = `{"map":` + wire + `}`
				}
			}
			input, inputErr := complexString(row["input"])
			if err != nil || inputErr != nil || input != wire {
				return fmt.Errorf("%s/%s cross input is not verbatim", name, id)
			}
		}
	}
	for name, inventory := range manifest.Datasets {
		digest, err := complexDigest(files[name])
		if err != nil || digest != inventory.SHA256 {
			return fmt.Errorf("%s captured bytes changed", name)
		}
	}
	return nil
}

func TestComplexKeyContractFixture(t *testing.T) {
	manifest, files := loadComplexBundle(t)
	if err := validateComplexBundle(manifest, files); err != nil {
		t.Fatal(err)
	}
}

func TestComplexKeyContractRejectsDamagedEvidence(t *testing.T) {
	manifest, files := loadComplexBundle(t)
	for _, tc := range []struct{ name, old, replacement string }{
		{"empty", `"count": 186`, `"count": 0`},
		{"boolean count", `"count": 186`, `"count": true`},
		{"duplicate ID", `"id": "string/null-map"`, `"id": "string/missing"`},
		{"missing nullable lookup", `"lookupKey": null,`, ``},
		{"missing not-attempted reason", `"reason": "no-independent-write-key"`, `"other": "no-independent-write-key"`},
		{"wrong stage", `"stage": "lookup"`, `"stage": "write"`},
		{"wrong cross input", `"sourceCaseId": "string/missing"`, `"sourceCaseId": "string/ordinary"`},
		{"number changed to bool", `"value": 42`, `"value": true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := files["javascript.json"]
			damaged := bytes.Replace(original, []byte(tc.old), []byte(tc.replacement), 1)
			if bytes.Equal(original, damaged) {
				t.Fatal("negative mutation did not apply")
			}
			copyFiles := make(map[string][]byte)
			for name, raw := range files {
				copyFiles[name] = raw
			}
			copyFiles["javascript.json"] = damaged
			if err := validateComplexBundle(manifest, copyFiles); err == nil {
				t.Fatal("damaged evidence accepted")
			}
		})
	}
}
