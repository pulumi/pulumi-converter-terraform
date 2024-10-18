locals {
  obj = {
    "snake_case_of_local_no_mangle" = "abc123",
    snake_case_of_local_no_mangle_unquoted = "abc123",
  }
}

output "unmangled_locals" {
  value = local.obj.snake_case_of_local_no_mangle
}

output "unmangled_locals_unquote" {
  value = local.obj.snake_case_of_local_no_mangle_unquoted
}
