# When converted, the object inside lookup should maintain the casing of the keys
# This tests bug #372: lookup function keys must preserve original casing

locals {
  data = {
    "first-123" = {
      random_string_length = 12
    }
    "second-456" = {}
  }
}

output "direct_lookup" {
  value = lookup({a_key="value_a", b_key="value_b"}, "a_key", "default")
}

output "lookup_with_locals" {
  value = lookup(local.data["first-123"], "random_string_length", 8)
}

output "nested_lookup" {
  value = lookup({
    snake_case_key = "first",
    another_key = "second"
  }, "snake_case_key", "fallback")
}
