config "users" "list(optional(object({age=number, name=string})))" {
}

output "names" {
  value = users[*].name
}
