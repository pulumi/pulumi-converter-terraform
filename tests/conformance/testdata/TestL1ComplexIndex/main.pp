config "items" "list(optional(string))" {
}

config "idx" "number" {
}

output "selected" {
  value = invoke("std:index:sort", {
    input = items
  }).result[idx]
}
