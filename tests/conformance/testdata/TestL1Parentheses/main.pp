config "a" "number" {
}

config "b" "number" {
}

output "result" {
  value = (a + b) * 2
}

output "grouped" {
  value = a * (b + 1)
}
