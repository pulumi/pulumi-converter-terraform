config "enabled" "bool" {
}

output "result" {
  value = enabled ? "yes" : "no"
}
