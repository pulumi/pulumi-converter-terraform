output "execution_role" {
  value = aws_iam_role.execution.arn
}

output "execution_role_name" {
  value = aws_iam_role.execution.name
}


output "latest_arn" {
  value = "arn:aws:ecs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}task-defintion/${var.name}"
}

output "arn" {
  value = aws_ecs_task_definition.def.arn
}
