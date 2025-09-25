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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
)

type TestFileMapper struct {
	Path string
}

func (l *TestFileMapper) GetMapping(
	_ context.Context,
	provider string,
	hint *convert.MapperPackageHint,
) ([]byte, error) {
	pulumiProvider := provider
	if hint != nil {
		pulumiProvider = hint.PluginName
	}
	if pulumiProvider == "" {
		panic("provider and hint cannot both be empty")
	}

	if provider == "tfe" {
		// a known parameterized provider for whuch we have local mappings for
		pulumiProvider = "tfe"
	}

	if pulumiProvider == "unknown" {
		// 'unknown' is used as a known provider name that will return nothing, so return early here so we
		// don't hit the standard unknown error below.
		return nil, nil
	}

	if pulumiProvider == "error" {
		// 'error' is used as a known provider name that will cause GetMapping to error, so return early here
		// so we don't hit the standard unknown error below.
		return nil, errors.New("test error")
	}

	mappingPath := filepath.Join(l.Path, pulumiProvider) + ".json"
	mappingBytes, err := os.ReadFile(mappingPath)
	if err != nil {
		if os.IsNotExist(err) {
			panic(fmt.Sprintf("provider %s (%s) is not known to the test system", provider, pulumiProvider))
		}
		panic(err)
	}

	return mappingBytes, nil
}

// MockMapper provides a way to mock the Mapper interface for testing purposes.
type MockMapper struct {
	// GetMappingF is a function that will be called when Mapper.GetMapping is invoked.
	GetMappingF func(
		context.Context,
		string,
		*convert.MapperPackageHint,
	) ([]byte, error)
}

func (m *MockMapper) GetMapping(
	ctx context.Context,
	provider string,
	hint *convert.MapperPackageHint,
) ([]byte, error) {
	if m.GetMappingF == nil {
		panic("GetMappingF is not implemented")
	}

	return m.GetMappingF(ctx, provider, hint)
}
