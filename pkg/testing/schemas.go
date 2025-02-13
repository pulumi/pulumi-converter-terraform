// Copyright 2016-2022, Pulumi Corporation.
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

package testing

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// If zeroValue is non-nil it's used if a JSON file is missing. This saves us having to keep a load of diagnostic.json
// files with just "[]" in them.
func AssertEqualsJSONFile[T any](
	t *testing.T,
	expectedJSONFile string,
	actualData T,
	zeroValue *T,
) {
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	err := enc.Encode(actualData)
	require.NoError(t, err)

	if os.Getenv("PULUMI_ACCEPT") != "" {
		// Don't write if its equal to the zero value.
		if zeroValue != nil && assert.ObjectsAreEqual(*zeroValue, actualData) {
			return
		}

		err := os.WriteFile(expectedJSONFile, buf.Bytes(), 0o600)
		require.NoError(t, err)
	}

	expectedData, err := os.ReadFile(expectedJSONFile)
	if os.IsNotExist(err) {
		if zeroValue != nil && assert.ObjectsAreEqual(*zeroValue, actualData) {
			return
		}
	}
	require.NoError(t, err)

	assert.Equal(t, string(expectedData), buf.String())
}
