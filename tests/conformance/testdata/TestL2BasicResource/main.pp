resource "example" "test:index/resource:Resource" {
  value = "hello"
}

output "value" {
  value = example.value
}

output "computedValue" {
  value = example.computedValue
}
