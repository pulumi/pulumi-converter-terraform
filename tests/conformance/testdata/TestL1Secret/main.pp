output "wrapped" {
  value = secret("hello")
}

output "markedOnly" {
  value = "world"
}
