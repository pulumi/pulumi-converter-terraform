items = ["a", "b", "c"]
mapping = {
  key = "value"
}

output "firstItem" {
  value = items[0]
}

output "mapValue" {
  value = mapping["key"]
}
