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
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParTransformMap(t *testing.T) {

	mkMap := func(n int) map[int]int {
		m := map[int]int{}
		for i := 0; i < n; i++ {
			m[i] = 2 * i
		}
		return m
	}

	inputs := mkMap(10)

	inputsBad := mkMap(10)
	inputsBad[4] = -8

	type testCase struct {
		inputs  map[int]int
		workers int
	}

	increment := func(k, v int) (int, error) {
		if v < 0 {
			return 0, fmt.Errorf("neg")
		}
		return v + 1, nil
	}

	testCases := []testCase{
		{inputs, -1},
		{inputs, 2},
		{inputs, 4},
		{inputsBad, -1},
		{inputsBad, 2},
		{inputsBad, 4},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(fmt.Sprintf("w%d", tc.workers), func(t *testing.T) {
			var ops atomic.Uint64

			inc := func(k, v int) (int, error) {
				ops.Add(1)
				return increment(k, v)
			}

			actual, actualErr := parTransformMapWith(tc.inputs, inc, tc.workers)
			expect, expectErr := apply(increment, tc.inputs)
			assert.Equal(t, len(tc.inputs), int(ops.Load()))
			assert.Equal(t, expectErr, actualErr)
			assert.Equal(t, expect, actual)
		})
	}
}

func apply[K comparable, T, U any](f func(K, T) (U, error), m map[K]T) (map[K]U, error) {
	r := make(map[K]U)
	for key, value := range m {
		var err error
		r[key], err = f(key, value)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}
