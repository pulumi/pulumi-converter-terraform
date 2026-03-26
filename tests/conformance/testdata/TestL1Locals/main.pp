greeting = "hello"
name     = "world"
message  = "${greeting}, ${name}!"

output "message" {
  value = message
}

output "name" {
  value = name
}
