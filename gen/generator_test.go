// Copyright © 2023 Cisco Systems, Inc. and its affiliates.
// All rights reserved.
//
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: MPL-2.0

// Tests for the generator merge logic.
//
// generator.go uses //go:build ignore (it is a go run tool, not a library),
// so these tests must be run with explicit file arguments:
//
//	go test -v generator.go generator_test.go
//
// Running "go test ./gen/..." will not find them.

//go:build ignore

package main

import (
	"testing"
)

// ---------------------------------------------------------------------------
// mergeConfigs tests
// ---------------------------------------------------------------------------

func TestMergeConfigs_BasicFieldOverride(t *testing.T) {
	base := YamlConfig{
		Name: "Logging",
		Path: "Cisco-IOS-XR-um-logging-cfg:/logging",
	}
	override := YamlConfig{
		Version:    "25.4",
		ResDescription: "Updated description",
		DocCategory: "Logging",
	}

	got := mergeConfigs(base, override)

	if got.Name != "Logging" {
		t.Errorf("Name: got %q, want %q", got.Name, "Logging")
	}
	if got.Path != "Cisco-IOS-XR-um-logging-cfg:/logging" {
		t.Errorf("Path: got %q, should be unchanged", got.Path)
	}
	if got.ResDescription != "Updated description" {
		t.Errorf("ResDescription: got %q, want %q", got.ResDescription, "Updated description")
	}
	if got.DocCategory != "Logging" {
		t.Errorf("DocCategory: got %q, want %q", got.DocCategory, "Logging")
	}
}

func TestMergeConfigs_NameOverride(t *testing.T) {
	base := YamlConfig{Name: "Service Timestamps Old", Path: "old-module:/service/timestamps"}
	override := YamlConfig{Version: "25.4", Name: "Service Timestamps", Path: "new-module:/service/timestamps"}

	got := mergeConfigs(base, override)

	if got.Name != "Service Timestamps" {
		t.Errorf("Name: got %q, want %q", got.Name, "Service Timestamps")
	}
	if got.Path != "new-module:/service/timestamps" {
		t.Errorf("Path: got %q, want %q", got.Path, "new-module:/service/timestamps")
	}
}

func TestMergeConfigs_PathUnchanged(t *testing.T) {
	path := "Cisco-IOS-XR-um-logging-cfg:/logging"
	base := YamlConfig{Name: "Logging", Path: path}
	override := YamlConfig{Version: "25.4", ResDescription: "New desc"}

	got := mergeConfigs(base, override)

	if got.Path != path {
		t.Errorf("Path: got %q, want %q (unchanged)", got.Path, path)
	}
}

func TestMergeConfigs_LegacyResource(t *testing.T) {
	base := YamlConfig{
		Name: "Old Feature",
		Path: "old-module:/feature",
		Attributes: []YamlConfigAttribute{
			{YangName: "attr1", TfName: "attr1", Type: "String"},
		},
	}
	override := YamlConfig{Version: "25.4", Legacy: true}

	got := mergeConfigs(base, override)

	if got.RemovedInVersion != "25.4" {
		t.Errorf("RemovedInVersion: got %q, want %q", got.RemovedInVersion, "25.4")
	}
	if !got.Legacy {
		t.Error("Legacy: got false, want true")
	}
	// Attributes should be unchanged — early return preserves base state
	if len(got.Attributes) != 1 {
		t.Errorf("Attributes: got %d, want 1 (preserved from base)", len(got.Attributes))
	}
}

func TestMergeConfigs_NoDeletePropagates(t *testing.T) {
	base := YamlConfig{Name: "Feature", NoDelete: false}
	override := YamlConfig{Version: "25.4", NoDelete: true}

	got := mergeConfigs(base, override)

	if !got.NoDelete {
		t.Error("NoDelete: got false, want true")
	}
}

// ---------------------------------------------------------------------------
// mergeAttributes tests
// ---------------------------------------------------------------------------

func TestMergeAttributes_NewAttrGetsAddedInVersion(t *testing.T) {
	base := []YamlConfigAttribute{
		{YangName: "console", TfName: "console", Type: "String"},
	}
	override := []YamlConfigAttribute{
		{YangName: "new-leaf", TfName: "new_leaf", Type: "String"},
	}

	got := mergeAttributes(base, override, "25.4")

	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2", len(got))
	}
	newAttr := got[1]
	if newAttr.YangName != "new-leaf" {
		t.Errorf("YangName: got %q, want %q", newAttr.YangName, "new-leaf")
	}
	if newAttr.AddedInVersion != "25.4" {
		t.Errorf("AddedInVersion: got %q, want %q", newAttr.AddedInVersion, "25.4")
	}
}

