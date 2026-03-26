config "item" "string" {
}

output "staticList" {
  value = ["a", "b", "c"]
}

output "listWithVar" {
  value = ["first", item, "last"]
}

output "nestedList" {
  value = [["x", "y"], ["z"]]
}
