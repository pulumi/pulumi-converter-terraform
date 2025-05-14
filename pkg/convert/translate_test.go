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

package convert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/spf13/afero"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfgen"
	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
	"github.com/pulumi/pulumi/pkg/v3/codegen/hcl2/syntax"
	"github.com/pulumi/pulumi/pkg/v3/codegen/pcl"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/diag"
	"github.com/pulumi/pulumi/sdk/v3/go/common/diag/colors"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bridgetesting "github.com/pulumi/pulumi-converter-terraform/pkg/testing"
)

type testLoader struct {
	path string
}

func (l *testLoader) LoadPackage(pkg string, version *semver.Version) (*schema.Package, error) {
	schemaPath := pkg
	if version != nil {
		schemaPath += "-" + version.String()
	}
	schemaPath = filepath.Join(l.path, schemaPath) + ".json"

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	var spec schema.PackageSpec
	err = json.Unmarshal(schemaBytes, &spec)
	if err != nil {
		return nil, err
	}

	schemaPackage, diags, err := schema.BindSpec(spec, l, schema.ValidationOptions{})
	if err != nil {
		return nil, err
	}
	if diags.HasErrors() {
		return nil, diags
	}

	return schemaPackage, nil
}

func (l *testLoader) LoadPackageV2(ctx context.Context, descriptor *schema.PackageDescriptor) (*schema.Package, error) {
	if descriptor.Parameterization != nil {
		packageName := descriptor.Parameterization.Name
		return l.LoadPackage(packageName, &descriptor.Parameterization.Version)
	}
	return l.LoadPackage(descriptor.Name, descriptor.Version)
}

func (l *testLoader) LoadPackageReference(pkg string, version *semver.Version) (schema.PackageReference, error) {
	schemaPackage, err := l.LoadPackage(pkg, version)
	if err != nil {
		return nil, err
	}
	return schemaPackage.Reference(), nil
}

