data = {
  "first-123" = {
    randomStringLength = 12
  }
  "second-456" = {}
}

output "directLookup" {
  value = invoke("std:index:lookup", {
    map = {
      "a_key" = "value_a"
      "b_key" = "value_b"
    }
    key     = "a_key"
    default = "default"
  }).result
}

output "lookupWithLocals" {
  value = invoke("std:index:lookup", {
    map     = data["first-123"]
    key     = "random_string_length"
    default = 8
  }).result
}

output "nestedLookup" {
  value = invoke("std:index:lookup", {
    map = {
      "snake_case_key" = "first"
      "another_key"    = "second"
    }
    key     = "snake_case_key"
    default = "fallback"
  }).result
}
