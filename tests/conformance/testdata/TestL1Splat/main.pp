config "users" "list(object({age=number, name=string}))" {
}

output "names" {
  value = users[*].name
}