// TestTranslate runs through all the folders in testdata (except for "schemas" and "mappings") and tries to
// convert all the .tf files in that folder into PCL.
//
// It will use schemas from the testdata/schemas folder, and mappings from the testdata/mappings folder. The
// resulting PCL will be checked against PCL written to a subfolder inside each test folder called "pcl".
func TestTranslate(t *testing.T) {
	t.Parallel()

	// Test framework for eject
	// Each folder in testdata has a pcl folder, we check that if we convert the hcl we get the expected pcl
	// You can regenerate the test data by running "PULUMI_ACCEPT=1 go test" in this folder (pkg/convert).
	testDir, err := filepath.Abs(filepath.Join("testdata"))
	require.NoError(t, err)
	infos, err := os.ReadDir(filepath.Join(testDir, "programs"))
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
				path: filepath.Join(testDir, "programs", info.Name()),
			})
		}
	}

	loader := &testLoader{path: filepath.Join(testDir, "schemas")}
	mapper := &bridgetesting.TestFileMapper{Path: filepath.Join(testDir, "mappings")}

	for _, tt := range tests {
		tt := tt // avoid capturing loop variable in the closure

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			snapshotPath := filepath.Join(tt.path, "pcl")
			if cmdutil.IsTruthy(os.Getenv("PULUMI_ACCEPT")) {
				err := os.RemoveAll(snapshotPath)
				require.NoError(t, err, "failed to remove existing files at %s", snapshotPath)
				err = os.MkdirAll(snapshotPath, 0o700)
				require.NoError(t, err, "failed to create directory at %s", snapshotPath)
			}

			// Copy the .tf files to a new directory
			tempDir := t.TempDir()
			pclPath := filepath.Join(tempDir, "pcl")
			hclPath := filepath.Join(tempDir, tt.name)
			modulePath := filepath.Join(tempDir, "modules")

			copyFiles := func(srcDirectory, dstDirectory, suffix string) {
				err = filepath.WalkDir(srcDirectory, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}

					if !d.IsDir() && (strings.HasSuffix(d.Name(), suffix) || suffix == "") {
						src, err := os.Open(path)
						if err != nil {
							return fmt.Errorf("open src: %w", err)
						}
						defer src.Close()

						relativePath, err := filepath.Rel(srcDirectory, path)
						if err != nil {
							return err
						}

						dstPath := filepath.Join(dstDirectory, relativePath)
						dstDir := filepath.Dir(dstPath)
						err = os.MkdirAll(dstDir, 0o700)
						if err != nil {
							return fmt.Errorf("create dst dir: %w", err)
						}

						dst, err := os.Create(dstPath)
						if err != nil {
							return fmt.Errorf("open dst: %w", err)
						}
						defer dst.Close()

						_, err = io.Copy(dst, src)
						if err != nil {
							return fmt.Errorf("copy: %w", err)
						}
					}

					return nil
				})
				require.NoError(t, err)
			}

			copyFiles(tt.path, hclPath, ".tf")
			copyFiles(filepath.Join(testDir, "modules"), modulePath, ".tf")

			osFs := afero.NewOsFs()
			pclFs := afero.NewBasePathFs(osFs, pclPath)

			providerInfoSource := NewMapperProviderInfoSource(mapper)
			diagnostics := TranslateModule(osFs, hclPath, pclFs, providerInfoSource, pclPath)

			// If PULUMI_ACCEPT is set then clear the PCL folder and copy the generated files out. Note we
			// copy these out even if this returned errors, this makes it easy in the local dev loop to see
			// what's wrong without going and looking in temp directories.
			if cmdutil.IsTruthy(os.Getenv("PULUMI_ACCEPT")) {
				copyFiles(pclPath, snapshotPath, "")
			}

			require.False(t,
				diagnostics.HasErrors(),
				"translate diagnostics should not have errors: %v",
				diagnostics)

			// Keep track of all diagnostics, from each of the three phases. We'll check these match what we expect at the end
			// of the test.
			allDiagnostics := []string{}
			logDiagnostics := func(label string, diagnostics hcl.Diagnostics) {
				if len(diagnostics) > 0 {
					t.Logf("%s diagnostics: %v", label, diagnostics)
				}
				for _, diagnostic := range diagnostics {
					sev := "error"
					if diagnostic.Severity == hcl.DiagWarning {
						sev = "warning"
					}
					assert.True(t,
						diagnostic.Severity == hcl.DiagError || diagnostic.Severity == hcl.DiagWarning,
						"diagnostic should be an error or warning")

					assert.NotNil(t, diagnostic.Subject, "diagnostic should have a subject")

					// We need to ensure the ranges just print relative paths here because the absolute paths
					// change on each test run. We're ok to mutate the struct here because we aren't going to
					// be using these diagnostic objects again after this.
					rangeToString := func(r *hcl.Range) string {
						// only rewrite the filename if it's in the tempDir
						if strings.HasPrefix(r.Filename, tempDir) {
							path, err := filepath.Rel(tempDir, r.Filename)
							if err == nil {
								r.Filename = path
							}
						}
						return r.String()
					}

					if diagnostic.Context == nil {
						allDiagnostics = append(allDiagnostics,
							fmt.Sprintf("%s:%s:%s:%s", sev, rangeToString(diagnostic.Subject), diagnostic.Summary, diagnostic.Detail))
					} else {
						allDiagnostics = append(allDiagnostics,
							fmt.Sprintf(
								"%s:%s:%s:%s:%s",
								sev,
								rangeToString(diagnostic.Context),
								rangeToString(diagnostic.Subject),
								diagnostic.Summary,
								diagnostic.Detail,
							),
						)
					}
				}
			}
			logDiagnostics("translate", diagnostics)

			// If this is a partial test turn on the options to allow missing bits
			partial := strings.HasPrefix(tt.name, "partial_")
			pulumiOptions := []pcl.BindOption{
				pcl.Loader(loader),
				pcl.DirPath("/"),
				pcl.ComponentBinder(componentProgramBinderFromAfero(pclFs)),
			}

			if partial {
				pulumiOptions = append(pulumiOptions, pcl.AllowMissingVariables)
				pulumiOptions = append(pulumiOptions, pcl.AllowMissingProperties)
				pulumiOptions = append(pulumiOptions, pcl.SkipResourceTypechecking)
			}

			// TODO: We should probably make a CLI command to do this, but for now just use the codegen
			// library to check the program is valid.
			pulumiParser := syntax.NewParser()
			files, err := os.ReadDir(pclPath)
			require.NoError(t, err)
			for _, file := range files {
				// These are all in the root folder, and Open needs the full filename.
				fileName := filepath.Join(pclPath, file.Name())
				if filepath.Ext(fileName) == ".pp" {
					reader, err := os.Open(fileName)
					require.NoError(t, err)

					err = pulumiParser.ParseFile(reader, filepath.Base(fileName))
					require.NoError(t, err)

					require.False(t,
						pulumiParser.Diagnostics.HasErrors(),
						"parse diagnostics should not have errors: %v",
						pulumiParser.Diagnostics)
				}
			}
			logDiagnostics("parser", pulumiParser.Diagnostics)

			_, diagnostics, err = pcl.BindProgram(pulumiParser.Files, pulumiOptions...)
			require.NoError(t, err)
			require.False(t,
				diagnostics.HasErrors(),
				"bind diagnostics should not have errors: %v",
				diagnostics)
			logDiagnostics("bind", diagnostics)

			// We have all the diagnostics now check they match what we expect.
			expectedDiagnosticsPath := filepath.Join(tt.path, "pcl", "diagnostics.json")
			bridgetesting.AssertEqualsJSONFile(t, expectedDiagnosticsPath, allDiagnostics, &[]string{})

			// Assert every pcl file is seen
			_, err = os.ReadDir(snapshotPath)
			if !os.IsNotExist(err) && !assert.NoError(t, err) {
				// If the directory was not found then the expected pcl results are the empty set, but if the
				// directory could not be read because of filesystem issues than just error out.
				assert.FailNow(t, "Could not read expected pcl results")
			}

			// compare the generated files with files on disk
			snapshotFs := afero.NewBasePathFs(osFs, snapshotPath)
			err = afero.Walk(pclFs, "/", func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info == nil || info.IsDir() {
					// ignore directories, just recuse down
					return nil
				}

				sourceOnDisk, err := afero.ReadFile(snapshotFs, path)
				assert.NoError(t, err, "generated source file must be on disk")
				sourceInMemory, err := afero.ReadFile(pclFs, path)
				assert.NoError(t, err, "should be able to read %s", path)
				expectedPcl := strings.Replace(string(sourceOnDisk), "\r\n", "\n", -1)
				actualPcl := strings.Replace(string(sourceInMemory), "\r\n", "\n", -1)
				assert.Equal(t, expectedPcl, actualPcl)
				return nil
			})
			require.NoError(t, err, "failed to check written files")

			// make sure _all_ files on disk are also generated in the source
			err = afero.Walk(snapshotFs, "/", func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info == nil || info.IsDir() || filepath.Ext(info.Name()) != ".pp" {
					// ignore directories and non-PCL files
					return nil
				}

				_, err = afero.ReadFile(pclFs, path)
				assert.NoError(t, err, "file on disk was not generated in memory: %s", path)
				return nil
			})
			// It's ok for the pcl directory to just not exist, this happens for the empty tests.
			if !errors.Is(err, fs.ErrNotExist) {
				require.NoError(t, err, "failed to check saved files")
			}
		})
	}
}

