// Copyright 2026, Pulumi Corporation.
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

package conformance

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/blang/semver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pulumi/pulumi-converter-terraform/pkg/convert"
	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/pulexec"
	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/tfexec"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfgen"
	pschema "github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/diag"
	"github.com/pulumi/pulumi/sdk/v3/go/common/diag/colors"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	"github.com/pulumi/terraform/pkg/configs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Provider pairs a terraform provider name with a factory that creates it.
type Provider struct {
	Name    string
	Factory func() *schema.Provider
}

// TestCase defines a test that asserts the converter produces the same outputs as
// running the HCL program directly against the Terraform provider.
type TestCase struct {
	Providers []Provider
	Config    map[string]string

	// Input maps file paths (relative to the project root) to file contents.
	// For a single-file program, use {"main.tf": "..."}.
	// For multi-file programs or modules, include all files:
	//   {"main.tf": "...", "mod/main.tf": "..."}
	Input map[string]string

	// AssertState is an optional callback that receives the Pulumi deployment
	// resources after `pulumi up`. Use it to assert on resource options like
	// Dependencies, IgnoreChanges, etc. that are not visible in stack outputs.
	AssertState func(t *testing.T, resources []apitype.ResourceV3)
}

// AssertConversion runs the HCL program through two paths and asserts identical outputs:
//
// Path A: TF Provider + HCL → terraform apply → outputs
// Path B: TF Provider + HCL → convert to PCL → bridge provider → pulumi up → outputs
//
// The generated PCL is also compared against a golden file at
// testdata/<TestName>/main.pp. Set PULUMI_ACCEPT=1 to update the golden files.
func AssertConversion(t *testing.T, tc TestCase) {
	t.Helper()

	_, callerFile, _, _ := runtime.Caller(1)
	testdataDir := filepath.Join(filepath.Dir(callerFile), "testdata")

	// Build TF providers and bridged providers from the test case.
	tfProviders := make([]tfexec.Provider, len(tc.Providers))
	bridgedProviders := make([]pulexec.Provider, len(tc.Providers))
	providerInfos := make(map[string]*tfbridge.ProviderInfo, len(tc.Providers))
	for i, p := range tc.Providers {
		tfProviders[i] = tfexec.Provider{Name: p.Name, Provider: p.Factory()}
		bridged := pulexec.BridgedProvider(t, p.Name, p.Factory())
		bridgedProviders[i] = pulexec.Provider{Name: p.Name, Info: bridged}
		providerInfos[p.Name] = &bridged
	}

	var tfOutputs map[string]string
	var pulumiResult pulexec.Result
	var wg sync.WaitGroup
	wg.Add(2)

	// Path A: run HCL directly via Terraform.
	go func() {
		defer wg.Done()
		driver := tfexec.NewDriver(t, tfProviders)
		tfOutputs = driver.Apply(t, tc.Input, tc.Config)
	}()

	// Path B: convert HCL → PCL, bridge the provider, run via Pulumi.
	var pclFiles map[string]string
	go func() {
		defer wg.Done()
		pclDir, err := convertHCLToPCL(t, tc.Input, providerInfos)
		if !assert.NoError(t, err, "convertHCLToPCL") {
			return
		}
		pclFiles, err = readPPFiles(pclDir)
		if !assert.NoError(t, err) {
			return
		}
		pulumiResult = pulexec.Run(t, bridgedProviders, pclFiles, tc.Config)
	}()

	wg.Wait()

	if t.Failed() {
		return
	}

	assertGoldenPCL(t, testdataDir, pclFiles)

	// The converter camelCases TF output names (e.g. "computed_value" → "computedValue").
	// Normalize TF output keys to camelCase so we compare values, not naming conventions.
	normalizedTF := make(map[string]string, len(tfOutputs))
	for k, v := range tfOutputs {
		normalizedTF[convert.CamelCaseName(k)] = v
	}
	require.Equal(t, normalizedTF, pulumiResult.Outputs)

	if tc.AssertState != nil {
		tc.AssertState(t, pulumiResult.Resources)
	}
}

