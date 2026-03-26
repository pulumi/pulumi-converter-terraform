config "name" "string" {
}

output "greeting" {
  value = "Hello, ${name}!"
}

output "plain" {
  value = "no interpolation"
}

output "wrapped" {
  value = "${name}"
}
