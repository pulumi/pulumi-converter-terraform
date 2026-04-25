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

package tfexec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

// Provider pairs a terraform provider name with an SDKv2 provider instance.
type Provider struct {
	Name     string
	Provider *schema.Provider
}

// Driver hosts TF providers in-process and runs terraform CLI against them.
type Driver struct {
	cwd             string
	reattachConfigs map[string]*plugin.ReattachConfig
	// Env is passed through to the terraform subprocess (and therefore to any
	// provisioner shell commands) on top of os.Environ().
	Env map[string]string
}

func init() {
	os.Setenv("TF_LOG_PROVIDER", "off")
	os.Setenv("TF_LOG_SDK", "off")
	os.Setenv("TF_LOG_SDK_PROTO", "off")
}

// NewDriver creates a Driver for the given SDKv2 providers. If no providers are given,
// the driver runs terraform without any reattach configuration.
func NewDriver(t *testing.T, providers []Provider) *Driver {
	t.Helper()

	reattachConfigs := make(map[string]*plugin.ReattachConfig, len(providers))
	for _, p := range providers {
		v6server, err := tf5to6server.UpgradeServer(context.Background(),
			func() tfprotov5.ProviderServer { return p.Provider.GRPCProvider() })
		require.NoError(t, err)

		reattachConfigCh := make(chan *plugin.ReattachConfig)
		closeCh := make(chan struct{})

		serverOpts := []tf6server.ServeOpt{
			tf6server.WithGoPluginLogger(hclog.FromStandardLogger(log.New(io.Discard, "", 0), hclog.DefaultOptions)),
			tf6server.WithDebug(t.Context(), reattachConfigCh, closeCh),
			tf6server.WithoutLogStderrOverride(),
		}

		name := p.Name
		go func() {
			err := tf6server.Serve(name, func() tfprotov6.ProviderServer { return v6server }, serverOpts...)
			if err != nil {
				t.Logf("tf6server.Serve error: %v", err)
			}
		}()

		reattachConfigs[p.Name] = <-reattachConfigCh
	}

	return &Driver{
		cwd:             t.TempDir(),
		reattachConfigs: reattachConfigs,
	}
}

// Apply writes the input files, runs terraform init + apply, and returns all outputs.
// Config values are passed as -var flags to terraform apply.
func (d *Driver) Apply(t *testing.T, input map[string]string, config map[string]string) map[string]string {
	t.Helper()

	for path, content := range input {
		fullPath := filepath.Join(d.cwd, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	_, err := d.execTf(t, "init", "-backend=false")
	require.NoError(t, err)

	applyArgs := append(make([]string, 0, 4+2*len(config)), "apply", "-auto-approve", "-refresh=false")
	for k, v := range config {
		applyArgs = append(applyArgs, "-var", k+"="+v)
	}
	_, err = d.execTf(t, applyArgs...)
	require.NoError(t, err)

	return d.parseOutputs(t)
}

func (d *Driver) parseOutputs(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(d.cwd, "terraform.tfstate"))
	require.NoError(t, err)

	var state struct {
		Outputs map[string]struct {
			Value json.RawMessage `json:"value"`
		} `json:"outputs"`
	}
	require.NoError(t, json.Unmarshal(raw, &state))

	result := make(map[string]string, len(state.Outputs))
	for k, v := range state.Outputs {
		var s string
		if err := json.Unmarshal(v.Value, &s); err == nil {
			result[k] = s
		} else {
			result[k] = string(v.Value)
		}
	}
	return result
}

func (d *Driver) formatReattachEnvVar() string {
	if len(d.reattachConfigs) == 0 {
		return ""
	}

	type reattachConfigAddr struct {
		Network string
		String  string
	}

	type reattachConfig struct {
		Protocol        string
		ProtocolVersion int
		Pid             int
		Test            bool
		Addr            reattachConfigAddr
	}

	configs := make(map[string]reattachConfig, len(d.reattachConfigs))
	for name, rc := range d.reattachConfigs {
		configs[name] = reattachConfig{
			Protocol:        string(rc.Protocol),
			ProtocolVersion: rc.ProtocolVersion,
			Pid:             rc.Pid,
			Test:            rc.Test,
			Addr: reattachConfigAddr{
				Network: rc.Addr.Network(),
				String:  rc.Addr.String(),
			},
		}
	}

	reattachBytes, err := json.Marshal(configs)
	if err != nil {
		panic(fmt.Sprintf("failed to build TF_REATTACH_PROVIDERS string: %v", err))
	}
	return "TF_REATTACH_PROVIDERS=" + string(reattachBytes)
}

func getTFCommand() string {
	if cmd := os.Getenv("TF_COMMAND_OVERRIDE"); cmd != "" {
		return cmd
	}
	return "tofu"
}
