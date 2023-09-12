// Copyright 2016-2023, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/stretchr/testify/require"
)

type testMapper struct{}

func (m *testMapper) GetMapping(context.Context, string, string) ([]byte, error) {
	// No mapping as yet, we'll get warning diagnostics about this but that's not important for this test.
	return nil, nil
}

func TestExamplesJson(t *testing.T) {
	t.Parallel()

	uut := &tfConverter{}

	ctx := context.Background()
	src := t.TempDir()
	dst := t.TempDir()

	// Create a mock mapper server.
	mapper := &testMapper{}
	// It's ok to pass a zero plugin.Context to NewServer
	grpcServer, err := plugin.NewServer(
		&plugin.Context{},
		convert.MapperRegistration(convert.NewMapperServer(mapper)))
	require.NoError(t, err)

	// Write an examples.json file to the source directory.
	examplesJSON := `{
		"empty": "",
		"aws": "resource aws_bucket foo { }"
	}`
	err = os.WriteFile(filepath.Join(src, "examples.json"), []byte(examplesJSON), 0o600)
	require.NoError(t, err)

	resp, err := uut.ConvertProgram(ctx, &plugin.ConvertProgramRequest{
		SourceDirectory: src,
		TargetDirectory: dst,
		MapperTarget:    grpcServer.Addr(),
		LoaderTarget:    "", // unused by the converter
	})
	require.NoError(t, err)
	// Check that response didn't return any diagnostics
	require.Empty(t, resp.Diagnostics)

	// Ensure an examples.json file was written to the target directory.
	resultJSONBytes, err := os.ReadFile(filepath.Join(dst, "examples.json"))
	require.NoError(t, err)
	var resultJSON map[string]interface{}
	err = json.Unmarshal(resultJSONBytes, &resultJSON)
	require.NoError(t, err)

	expectedJSON := map[string]interface{}{
		"empty": map[string]interface{}{
			"pcl":         "",
			"diagnostics": []map[string]interface{}{},
		},
		"aws": map[string]interface{}{
			"pcl": "resource \"foo\" \"aws:index:bucket\" {}\n",
			"diagnostics": []map[string]interface{}{
				{
					"Severity": 2,
					"Summary":  "Failed to get provider info",
					"Detail":   "Failed to get provider info for \"aws_bucket\": could not find mapping information for provider aws; try installing a pulumi plugin that supports this terraform provider",
					"Subject": map[string]interface{}{
						"Filename": "/aws.tf",
						"Start":    map[string]interface{}{"Line": 1, "Column": 1, "Byte": 0},
						"End":      map[string]interface{}{"Line": 1, "Column": 24, "Byte": 23},
					},
					"Context":     nil,
					"Expression":  nil,
					"EvalContext": nil,
					"Extra":       nil,
				},
			},
		},
	}
	require.Equal(t, expectedJSON, resultJSON)
}
