// Copyright 2016-2026, Pulumi Corporation.
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

package convert

import (
	"context"
	"testing"

	bridgetesting "github.com/pulumi/pulumi-converter-terraform/pkg/testing"
	codegenconvert "github.com/pulumi/pulumi/pkg/v3/codegen/convert"
	"github.com/stretchr/testify/require"
)

func TestMapperProviderInfoSourceRejectsTrailingMappingData(t *testing.T) {
	t.Parallel()

	mapper := &bridgetesting.MockMapper{
		GetMappingF: func(
			context.Context,
			string,
			*codegenconvert.MapperPackageHint,
			string,
		) ([]byte, error) {
			return []byte("{} trailing data"), nil
		},
	}

	_, err := NewMapperProviderInfoSource(mapper).GetProviderInfo("aws", nil)
	require.ErrorContains(t, err, "could not decode mapping information for provider aws")
}

func TestMapperProviderInfoSourceAllowsTrailingMappingWhitespace(t *testing.T) {
	t.Parallel()

	mapper := &bridgetesting.MockMapper{
		GetMappingF: func(
			context.Context,
			string,
			*codegenconvert.MapperPackageHint,
			string,
		) ([]byte, error) {
			return []byte("{} \n\t"), nil
		},
	}

	_, err := NewMapperProviderInfoSource(mapper).GetProviderInfo("aws", nil)
	require.NoError(t, err)
}
