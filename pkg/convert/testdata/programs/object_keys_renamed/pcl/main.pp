obj_key_snake = {
  "snake_case_of_local_no_mangle"        = "abc123"
  snake_case_of_local_no_mangle_unquoted = "abc123"
}

output "unmangledLocals" {
  value = obj_key_snake.snake_case_of_local_no_mangle
}

output "unmangledLocalsUnquote" {
  value = obj_key_snake.snake_case_of_local_no_mangle_unquoted
}
