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
	"io"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge/tokens"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfgen"
	shimv2 "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfshim/sdk-v2"
	pulumidiag "github.com/pulumi/pulumi/sdk/v3/go/common/diag"
	"github.com/pulumi/pulumi/sdk/v3/go/common/diag/colors"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"github.com/stretchr/testify/require"
)

// BridgedProvider wraps a *schema.Provider into a tfbridge.ProviderInfo with auto-generated
// tokens and no-op CRUD methods where missing.
func BridgedProvider(t *testing.T, providerName string, tfp *schema.Provider) tfbridge.ProviderInfo {
	t.Helper()

	require.NoError(t, tfp.InternalValidate())

	shimProvider := shimv2.NewProvider(tfp)

	provider := tfbridge.ProviderInfo{
		P:                              shimProvider,
		Name:                           providerName,
		Version:                        "0.0.1",
		MetadataInfo:                   &tfbridge.MetadataInfo{},
		EnableZeroDefaultSchemaVersion: true,
	}
	provider.MustComputeTokens(tokens.SingleModule(providerName, "index", tokens.MakeStandard(providerName)))

	return provider
}

func providerServerFromInfo(
	ctx context.Context, providerInfo tfbridge.ProviderInfo,
) (pulumirpc.ResourceProviderServer, error) {
	sink := pulumidiag.DefaultSink(io.Discard, io.Discard, pulumidiag.FormatOptions{
		Color: colors.Never,
	})

	schema, err := tfgen.GenerateSchema(providerInfo, sink)
	if err != nil {
		return nil, fmt.Errorf("tfgen.GenerateSchema failed: %w", err)
	}

	schemaBytes, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		return nil, fmt.Errorf("json.MarshalIndent(schema, ..) failed: %w", err)
	}

	return tfbridge.NewProvider(
		ctx, nil, providerInfo.Name, providerInfo.Version, providerInfo.P, providerInfo, schemaBytes), nil
}
