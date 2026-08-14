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
	"errors"
	"sync"
	"testing"
	"time"

	version "github.com/hashicorp/go-version"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/terraform/pkg/configs"
	"github.com/stretchr/testify/require"
)

type providerInfoSourceFunc func(string, *configs.RequiredProvider) (*tfbridge.ProviderInfo, error)

func (f providerInfoSourceFunc) GetProviderInfo(
	provider string,
	requiredProvider *configs.RequiredProvider,
) (*tfbridge.ProviderInfo, error) {
	return f(provider, requiredProvider)
}

func TestCachingProviderInfoSourceDeduplicatesConcurrentRequests(t *testing.T) {
	t.Parallel()

	const callers = 32
	want := &tfbridge.ProviderInfo{}
	var calls int
	source := providerInfoSourceFunc(func(
		_ string, _ *configs.RequiredProvider,
	) (*tfbridge.ProviderInfo, error) {
		calls++
		return want, nil
	})
	cache := NewCachingProviderInfoSource(source)

	start := make(chan struct{})
	results := make([]*tfbridge.ProviderInfo, callers)
	errs := make([]error, callers)
	var workers sync.WaitGroup
	workers.Add(callers)
	for i := range callers {
		go func() {
			defer workers.Done()
			<-start
			results[i], errs[i] = cache.GetProviderInfo("aws", nil)
		}()
	}
	close(start)
	workers.Wait()

	require.Equal(t, 1, calls)
	for i := range callers {
		require.NoError(t, errs[i])
		require.Same(t, want, results[i])
	}
}

func TestCachingProviderInfoSourceDoesNotBlockUnrelatedCachedKeys(t *testing.T) {
	t.Parallel()

	cachedInfo := &tfbridge.ProviderInfo{}
	slowInfo := &tfbridge.ProviderInfo{}
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseSlow) }) }
	t.Cleanup(release)

	source := providerInfoSourceFunc(func(
		provider string, _ *configs.RequiredProvider,
	) (*tfbridge.ProviderInfo, error) {
		switch provider {
		case "cached":
			return cachedInfo, nil
		case "slow":
			close(slowStarted)
			<-releaseSlow
			return slowInfo, nil
		default:
			return nil, errors.New("unexpected provider")
		}
	})
	cache := NewCachingProviderInfoSource(source)

	got, err := cache.GetProviderInfo("cached", nil)
	require.NoError(t, err)
	require.Same(t, cachedInfo, got)

	slowDone := make(chan error, 1)
	go func() {
		_, err := cache.GetProviderInfo("slow", nil)
		slowDone <- err
	}()
	<-slowStarted

	cachedDone := make(chan error, 1)
	go func() {
		got, err := cache.GetProviderInfo("cached", nil)
		if err == nil && got != cachedInfo {
			err = errors.New("cached provider info changed")
		}
		cachedDone <- err
	}()

	select {
	case err := <-cachedDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("cached lookup blocked behind an unrelated source request")
	}

	release()
	require.NoError(t, <-slowDone)
}

func TestCachingProviderInfoSourceDoesNotCacheErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("transient mapper failure")
	calls := 0
	source := providerInfoSourceFunc(func(
		_ string, _ *configs.RequiredProvider,
	) (*tfbridge.ProviderInfo, error) {
		calls++
		return nil, wantErr
	})
	cache := NewCachingProviderInfoSource(source)

	_, err := cache.GetProviderInfo("aws", nil)
	require.ErrorIs(t, err, wantErr)
	_, err = cache.GetProviderInfo("aws", nil)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 2, calls)
}

func TestCachingProviderInfoSourceSeparatesDynamicProviderIdentities(t *testing.T) {
	t.Parallel()

	constraint1, err := version.NewConstraint("~> 1.0")
	require.NoError(t, err)
	constraint2, err := version.NewConstraint("~> 2.0")
	require.NoError(t, err)

	required1 := &configs.RequiredProvider{
		Source: "registry.example.com/example/provider",
		Requirement: configs.VersionConstraint{
			Required: constraint1,
		},
	}
	required2 := &configs.RequiredProvider{
		Source: "registry.example.com/example/provider",
		Requirement: configs.VersionConstraint{
			Required: constraint2,
		},
	}
	required3 := &configs.RequiredProvider{
		Source: "registry.example.com/other/provider",
		Requirement: configs.VersionConstraint{
			Required: constraint1,
		},
	}
	want1 := &tfbridge.ProviderInfo{}
	want2 := &tfbridge.ProviderInfo{}
	want3 := &tfbridge.ProviderInfo{}
	infos := map[string]*tfbridge.ProviderInfo{
		providerInfoCacheKey("example", required1): want1,
		providerInfoCacheKey("example", required2): want2,
		providerInfoCacheKey("example", required3): want3,
	}
	calls := 0
	source := providerInfoSourceFunc(func(
		provider string, required *configs.RequiredProvider,
	) (*tfbridge.ProviderInfo, error) {
		calls++
		return infos[providerInfoCacheKey(provider, required)], nil
	})
	cache := NewCachingProviderInfoSource(source)

	got1, err := cache.GetProviderInfo("example", required1)
	require.NoError(t, err)
	got1Again, err := cache.GetProviderInfo("example", required1)
	require.NoError(t, err)
	got2, err := cache.GetProviderInfo("example", required2)
	require.NoError(t, err)
	got3, err := cache.GetProviderInfo("example", required3)
	require.NoError(t, err)

	require.Same(t, want1, got1)
	require.Same(t, want1, got1Again)
	require.Same(t, want2, got2)
	require.Same(t, want3, got3)
	require.Equal(t, 3, calls)
}
