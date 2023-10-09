variable "data" {
  type = string
  default = "Test"
}

output "data" {
  value = var.data
}