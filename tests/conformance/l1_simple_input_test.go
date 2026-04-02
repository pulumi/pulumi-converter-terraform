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

package conformance

import (
	"testing"

	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/conformance"
)

func TestL1SimpleInput(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Config: map[string]string{
			"number_in": "42",
			"any_in":    "test-value",
		},
		Input: map[string]string{"main.tf": `variable "opt_str_in" {
  default = "some string"
}

variable "number_in" {
    type = number
}

variable "any_in" {
}

output "region_out" {
    value = var.opt_str_in
}

output "number_out" {
    value = var.number_in
}

output "any_out" {
    value = var.any_in
}`},
	})
}
