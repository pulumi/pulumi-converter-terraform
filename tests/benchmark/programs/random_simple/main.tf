resource "random_string" "bucket_name" {
  length  = 8
  special = false
}

output "name" {
  value = random_string.bucket_name.result
}
