config "names" "list(string)" {
}
output "greeting" {
  value = "%{for name in names~}Hello ${name}! %{endfor~}"
}