func TestMergeAttributes_LegacyAttrGetsRemovedInVersion(t *testing.T) {
	base := []YamlConfigAttribute{
		{YangName: "source-interface-name", TfName: "name", Type: "String", Id: true},
	}
	override := []YamlConfigAttribute{
		{YangName: "source-interface-name", Legacy: true},
	}

	got := mergeAttributes(base, override, "25.4")

	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1 (legacy keeps the attr)", len(got))
	}
	if got[0].RemovedInVersion != "25.4" {
		t.Errorf("RemovedInVersion: got %q, want %q", got[0].RemovedInVersion, "25.4")
	}
	if !got[0].Legacy {
		t.Error("Legacy: got false, want true")
	}
	// TfName should be unchanged when override doesn't specify one
	if got[0].TfName != "name" {
		t.Errorf("TfName: got %q, want %q (unchanged)", got[0].TfName, "name")
	}
}

func TestMergeAttributes_LegacyAttrRenamesTfName(t *testing.T) {
	// F7 fix: legacy block with explicit tf_name renames the base attribute
	// so the natural name is freed for a replacement attribute.
	base := []YamlConfigAttribute{
		{YangName: "console", TfName: "console", Type: "String"},
	}
	override := []YamlConfigAttribute{
		{YangName: "console", TfName: "console_legacy", Legacy: true},
	}

	got := mergeAttributes(base, override, "25.4")

	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	if got[0].TfName != "console_legacy" {
		t.Errorf("TfName: got %q, want %q", got[0].TfName, "console_legacy")
	}
	if got[0].RemovedInVersion != "25.4" {
		t.Errorf("RemovedInVersion: got %q, want %q", got[0].RemovedInVersion, "25.4")
	}
}

func TestMergeAttributes_LegacyAttrNotInBase_IsSkipped(t *testing.T) {
	// A legacy entry in the override that has no matching base attr is a no-op.
	base := []YamlConfigAttribute{
		{YangName: "existing", TfName: "existing", Type: "String"},
	}
	override := []YamlConfigAttribute{
		{YangName: "ghost", Legacy: true},
	}

	got := mergeAttributes(base, override, "25.4")

	if len(got) != 1 {
		t.Errorf("len: got %d, want 1 (ghost attr should be skipped)", len(got))
	}
}

func TestMergeAttributes_ExistingAttrFieldsUpdated(t *testing.T) {
	base := []YamlConfigAttribute{
		{YangName: "severity", TfName: "severity", Type: "String", Description: "old desc"},
	}
	override := []YamlConfigAttribute{
		{YangName: "severity", Description: "new desc", DefaultValue: "informational"},
	}

	got := mergeAttributes(base, override, "25.4")

	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	if got[0].Description != "new desc" {
		t.Errorf("Description: got %q, want %q", got[0].Description, "new desc")
	}
	// DefaultValue introduced in an override becomes a versioned default (F17 VersionDefaults).
	// The base had no default (""), so this is a version-scoped change: stored in VersionDefaults,
	// DefaultValue is cleared to "".
	if got[0].DefaultValue != "" {
		t.Errorf("DefaultValue: got %q, want empty (versioned default stored in VersionDefaults)", got[0].DefaultValue)
	}
	if got[0].VersionDefaults["25.4"] != "informational" {
		t.Errorf("VersionDefaults[25.4]: got %q, want %q", got[0].VersionDefaults["25.4"], "informational")
	}
	// Version not stamped on existing attrs
	if got[0].AddedInVersion != "" {
		t.Errorf("AddedInVersion: got %q, want empty (existing attr)", got[0].AddedInVersion)
	}
}

func TestMergeAttributes_MatchByTfName(t *testing.T) {
	// When yang_names differ but tf_names match, merge uses tf_name match.
	base := []YamlConfigAttribute{
		{YangName: "old-yang", TfName: "shared_name", Type: "String"},
	}
	override := []YamlConfigAttribute{
		{YangName: "new-yang", TfName: "shared_name", Type: "Int64"},
	}

	got := mergeAttributes(base, override, "25.4")

	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1 (matched by tf_name)", len(got))
	}
	if got[0].Type != "Int64" {
		t.Errorf("Type: got %q, want %q", got[0].Type, "Int64")
	}
}

func TestMergeAttributes_NewAttrWithNestedGetAddedInVersion(t *testing.T) {
	// A completely new list attr in the override: its nested attrs should be
	// stamped with AddedInVersion too.
	base := []YamlConfigAttribute{}
	override := []YamlConfigAttribute{
		{
			YangName: "new-list",
			TfName:   "new_list",
			Type:     "List",
			Attributes: []YamlConfigAttribute{
				{YangName: "key", TfName: "key", Type: "String", Id: true},
				{YangName: "value", TfName: "value", Type: "String"},
			},
		},
	}

	got := mergeAttributes(base, override, "25.4")

	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	if got[0].AddedInVersion != "25.4" {
		t.Errorf("outer AddedInVersion: got %q, want %q", got[0].AddedInVersion, "25.4")
	}
	for _, child := range got[0].Attributes {
		if child.AddedInVersion != "25.4" {
			t.Errorf("child %q AddedInVersion: got %q, want %q",
				child.YangName, child.AddedInVersion, "25.4")
		}
	}
}

