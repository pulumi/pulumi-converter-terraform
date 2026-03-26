output "upper" {
  value = invoke("std:index:upper", {
    input = "hello"
  }).result
}

output "lower" {
  value = invoke("std:index:lower", {
    input = "HELLO"
  }).result
}

output "joined" {
  value = invoke("std:index:join", {
    separator = "-"
    input     = ["a", "b", "c"]
  }).result
}

output "len" {
  value = length(["x", "y"])
}
