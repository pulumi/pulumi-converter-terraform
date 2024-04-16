// Copyright 2016-2024, Pulumi Corporation.
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCachingMapper(t *testing.T) {
	calls := 0
	m := &testMapperWithFunc{
		getMapping: func(ctx context.Context, provider string, pulumiProvider string) ([]byte, error) {
			calls += 1
			if provider == "myprov" {
				return []byte("myprov-mappings"), nil
			}
			return nil, fmt.Errorf("unkonwn provider")
		},
	}

	cm := newCachingMapper(m)

	t.Run("error case", func(t *testing.T) {
		_, err := cm.GetMapping(context.Background(), "unknown", "unknown")
		require.Error(t, err)
		require.Equal(t, 1, calls)
	})

	t.Run("cache miss", func(t *testing.T) {
		mappings, err := cm.GetMapping(context.Background(), "myprov", "myprov")
		require.NoError(t, err)
		require.Equal(t, "myprov-mappings", string(mappings))
		require.Equal(t, 2, calls)
	})

	t.Run("cache hit", func(t *testing.T) {
		mappings, err := cm.GetMapping(context.Background(), "myprov", "myprov")
		require.NoError(t, err)
		require.Equal(t, "myprov-mappings", string(mappings))
		require.Equal(t, 2, calls)
	})
}

type testMapperWithFunc struct {
	getMapping func(context.Context, string, string) ([]byte, error)
}

func (m *testMapperWithFunc) GetMapping(ctx context.Context, provider string, pulumiProvider string) ([]byte, error) {
	return m.getMapping(ctx, provider, pulumiProvider)
}
