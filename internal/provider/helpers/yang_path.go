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

// SelectYangPath returns the correct gNMI JSON path for the given device version.
// newPath is used for versions >= movedInVersion; oldPath for older versions.
// When version is empty (unknown), oldPath is returned as the safe fallback —
// VersionAtLeast("", x) returns true, so we guard against that explicitly.
func SelectYangPath(version, newPath, oldPath, movedInVersion string) string {
	if version != "" && VersionAtLeast(version, movedInVersion) {
		return newPath
	}
	return oldPath
}
