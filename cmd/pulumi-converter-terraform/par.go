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

package main

import (
	"errors"
	"runtime"
	"sync"
)

// Transforms map values in parallel over n workers. If workers is negative use NumCPU.
func parTransformMapWith[K comparable, T any, U any](
	inputs map[K]T,
	transform func(K, T) (U, error),
	workers int,
) (map[K]U, error) {
	n := workers
	if workers < 1 {
		n = runtime.NumCPU()
		if n < 2 {
			n = 2
		}
	}

	type kv struct {
		k K
		v T
	}

	ch := make(chan kv)
	errorSlice := make([]error, n)

	// Start n workers to do convertViaPulumiCLI work
	wg := sync.WaitGroup{}
	wg.Add(n)

	var results sync.Map

	for i := 0; i < n; i++ {
		go func(worker int) {
			defer wg.Done()
			for entry := range ch {
				result, err := transform(entry.k, entry.v)
				if err != nil {
					errorSlice[worker] = err
					return
				}
				results.Store(entry.k, result)
			}
		}(i)
	}

	for k, v := range inputs {
		ch <- kv{k, v}
	}

	close(ch)

	wg.Wait()

	if err := errors.Join(errorSlice...); err != nil {
		return nil, err
	}

	translatedMap := map[K]U{}
	results.Range(func(k, v any) bool {
		translatedMap[k.(K)] = v.(U)
		return true
	})

	return translatedMap, nil
}
