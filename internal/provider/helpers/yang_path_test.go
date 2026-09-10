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

package helpers

import "testing"

func TestSelectYangPath(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		newPath        string
		oldPath        string
		movedInVersion string
		want           string
	}{
		{"exact match on moved version returns new path", "25.4", "new/path", "old/path", "25.4", "new/path"},
		{"version below threshold returns old path", "24.4", "new/path", "old/path", "25.4", "old/path"},
		{"version above threshold returns new path", "26.1", "new/path", "old/path", "25.4", "new/path"},
		{"empty version returns old path (safe fallback)", "", "new/path", "old/path", "25.4", "old/path"},
		{"patch component ignored: 24.4.2 treated as 24.4", "24.4.2", "new/path", "old/path", "25.4", "old/path"},
		{"patch component ignored: 25.4.2 treated as 25.4", "25.4.2", "new/path", "old/path", "25.4", "new/path"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SelectYangPath(tc.version, tc.newPath, tc.oldPath, tc.movedInVersion)
			if got != tc.want {
				t.Errorf("SelectYangPath(%q, %q, %q, %q) = %q, want %q",
					tc.version, tc.newPath, tc.oldPath, tc.movedInVersion, got, tc.want)
			}
		})
	}
}
