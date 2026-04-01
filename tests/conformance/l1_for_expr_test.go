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

func TestL1ForExpr(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Config: map[string]string{
			"names":  `["alice","bob","charlie"]`,
			"labels": `{"env":"prod","team":"core"}`,
		},
		Input: map[string]string{"main.tf": `
variable "names" {
  type = list(string)
}

variable "labels" {
  type = map(string)
}

output "upper_names" {
  value = join(",", [for s in var.names : upper(s)])
}

output "label_entries" {
  value = join(",", [for k, v in var.labels : "${k}=${upper(v)}"])
}

output "short_names" {
  value = join(",", [for s in var.names : s if s != "bob"])
}
`},
	})
}
