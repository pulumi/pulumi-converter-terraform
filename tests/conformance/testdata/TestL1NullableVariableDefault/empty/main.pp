config "list" "list(optional(string))" {
  default  = [null]
  nullable = true
}

output "listOutput" {
  value = list
}

config "string" "string" {
  default  = null
  nullable = true
}

output "stringOutput" {
  value = [string]
}
