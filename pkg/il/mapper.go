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

package il

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tf2pulumi/il"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
)

// mapperProviderInfoSource wraps a convert.Mapper to return tfbridge.ProviderInfo
type mapperProviderInfoSource struct {
	mapper convert.Mapper
}

func NewMapperProviderInfoSource(mapper convert.Mapper) il.ProviderInfoSource {
	return &mapperProviderInfoSource{mapper: mapper}
}

func (mapper *mapperProviderInfoSource) GetProviderInfo(
	registryName, namespace, name, version string,
) (*tfbridge.ProviderInfo, error) {
	// TODO: Mapper has been made context aware, but ProviderInfoSource isn't.
	data, err := mapper.mapper.GetMapping(context.TODO(), name, &convert.MapperPackageHint{
		PluginName: il.GetPulumiProviderName(name),
	})
	if err != nil {
		return nil, err
	}
	// Might be nil or []
	if len(data) == 0 {
		return nil, fmt.Errorf(
			"could not find mapping information for provider %s; "+
				"try installing a pulumi plugin that supports this terraform provider",
			name,
		)
	}

	var info *tfbridge.MarshallableProviderInfo
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, fmt.Errorf("could not decode schema information for provider %s: %w", name, err)
	}
	return info.Unmarshal(), nil
}
