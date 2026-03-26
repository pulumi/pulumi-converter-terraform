resource "multi" "test:index/resource:Resource" {
  options {
    range = 3
  }
  value = "item-${range.value}"
}

output "firstValue" {
  value = multi[0].computedValue
}