func TestMergeAttributes_CompositeKeyVersionedKeys(t *testing.T) {
	// Reproduces the logging source_interfaces scenario:
	// 24.4 key: source-interface-name (id:true)
	// 25.4: retire source-interface-name, add interface-name + vrf-name (both id:true)
	base := []YamlConfigAttribute{
		{YangName: "source-interface-name", TfName: "name", Type: "String", Id: true},
	}
	override := []YamlConfigAttribute{
		{YangName: "source-interface-name", Legacy: true},
		{YangName: "interface-name", TfName: "interface_name", Type: "String", Id: true},
		{YangName: "vrf-name", TfName: "vrf_name", Type: "String", Id: true},
	}

	got := mergeAttributes(base, override, "25.4")

	// Expect 3 attrs: retired name, new interface_name, new vrf_name
	if len(got) != 3 {
		t.Fatalf("len: got %d, want 3", len(got))
	}

	// Index 0: source-interface-name retired
	if got[0].RemovedInVersion != "25.4" {
		t.Errorf("name RemovedInVersion: got %q, want %q", got[0].RemovedInVersion, "25.4")
	}
	if !got[0].Id {
		t.Error("name Id: got false, want true (should retain Id)")
	}

	// Index 1: interface-name added
	if got[1].YangName != "interface-name" {
		t.Errorf("got[1] YangName: got %q, want %q", got[1].YangName, "interface-name")
	}
	if got[1].AddedInVersion != "25.4" {
		t.Errorf("interface-name AddedInVersion: got %q, want %q", got[1].AddedInVersion, "25.4")
	}
	if !got[1].Id {
		t.Error("interface-name Id: got false, want true")
	}

	// Index 2: vrf-name added
	if got[2].YangName != "vrf-name" {
		t.Errorf("got[2] YangName: got %q, want %q", got[2].YangName, "vrf-name")
	}
	if got[2].AddedInVersion != "25.4" {
		t.Errorf("vrf-name AddedInVersion: got %q, want %q", got[2].AddedInVersion, "25.4")
	}
}

func TestMergeAttributes_ReplacesYangName_OnKeyAttr(t *testing.T) {
	// Verifies that replaces_yang_name on an id:true attribute is fully processed:
	// Phase 1 (mergeAttributes): VersionYangNames populated with "_base" placeholder.
	// Phase 2 (fixAttributeBaseVersion): "_base" replaced with real base version,
	// MovedInVersion derived.
	base := []YamlConfigAttribute{
		{YangName: "name", TfName: "name", Type: "String", Id: true},
	}
	override := []YamlConfigAttribute{
		{YangName: "host", TfName: "name", Type: "String", Id: true, ReplacesYangName: "name"},
	}

	// Phase 1
	got := mergeAttributes(base, override, "25.4")
	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	if got[0].YangName != "host" {
		t.Errorf("YangName: got %q, want %q", got[0].YangName, "host")
	}
	if !got[0].Id {
		t.Error("Id: got false, want true")
	}
	if got[0].TfName != "name" {
		t.Errorf("TfName: got %q, want %q", got[0].TfName, "name")
	}
	if got[0].AddedInVersion != "" {
		t.Errorf("AddedInVersion: got %q, want empty (not treated as new attr)", got[0].AddedInVersion)
	}
	if got[0].RemovedInVersion != "" {
		t.Errorf("RemovedInVersion: got %q, want empty", got[0].RemovedInVersion)
	}
	if got[0].VersionYangNames["_base"] != "name" {
		t.Errorf("VersionYangNames[_base]: got %q, want %q", got[0].VersionYangNames["_base"], "name")
	}
	if got[0].VersionYangNames["25.4"] != "host" {
		t.Errorf("VersionYangNames[25.4]: got %q, want %q", got[0].VersionYangNames["25.4"], "host")
	}
	if got[0].MovedInVersion != "" {
		t.Errorf("MovedInVersion: got %q, want empty (not set until fixAttributeBaseVersion)", got[0].MovedInVersion)
	}

	// Phase 2
	fixAttributeBaseVersion(&got[0], "24.4")
	if got[0].VersionYangNames["24.4"] != "name" {
		t.Errorf("VersionYangNames[24.4]: got %q, want %q", got[0].VersionYangNames["24.4"], "name")
	}
	if _, hasBase := got[0].VersionYangNames["_base"]; hasBase {
		t.Error("VersionYangNames[_base]: still present after fixAttributeBaseVersion, want removed")
	}
	if got[0].MovedInVersion != "25.4" {
		t.Errorf("MovedInVersion: got %q, want %q", got[0].MovedInVersion, "25.4")
	}
}

// ---------------------------------------------------------------------------
// TODO (F4a): Add GetPathVersion tests in internal/provider/helpers/version_path_test.go
// when helpers.GetPathVersion is implemented. Test cases to cover:
//
//  - empty version string → returns defaultPath (base fallback)
//  - version below all thresholds → returns defaultPath
//  - exact match on lowest threshold → returns that path
//  - version between two thresholds → returns the lower threshold's path
//  - exact match on upper threshold → returns upper path
//  - version above all thresholds → returns highest threshold's path
//  - patch version component ignored ("24.4.2" == "24.4")
//  - empty pathByVersion map → returns defaultPath
// ---------------------------------------------------------------------------
