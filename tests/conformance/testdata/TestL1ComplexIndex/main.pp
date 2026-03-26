config "items" "list(string)" {
}

config "idx" "number" {
}

output "selected" {
  value = invoke("std:index:sort", {
    input = items
  }).result[idx]
}