func Test_GenerateTestDataSchemas(t *testing.T) {
	t.Parallel()

	// This is to assert that all the schemas we save in testdata/schemas, match up with the
	// mapping files in testdata/mappings. Add in the use of PULUMI_ACCEPT and it means you
	// don't have to manually write schemas, just mappings for tests.

	testDir, err := filepath.Abs("testdata")
	require.NoError(t, err)
	mappingsPath := filepath.Join(testDir, "mappings")
	schemasPath := filepath.Join(testDir, "schemas")
	mapper := &bridgetesting.TestFileMapper{Path: mappingsPath}
	providerInfoSource := NewMapperProviderInfoSource(mapper)

	nilSink := diag.DefaultSink(io.Discard, io.Discard, diag.FormatOptions{
		Color: colors.Never,
	})

	// Generate the schemas from the mappings
	infos, err := os.ReadDir(mappingsPath)
	require.NoError(t, err)
	for _, info := range infos {
		info := info

		t.Run(info.Name(), func(t *testing.T) {
			t.Parallel()

			// Strip off the .json part to make the package name
			pkg := strings.Replace(info.Name(), filepath.Ext(info.Name()), "", -1)
			provInfo, err := providerInfoSource.GetProviderInfo(pkg, nil /*requiredProvider*/)
			require.NoError(t, err)

			schema, err := tfgen.GenerateSchema(*provInfo, nilSink)
			require.NoError(t, err)

			schemaPath := filepath.Join(schemasPath, pkg+".json")
			bridgetesting.AssertEqualsJSONFile(t, schemaPath, schema, nil)
		})
	}
}

