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

func TestL1PclKeywordOverlap(t *testing.T) {
	t.Skip("Converter bug: renamePclOverlap dereferences nil hclType at tf.go:960" +
		" when renaming variables with PCL keyword names")
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Config: map[string]string{
			"for":  "forValue",
			"if":   "ifValue",
			"else": "elseValue",
		},
		HCL: `
variable "for" {
  type = string
}

variable "if" {
  type = string
}

variable "else" {
  type = string
}

output "result_for" {
  value = var.for
}

output "result_if" {
  value = var.if
}

output "result_else" {
  value = var.else
}
`,
	})
}
