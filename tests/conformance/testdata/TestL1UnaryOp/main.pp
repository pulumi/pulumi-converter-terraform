config "flag" "bool" {
}

config "num" "number" {
}

output "negatedFlag" {
  value = !flag
}

output "negatedNum" {
  value = -num
}
