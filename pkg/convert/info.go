// Copyright 2025, Pulumi Corporation.
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
	"fmt"
	"sync"

	"github.com/blang/semver"
	"github.com/opentofu/opentofu/shim"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/terraform/pkg/configs"
)

// ProviderInfoSource is an interface for retrieving information about a bridged Terraform provider.
type ProviderInfoSource interface {
	// GetProviderInfo returns bridged provider information for the given Terraform provider.
	GetProviderInfo(
		tfProvider string,
		requiredProvider *configs.RequiredProvider,
	) (*tfbridge.ProviderInfo, error)
}

// mapperProviderInfoSource wraps a convert.Mapper to return tfbridge.ProviderInfo.
type mapperProviderInfoSource struct {
	mapper convert.Mapper
}

// NewMapperProviderInfoSource creates a new ProviderInfoSource that uses the provided mapper to build provider
// information.
func NewMapperProviderInfoSource(mapper convert.Mapper) ProviderInfoSource {
	return &mapperProviderInfoSource{mapper: mapper}
}

// Implements ProviderInfoSource.GetProviderInfo by working out whether the requested provider is a Pulumi-managed
// provider or a Terraform provider that needs to be dynamically bridged, and then retrieving the relevant information
// from the mapper.
func (s *mapperProviderInfoSource) GetProviderInfo(
	tfProvider string,
	requiredProvider *configs.RequiredProvider,
) (*tfbridge.ProviderInfo, error) {
	// First up, we need to work out whether the Terraform provider name is the one we should look for in the Pulumi
	// universe. For most providers, it is, but for some (e.g. "google") we need to rename it (e.g. to "gcp", which is the
	// Pulumi provider name for GCP). Generally, we'll use the Terraform name for errors, since that's the one the user
	// will have written in the program being converted.
	var pulumiProvider string
	if renamed, ok := pulumiRenamedProviderNames[tfProvider]; ok {
		pulumiProvider = renamed
	} else {
		pulumiProvider = tfProvider
	}

	var hint *convert.MapperPackageHint

	// If the Pulumi provider name is one that we manage ourselves, we'll use that provider to retrieve information about
	// mappings from Terraform to Pulumi. If not, then we'll assume that we are going to dynamically bridge a Terraform
	// provider, and thus provide a hint that asks the mapper to boot up the terraform-provider plugin and parameterize it
	// with the relevant Terraform provider details before retrieving a mapping.
	if isTerraformProvider(pulumiProvider) && requiredProvider != nil {
		tfVersion, diags := shim.FindTfPackageVersion(requiredProvider)
		if diags.HasErrors() {
			hint = &convert.MapperPackageHint{
				PluginName: pulumiProvider,
			}
		} else {
			version := semver.MustParse(tfVersion.String())

			hint = &convert.MapperPackageHint{
				PluginName: "terraform-provider",
				Parameterization: &workspace.Parameterization{
					Name:    tfProvider,
					Version: version,
					Value: []byte(fmt.Sprintf(
						`{"remote":{"url":"%s","version":"%s"}}`,
						requiredProvider.Source,
						tfVersion.String(),
					)),
				},
			}
		}
	} else {
		// Again, for non-bridged providers, the plugin name we want to find is the *Pulumi universe name* (e.g. "gcp", not
		// "google" for GCP). That said, when we finally call GetMapping, we are always passing the Terraform provider name,
		// since that's the thing we want mappings for.
		hint = &convert.MapperPackageHint{
			PluginName: pulumiProvider,
		}
	}

	mapping, err := s.mapper.GetMapping(context.TODO(), tfProvider, hint)
	if err != nil {
		return nil, err
	}

	// Might be nil or []
	if len(mapping) == 0 {
		return nil, fmt.Errorf(
			"could not find mapping information for provider %s; "+
				"try installing a pulumi plugin that supports this terraform provider",
			tfProvider,
		)
	}

	var info *tfbridge.MarshallableProviderInfo
	err = json.Unmarshal(mapping, &info)
	if err != nil {
		return nil, fmt.Errorf("could not decode mapping information for provider %s: %s", tfProvider, mapping)
	}

	return info.Unmarshal(), nil
}

// CachingProviderInfoSource wraps a ProviderInfoSource in a cache for faster access.
type CachingProviderInfoSource struct {
	lock sync.RWMutex

	source  ProviderInfoSource
	entries map[string]*tfbridge.ProviderInfo
}

// NewCachingProviderInfoSource creates a new CachingProviderInfoSource that wraps the given ProviderInfoSource.
func NewCachingProviderInfoSource(source ProviderInfoSource) *CachingProviderInfoSource {
	return &CachingProviderInfoSource{
		source:  source,
		entries: map[string]*tfbridge.ProviderInfo{},
	}
}

// GetProviderInfo returns the tfbridge information for the indicated Terraform provider as well as the name of the
// corresponding Pulumi resource provider.
func (s *CachingProviderInfoSource) GetProviderInfo(
	provider string,
	requiredProvider *configs.RequiredProvider,
) (*tfbridge.ProviderInfo, error) {
	if info, ok := s.getFromCache(provider); ok {
		return info, nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	info, err := s.source.GetProviderInfo(provider, requiredProvider)
	if err != nil {
		return nil, err
	}

	s.entries[provider] = info
	return info, nil
}

// getFromCache retrieves the provider information from the cache, taking a read lock to do so.
func (s *CachingProviderInfoSource) getFromCache(provider string) (*tfbridge.ProviderInfo, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	info, ok := s.entries[provider]
	return info, ok
}
