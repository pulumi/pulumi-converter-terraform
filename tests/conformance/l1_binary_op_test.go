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

func TestL1BinaryOp(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Config: map[string]string{
			"a": "10",
			"b": "2",
			"x": "true",
			"y": "false",
		},
		HCL: `
variable "a" {
  type = number
}

variable "b" {
  type = number
}

variable "x" {
  type = bool
}

variable "y" {
  type = bool
}

output "add" {
  value = var.a + var.b
}

output "subtract" {
  value = var.a - var.b
}

output "multiply" {
  value = var.a * var.b
}

output "divide" {
  value = var.a / var.b
}

output "modulo" {
  value = var.a % var.b
}

output "equal" {
  value = var.a == var.b
}

output "not_equal" {
  value = var.a != var.b
}

output "greater_than" {
  value = var.a > var.b
}

output "greater_than_or_equal" {
  value = var.a >= var.b
}

output "less_than" {
  value = var.a < var.b
}

output "less_than_or_equal" {
  value = var.a <= var.b
}

output "logical_and" {
  value = var.x && var.y
}

output "logical_or" {
  value = var.x || var.y
}
`,
	})
}
