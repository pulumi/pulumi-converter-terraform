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
	"sync"

	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
)

func newCachingMapper(m convert.Mapper) convert.Mapper {
	return &cachingMapper{
		inner: m,
		cache: make(map[string]map[string][]byte),
	}
}

type cachingMapper struct {
	inner convert.Mapper
	mu    sync.Mutex
	cache map[string]map[string][]byte
}

func (cm *cachingMapper) GetMapping(ctx context.Context, provider string, pulumiProvider string) ([]byte, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	_, ok := cm.cache[provider]
	if !ok {
		cm.cache[provider] = make(map[string][]byte)
	}
	cached, cacheHit := cm.cache[provider][pulumiProvider]
	if cacheHit {
		return cached, nil
	}
	mapping, err := cm.inner.GetMapping(ctx, provider, pulumiProvider)
	if err != nil {
		return nil, err
	}
	cm.cache[provider][pulumiProvider] = mapping
	return mapping, nil
}
