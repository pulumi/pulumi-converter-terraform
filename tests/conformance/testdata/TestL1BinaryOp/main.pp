config "a" "number" {
}

config "b" "number" {
}

config "x" "bool" {
}

config "y" "bool" {
}

output "add" {
  value = a + b
}

output "subtract" {
  value = a - b
}

output "multiply" {
  value = a * b
}

output "divide" {
  value = a / b
}

output "modulo" {
  value = a % b
}

output "equal" {
  value = a == b
}

output "notEqual" {
  value = a != b
}

output "greaterThan" {
  value = a > b
}

output "greaterThanOrEqual" {
  value = a >= b
}

output "lessThan" {
  value = a < b
}

output "lessThanOrEqual" {
  value = a <= b
}

output "logicalAnd" {
  value = x && y
}

output "logicalOr" {
  value = x || y
}
