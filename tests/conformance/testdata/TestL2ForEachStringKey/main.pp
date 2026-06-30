resource "mapped" "test:index/resource:Resource" {
  options {
    range = { for entry in ["alpha", "beta"] : entry => entry }
  }
  value = range.value
}

output "alpha" {
  value = mapped["alpha"].computedValue
}

output "beta" {
  value = mapped["beta"].computedValue
}
