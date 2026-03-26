resource "example" "test:index/resource:Resource" {
  options {
    ignoreChanges = [value]
  }
  value = "hello"
}

output "result" {
  value = example.computedValue
}
