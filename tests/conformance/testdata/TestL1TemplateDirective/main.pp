config "names" "list(optional(string))" {
}
output "greeting" {
  value = "%{for name in names~}Hello ${name}! %{endfor~}"
}
