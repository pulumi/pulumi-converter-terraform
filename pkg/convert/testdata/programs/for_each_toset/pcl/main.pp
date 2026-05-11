resource "tagged" "simple:index:resource" {
  options {
    range = { for entry in ["alpha", "beta"] : entry => entry }
  }
  inputOne = range.value
  inputTwo = false
}

output "alphaResult" {
  value = tagged["alpha"].result
}
