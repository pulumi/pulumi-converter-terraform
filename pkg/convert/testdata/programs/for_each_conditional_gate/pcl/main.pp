config "enabled" "bool" {
  default = true
}


# Resource-level `for_each = <cond> ? [<x>] : []` gate. TF itself rejects this
# at apply (`for_each` requires a map or set of strings), but the converter
# still accepts the syntax and rewrites it: `range = <cond>` makes PCL produce
# an optional resource, and `each.value` / `each.key` are inlined as `<x>` and
# `0` respectively.
resource "gated" "simple:index:resource" {
  options {
    range = enabled
  }
  inputOne = 42
  inputTwo = 0 == 0
}
