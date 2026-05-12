variable "enabled" {
    type    = bool
    default = true
}

# Resource-level `for_each = <cond> ? [<x>] : []` gate. TF itself rejects this
# at apply (`for_each` requires a map or set of strings), but the converter
# still accepts the syntax and rewrites it: `range = <cond>` makes PCL produce
# an optional resource, and `each.value` / `each.key` are inlined as `<x>` and
# `0` respectively.
resource "simple_resource" "gated" {
    for_each  = var.enabled ? [42] : []
    input_one = each.value
    input_two = each.key == 0
}
