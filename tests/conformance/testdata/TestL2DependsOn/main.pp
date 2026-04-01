resource "first" "test:index/resource:Resource" {
  value = "first"
}
resource "second" "test:index/resource:Resource" {
  options {
    dependsOn = [first]
  }
  value = "second"
}
output "result" {
  value = second.computedValue
}