// Tests that the converter correctly loads mappings for providers that are not part of the Pulumiverse, by requesting
// mapping for an appropriately parameterized instance of the Terraform provider plugin.
func TestTranslateParameterized(t *testing.T) {
	t.Parallel()

	// Arrange.
	testDir, err := filepath.Abs(filepath.Join("testdata"))
	require.NoError(t, err)

	testPath := filepath.Join(testDir, "terraform-provider")

	seen := map[string]*convert.MapperPackageHint{}

	mapper := &bridgetesting.MockMapper{
		GetMappingF: func(
			_ context.Context,
			provider string,
			hint *convert.MapperPackageHint,
		) ([]byte, error) {
			seen[provider] = hint
			return []byte{}, nil
		},
	}

	osFs := afero.NewOsFs()

	tempDir := t.TempDir()
	pclPath := filepath.Join(tempDir, "pcl")
	pclFs := afero.NewBasePathFs(osFs, pclPath)

	providerInfoSource := NewMapperProviderInfoSource(mapper)

	expectedToSee := map[string]*convert.MapperPackageHint{
		// "google" has a Pulumiverse provider ("gcp"), so we should expect the converter to request the Pulumi plugin with
		// that name, with no parameterization.
		"google": {
			PluginName: "gcp",
		},

		// "planetscale" is not a Pulumiverse provider, so we should expect the converter to request for it to be
		// dynamically bridged, by providing a mapping hint that mentions the terraform-provider plugin with an appropriate
		// parameterization.
		"planetscale": {
			PluginName: "terraform-provider",
			Parameterization: &workspace.Parameterization{
				Name:    "planetscale",
				Version: semver.MustParse("0.1.0"),
				Value:   []byte(`{"remote":{"url":"planetscale/planetscale","version":"0.1.0"}}`),
			},
		},
	}

	// Act.
	diagnostics := TranslateModule(osFs, testPath, pclFs, providerInfoSource, "/")

	// Assert.
	require.False(t, diagnostics.HasErrors(), "translate diagnostics should not have errors: %v", diagnostics)
	require.Equal(t, expectedToSee, seen, "expected to see an appropriate set of provider hints")
}

func componentProgramBinderFromAfero(fs afero.Fs) pcl.ComponentProgramBinder {
	return func(args pcl.ComponentProgramBinderArgs) (*pcl.Program, hcl.Diagnostics, error) {
		var diagnostics hcl.Diagnostics
		binderDirPath := args.BinderDirPath
		componentSource := args.ComponentSource
		nodeRange := args.ComponentNodeRange
		loader := args.BinderLoader
		// bind the component here as if it was a new program
		// this becomes the DirPath for the new binder
		componentSourceDir := filepath.Join(binderDirPath, componentSource)

		parser := syntax.NewParser()
		// Load all .pp files in the components' directory
		files, err := afero.ReadDir(fs, componentSourceDir)
		if err != nil {
			diagnostics = diagnostics.Append(errorf(nodeRange, "%s", err.Error()))
			return nil, diagnostics, nil
		}

		if len(files) == 0 {
			diagnostics = diagnostics.Append(errorf(nodeRange, "no .pp files found"))
			return nil, diagnostics, nil
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			fileName := file.Name()
			path := filepath.Join(componentSourceDir, fileName)

			if filepath.Ext(fileName) == ".pp" {
				file, err := fs.Open(path)
				if err != nil {
					diagnostics = diagnostics.Append(errorf(nodeRange, "%s", err.Error()))
					return nil, diagnostics, err
				}

				err = parser.ParseFile(file, fileName)
				if err != nil {
					diagnostics = diagnostics.Append(errorf(nodeRange, "%s", err.Error()))
					return nil, diagnostics, err
				}

				diags := parser.Diagnostics
				if diags.HasErrors() {
					return nil, diagnostics, err
				}
			}
		}

		componentProgram, programDiags, err := pcl.BindProgram(parser.Files,
			pcl.Loader(loader),
			pcl.DirPath(componentSourceDir),
			pcl.ComponentBinder(componentProgramBinderFromAfero(fs)))

		return componentProgram, programDiags, err
	}
}
