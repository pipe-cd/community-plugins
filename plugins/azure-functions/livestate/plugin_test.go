// Copyright 2025 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package livestate

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/stretchr/testify/assert"
)

// renderWhatIfChange used to take its *strings.Builder argument by value. strings.Builder
// is documented as unsafe to copy once it has been written to, and its real (and only)
// caller, getFunctionSyncState, always writes to its Builder before calling
// renderWhatIfChange, so the by-value copy panicked on every non-empty diff. This test
// reproduces that exact calling pattern to guard against a regression.
func TestRenderWhatIfChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		change   armresources.WhatIfPropertyChange
		expected string
	}{
		{
			name: "create",
			change: armresources.WhatIfPropertyChange{
				Path:               to.Ptr("properties.siteConfig.alwaysOn"),
				PropertyChangeType: to.Ptr(armresources.PropertyChangeTypeCreate),
				After:              "true",
			},
			expected: "+ path:properties.siteConfig.alwaysOn value:true\n\n",
		},
		{
			name: "modify",
			change: armresources.WhatIfPropertyChange{
				Path:               to.Ptr("properties.siteConfig.linuxFxVersion"),
				PropertyChangeType: to.Ptr(armresources.PropertyChangeTypeModify),
				Before:             "node|18",
				After:              "node|20",
			},
			expected: "# path:properties.siteConfig.linuxFxVersion value:node|18 -> node|20\n\n",
		},
		{
			name: "delete",
			change: armresources.WhatIfPropertyChange{
				Path:               to.Ptr("tags.env"),
				PropertyChangeType: to.Ptr(armresources.PropertyChangeTypeDelete),
				Before:             "staging",
			},
			expected: "- path:tags.env value:staging\n\n",
		},
		{
			name: "array with nested children",
			change: armresources.WhatIfPropertyChange{
				Path:               to.Ptr("properties.siteConfig.appSettings"),
				PropertyChangeType: to.Ptr(armresources.PropertyChangeTypeArray),
				Children: []*armresources.WhatIfPropertyChange{
					{
						Path:               to.Ptr("properties.siteConfig.appSettings[0]"),
						PropertyChangeType: to.Ptr(armresources.PropertyChangeTypeCreate),
						After:              "WEBSITE_RUN_FROM_PACKAGE",
					},
				},
			},
			expected: "# path:properties.siteConfig.appSettings\n\n+ path:properties.siteConfig.appSettings[0] value:WEBSITE_RUN_FROM_PACKAGE\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Reproduce the real caller's usage: the Builder already has content
			// written to it before renderWhatIfChange is called. Under the old
			// by-value signature, this alone was enough to panic.
			var b strings.Builder
			b.WriteString("Diff between the defined state in Git and actual live state:\n\n")

			assert.NotPanics(t, func() {
				renderWhatIfChange(&b, tt.change)
			})

			assert.Contains(t, b.String(), tt.expected)
		})
	}
}