// convertHCLToPCL writes the input files to a temp directory and runs TranslateModule to produce PCL.
func convertHCLToPCL(
	t *testing.T, input map[string]string, providerInfos map[string]*tfbridge.ProviderInfo,
) (string, error) {
	t.Helper()

	srcDir := t.TempDir()
	for path, content := range input {
		fullPath := filepath.Join(srcDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			return "", err
		}
	}

	dstDir := t.TempDir()
	osFs := afero.NewOsFs()
	dstFs := afero.NewBasePathFs(osFs, dstDir)

	infoSource := &testProviderInfoSource{providers: providerInfos}
	resolver := &testProviderInfoResolver{}
	loader := newTestLoader(t, providerInfos)

	diags := convert.TranslateModule(osFs, srcDir, dstFs, infoSource, resolver, dstDir, loader)
	require.False(t, diags.HasErrors(), "TranslateModule failed: %v", diags)

	return dstDir, nil
}

// testLoader implements pschema.ReferenceLoader backed by in-memory package references
// generated from bridged provider infos.
type testLoader struct {
	packages map[string]pschema.PackageReference
}

func newTestLoader(t *testing.T, providerInfos map[string]*tfbridge.ProviderInfo) *testLoader {
	t.Helper()

	sink := diag.DefaultSink(io.Discard, io.Discard, diag.FormatOptions{Color: colors.Never})
	packages := make(map[string]pschema.PackageReference, len(providerInfos))
	for _, info := range providerInfos {
		spec, err := tfgen.GenerateSchema(*info, sink)
		require.NoError(t, err)

		pkg, err := pschema.ImportSpec(spec, nil, pschema.ValidationOptions{})
		require.NoError(t, err)

		packages[pkg.Name] = pkg.Reference()
	}
	return &testLoader{packages: packages}
}

func (l *testLoader) LoadPackage(pkg string, version *semver.Version) (*pschema.Package, error) {
	ref, err := l.LoadPackageReference(pkg, version)
	if err != nil {
		return nil, err
	}
	return ref.Definition()
}

func (l *testLoader) LoadPackageReference(pkg string, _ *semver.Version) (pschema.PackageReference, error) {
	ref, ok := l.packages[pkg]
	if !ok {
		return nil, fmt.Errorf("unknown package %q", pkg)
	}
	return ref, nil
}

func (l *testLoader) LoadPackageV2(
	_ context.Context, descriptor *pschema.PackageDescriptor,
) (*pschema.Package, error) {
	ref, err := l.LoadPackageReference(descriptor.Name, descriptor.Version)
	if err != nil {
		return nil, err
	}
	return ref.Definition()
}

func (l *testLoader) LoadPackageReferenceV2(
	_ context.Context, descriptor *pschema.PackageDescriptor,
) (pschema.PackageReference, error) {
	return l.LoadPackageReference(descriptor.Name, descriptor.Version)
}

// testProviderInfoSource returns provider info from an in-memory map.
type testProviderInfoSource struct {
	providers map[string]*tfbridge.ProviderInfo
}

func (s *testProviderInfoSource) GetProviderInfo(
	tfProvider string, _ *configs.RequiredProvider,
) (*tfbridge.ProviderInfo, error) {
	info, ok := s.providers[tfProvider]
	if !ok {
		return nil, nil
	}
	return info, nil
}

// testProviderInfoResolver returns nil for all providers, skipping registry resolution.
type testProviderInfoResolver struct{}

func (r *testProviderInfoResolver) ResolveLatest(string) (*configs.RequiredProvider, error) {
	return nil, nil
}

// readPPFiles walks a directory and returns all .pp files as a map of relative paths to contents.
func readPPFiles(dir string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".pp" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path) //nolint:gosec // dir is a trusted temp directory we created
		if err != nil {
			return err
		}
		files[rel] = string(content)
		return nil
	})
	return files, err
}

// assertGoldenPCL compares all generated PCL files against golden files.
// When PULUMI_ACCEPT=1, the golden files are updated instead.
func assertGoldenPCL(t *testing.T, testdataDir string, pclFiles map[string]string) {
	t.Helper()

	goldenDir := filepath.Join(testdataDir, t.Name())

	if cmdutil.IsTruthy(os.Getenv("PULUMI_ACCEPT")) {
		for path, content := range pclFiles {
			goldenPath := filepath.Join(goldenDir, path)
			err := os.MkdirAll(filepath.Dir(goldenPath), 0o755)
			require.NoError(t, err)
			err = os.WriteFile(goldenPath, []byte(content), 0o600)
			require.NoError(t, err)
		}
		return
	}

	for path, actual := range pclFiles {
		goldenPath := filepath.Join(goldenDir, path)
		expected, err := os.ReadFile(goldenPath)
		require.NoError(t, err, "golden file %s not found; run with PULUMI_ACCEPT=1 to create it", goldenPath)
		assert.Equal(t, string(expected), actual, "mismatch in %s", path)
	}
}
