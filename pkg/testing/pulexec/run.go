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

package pulexec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pulumi/providertest/providers"
	"github.com/pulumi/providertest/pulumitest"
	"github.com/pulumi/providertest/pulumitest/opttest"
	"github.com/pulumi/pulumi-converter-terraform/pkg/convert"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/rpcutil"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var (
	buildPCLOnce sync.Once
	pclBinDir    string
	pclBuildErr  error
)

func ensurePCLLanguagePlugin(t *testing.T) string {
	t.Helper()
	buildPCLOnce.Do(func() {
		dir, err := os.MkdirTemp("", "pulumi-language-pcl-*")
		if err != nil {
			pclBuildErr = fmt.Errorf("creating temp dir: %w", err)
			return
		}
		bin := filepath.Join(dir, "pulumi-language-pcl")
		cmd := exec.Command("go", "build", "-o", bin, "github.com/pulumi/pulumi/sdk/pcl/v3/cmd/pulumi-language-pcl")
		out, err := cmd.CombinedOutput()
		if err != nil {
			pclBuildErr = fmt.Errorf("building pulumi-language-pcl: %w\n%s", err, out)
			return
		}
		pclBinDir = dir
	})
	require.NoError(t, pclBuildErr)
	return pclBinDir
}

// Provider pairs a provider name with its bridged info.
type Provider struct {
	Name string
	Info tfbridge.ProviderInfo
}

// Result holds the outputs and resource state from a Pulumi deployment.
type Result struct {
	Outputs   map[string]string
	Resources []apitype.ResourceV3
}

// Run writes a Pulumi.yaml and .pp files to a temp dir, starts the bridged providers on
// gRPC, runs `pulumi up` via pulumitest, and returns stack outputs and resource state.
// Config values are set on the stack before deployment.
//
// programFiles maps relative paths to file contents (e.g. {"main.pp": "...", "mod/main.pp": "..."}).
//
// env is passed through to the pulumi CLI subprocess, which propagates it to language and
// resource provider plugins (including command:local:Command's shell).
func Run(
	t *testing.T, provs []Provider, programFiles map[string]string,
	config map[string]string, env map[string]string,
) Result {
	t.Helper()

	binDir := ensurePCLLanguagePlugin(t)

	dir := t.TempDir()

	// The project name is used as the default namespace for user config. It
	// must not collide with any attached provider name, or user config like
	// "<project>:foo" would be misrouted to the provider.
	pulumiYAML := `name: conformance
runtime: pcl
backend:
  url: file://` + filepath.Join(dir, "state") + "\n"

	err := os.WriteFile(filepath.Join(dir, "Pulumi.yaml"), []byte(pulumiYAML), 0o600)
	require.NoError(t, err)

	for path, content := range programFiles {
		fullPath := filepath.Join(dir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	opts := append(make([]opttest.Option, 0, 4+len(env)+len(provs)),
		opttest.Env("PULUMI_DISABLE_AUTOMATIC_PLUGIN_ACQUISITION", "true"),
		opttest.Env("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH")),
		opttest.TestInPlace(),
		opttest.SkipInstall(),
	)

	for k, v := range env {
		opts = append(opts, opttest.Env(k, v))
	}

	for _, p := range provs {
		info := p.Info
		opts = append(opts, opttest.AttachProvider(
			p.Name,
			func(ctx context.Context, pt providers.PulumiTest) (providers.Port, error) {
				handle, err := startProvider(ctx, info)
				if err != nil {
					return 0, err
				}
				return providers.Port(handle.Port), nil
			},
		))
	}

	pt := pulumitest.NewPulumiTest(t, dir, opts...)

	for k, v := range config {
		pt.SetConfig(t, convert.CamelCaseName(k), v)
	}

	upResult := pt.Up(t)

	outputs := make(map[string]string, len(upResult.Outputs))
	for k, v := range upResult.Outputs {
		if s, ok := v.Value.(string); ok {
			outputs[k] = s
		} else {
			raw, err := json.Marshal(v.Value)
			require.NoError(t, err)
			outputs[k] = string(raw)
		}
	}

	exported := pt.ExportStack(t)
	var deployment apitype.DeploymentV3
	require.NoError(t, json.Unmarshal(exported.Deployment, &deployment))

	return Result{
		Outputs:   outputs,
		Resources: deployment.Resources,
	}
}

func startProvider(ctx context.Context, providerInfo tfbridge.ProviderInfo) (*rpcutil.ServeHandle, error) {
	prov, err := providerServerFromInfo(ctx, providerInfo)
	if err != nil {
		return nil, fmt.Errorf("providerServerFromInfo failed: %w", err)
	}

	handle, err := rpcutil.ServeWithOptions(rpcutil.ServeOptions{
		Init: func(srv *grpc.Server) error {
			pulumirpc.RegisterResourceProviderServer(srv, prov)
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("rpcutil.ServeWithOptions failed: %w", err)
	}

	return &handle, nil
}
