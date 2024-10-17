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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	tfconvert "github.com/pulumi/pulumi-converter-terraform/pkg/convert"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tf2pulumi/il"
	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/logging"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/rpcutil"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"github.com/pulumi/terraform/pkg/configs"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

type tfConverter struct{}

func (*tfConverter) Close() error {
	return nil
}

func (*tfConverter) ConvertState(_ context.Context,
	req *plugin.ConvertStateRequest,
) (*plugin.ConvertStateResponse, error) {
	mapper, err := convert.NewMapperClient(req.MapperTarget)
	if err != nil {
		return nil, fmt.Errorf("create mapper: %w", err)
	}
	providerInfoSource := il.NewMapperProviderInfoSource(mapper)

	if len(req.Args) != 1 {
		return nil, fmt.Errorf("expected exactly one argument")
	}
	path := req.Args[0]

	return tfconvert.TranslateState(providerInfoSource, path)
}

type translatedExample struct {
	PCL         string          `json:"pcl"`
	PulumiYAML  string          `json:"pulumiYaml"`
	Diagnostics hcl.Diagnostics `json:"diagnostics"`
}

func (*tfConverter) ConvertProgram(_ context.Context,
	req *plugin.ConvertProgramRequest,
) (*plugin.ConvertProgramResponse, error) {
	flags := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	// --convert-examples is a special flag that allows us to convert a set of examples in a single run of `pulumi
	// convert`. This is a bit of a hack to support bulk conversion of terraform examples without needing a fresh run of
	// `pulumi convert` (and thus a fresh instance of the converter and schema loadings) per example.
	//
	// The intention is to move this to a cleaner model using `plugin run` in the future
	// (https://github.com/pulumi/pulumi-converter-terraform/issues/180), but it's not a pressing priority.
	//
	// The expected input is a JSON file with a map of example names to example contents. The converter will then
	// convert each example and write all the results out in a new JSON file (with the same name, but in the target
	// directory). That JSON file will be a map of example name to {pcl, yaml, diagnostics}, pcl being the converted
	// pcl, and yaml being the converted Pulumi.yaml.
	//
	// This is then used by tfbridge like `pulumi convert --from=terraform --language=pcl --out=./out --generate-only --
	// --convert-examples=examples.json`. The `--language=pcl` is important, as the cli would otherwise try to take
	// non-existing PCL output it normally expects from ConvertProgram and pass it off to the language plugins to
	// further convert. `--out` is important and must be different from the input file directory, as the converter will
	// use the same file name as the input file for it's output file.
	//
	// Using this option completely ignores the source directory that ConvertProgram would normally look at to convert.
	convertExamples := flags.String("convert-examples", "", "path to a terraform bridge example file to convert")
	err := flags.Parse(req.Args)
	if err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	mapper, err := convert.NewMapperClient(req.MapperTarget)
	if err != nil {
		return nil, fmt.Errorf("create mapper: %w", err)
	}

	terraformProviderDeps, err := readAllTerraformDependencies(afero.NewOsFs(), "./")
	if err != nil {
		return nil, fmt.Errorf("could not read terraform deps: %w", err)
	}

	providerInfoSource := il.NewMapperProviderInfoSourceWithDependencies(mapper, terraformProviderDeps)

	if *convertExamples != "" {
		examplesBytes, err := os.ReadFile(filepath.Join(req.SourceDirectory, *convertExamples))
		if err != nil {
			return nil, fmt.Errorf("read examples.json: %w", err)
		}

		var examples map[string]string
		err = json.Unmarshal(examplesBytes, &examples)
		if err != nil {
			return nil, fmt.Errorf("unmarshal examples.json, expected map[string]string: %w", err)
		}

		// For each example make up a small InMemFs for it and run the translation and save the results
		translateExample := func(name, example string) (translatedExample, error) {
			src := afero.NewMemMapFs()
			safename := strings.ReplaceAll(name, "/", "-")
			err := afero.WriteFile(src, "/"+safename+".tf", []byte(example), 0o600)
			if err != nil {
				return translatedExample{}, fmt.Errorf("write example %s to memory store: %w", name, err)
			}

			dst := afero.NewMemMapFs()
			diags := tfconvert.TranslateModule(src, "/", dst, providerInfoSource)

			pcl, err := afero.ReadFile(dst, "/"+safename+".pp")
			if err != nil && !os.IsNotExist(err) {
				return translatedExample{}, fmt.Errorf("read example %s from memory store: %w", name, err)
			}

			yml, err := afero.ReadFile(dst, "/Pulumi.yaml")
			if err != nil && !os.IsNotExist(err) {
				return translatedExample{}, fmt.Errorf("read example Pulumi.yaml %s from memory store: %w", name, err)
			}

			return translatedExample{
				PCL:         string(pcl),
				PulumiYAML:  string(yml),
				Diagnostics: diags,
			}, nil
		}

		workers := -1 // numCPU

		results, err := parTransformMapWith(examples, translateExample, workers)
		if err != nil {
			return nil, fmt.Errorf("translate examples: %w", err)
		}

		// Now marshal the results and return them, we use the same base name as our input file but written to the
		// target directory
		resultsBytes, err := json.Marshal(results)
		if err != nil {
			return nil, fmt.Errorf("marshal results: %w", err)
		}
		basename := filepath.Base(*convertExamples)
		err = os.WriteFile(filepath.Join(req.TargetDirectory, basename), resultsBytes, 0o600)
		if err != nil {
			return nil, fmt.Errorf("write results: %w", err)
		}
		// We don't return any diagnostics here, the bridge will parse them out of the examples.json file.
		return &plugin.ConvertProgramResponse{}, nil
	}

	// Normal path, just doing a plain module translation
	fs := afero.NewOsFs()
	dst := afero.NewBasePathFs(fs, req.TargetDirectory)

	diags := tfconvert.TranslateModule(fs, req.SourceDirectory, dst, providerInfoSource)
	return &plugin.ConvertProgramResponse{
		Diagnostics: diags,
	}, nil
}

func main() {
	// Fire up a gRPC server, letting the kernel choose a free port for us.
	handle, err := rpcutil.ServeWithOptions(rpcutil.ServeOptions{
		Init: func(srv *grpc.Server) error {
			pulumirpc.RegisterConverterServer(srv, plugin.NewConverterServer(&tfConverter{}))
			return nil
		},
		Options: rpcutil.OpenTracingServerInterceptorOptions(nil),
	})
	if err != nil {
		log.Fatalf("fatal: %v", err)
	}

	// The converter protocol requires that we now write out the port we have chosen to listen on.
	fmt.Printf("%d\n", handle.Port)

	// Finally, wait for the server to stop serving.
	if err := <-handle.Done; err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func readAllTerraformDependencies(fs afero.Fs, dir string) (map[string]string, error) {
	result := make(map[string]string)

	p := configs.NewParser(fs)
	mod, diags := p.LoadConfigDir(dir)
	if len(diags.Errs()) != 0 {
		return nil, fmt.Errorf("error parsing hcl: %s", diags.Error())
	}

	for name, provider := range mod.ProviderRequirements.RequiredProviders {
		logging.V(4).Infof("Found dep: %s: %v", name, provider.Source)
		result[name] = provider.Source
	}

	return result, nil
}
