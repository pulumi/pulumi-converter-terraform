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
	"github.com/pulumi/pulumi-converter-terraform/tests/conformance/providers"
)

func TestL2NameConflict(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "simple", Factory: providers.SimpleProvider},
		},
		Config: map[string]string{
			"a_thing": "test-value",
		},
		Input: map[string]string{"main.tf": `variable "a_thing" {

}

locals {
    a_thing = true
}

resource "simple_resource" "a_thing" {
    input_one = "Hello ${var.a_thing}"
    input_two = local.a_thing
}

data "simple_data_source" "a_thing" {
    input_one = "Hello ${simple_resource.a_thing.result}"
    input_two = local.a_thing
}

resource "simple_another_resource" "a_thing" {
    input_one = "Hello ${simple_resource.a_thing.result}"
}

output "a_thing" {
    value = data.simple_data_source.a_thing.result
}`},
	})
}
