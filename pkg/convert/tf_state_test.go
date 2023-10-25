// Copyright 2016-2023, Pulumi Corporation.
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

package convert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	bridgetesting "github.com/pulumi/pulumi-converter-terraform/pkg/testing"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tf2pulumi/il"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTranslateState runs through all the folders in testdata and tries to convert their tfstate.json file to
// a pulumi import response.
func TestTranslateState(t *testing.T) {
	// Test framework for TranslateState
	// Each folder in testdata has a tfstate.json file and a .json file with the expected output
	testDir, err := filepath.Abs(filepath.Join("testdata"))
	require.NoError(t, err)
	infos, err := os.ReadDir(filepath.Join(testDir, "states"))
	require.NoError(t, err)

	tests := make([]struct {
		name string
		path string
	}, 0)
	for _, info := range infos {
		if info.IsDir() {
			tests = append(tests, struct {
				name string
				path string
			}{
				name: info.Name(),
				path: filepath.Join(testDir, "states", info.Name()),
			})
		}
	}

	mapper := &bridgetesting.TestFileMapper{Path: filepath.Join(testDir, "mappings")}
	info := il.NewCachingProviderInfoSource(il.NewMapperProviderInfoSource(mapper))

	for _, tt := range tests {
		tt := tt // avoid capturing loop variable in the closure

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			statePath := filepath.Join(tt.path, "tfstate.json")

			actualImport, err := TranslateState(info, statePath)
			require.NoError(t, err)
			diagnostics := []string{}
			if len(actualImport.Diagnostics) > 0 {
				t.Logf("diagnostics: %v", actualImport.Diagnostics)
			}
			for _, diagnostic := range actualImport.Diagnostics {
				sev := "error"
				if diagnostic.Severity == hcl.DiagWarning {
					sev = "warning"
				}
				assert.True(t,
					diagnostic.Severity == hcl.DiagError || diagnostic.Severity == hcl.DiagWarning,
					"diagnostic should be an error or warning")

				assert.Nil(t, diagnostic.Subject, "diagnostic should not have a subject")
				assert.Nil(t, diagnostic.Context, "diagnostic should not have a context")

				diagnostics = append(diagnostics,
					fmt.Sprintf("%s:%s:%s", sev, diagnostic.Summary, diagnostic.Detail))
			}

			// Check the diagnostics match what we expect
			expectedDiagnosticsPath := filepath.Join(tt.path, "diagnostics.json")
			bridgetesting.AssertEqualsJSONFile(t, expectedDiagnosticsPath, diagnostics, &[]string{})

			// Check the set of imported resources is what we expect. We can't use AssertEqualsJSONFile here
			// because this looks like a list but we actually need to treat it as a set.
			if cmdutil.IsTruthy(os.Getenv("PULUMI_ACCEPT")) {
				// If PULUMI_ACCEPT is set then write the expected file
				actualImportBytes, err := json.MarshalIndent(actualImport.Resources, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(tt.path, "import.json"), actualImportBytes, 0o600)
				require.NoError(t, err)
			}

			expectedImportBytes, err := os.ReadFile(filepath.Join(tt.path, "import.json"))
			require.NoError(t, err)
			var expectedImport []plugin.ResourceImport
			err = json.Unmarshal(expectedImportBytes, &expectedImport)
			require.NoError(t, err)

			// We don't actually care about order here, just that the "set" of resources is the same.
			assert.ElementsMatch(t, expectedImport, actualImport.Resources)
		})
	}
}
