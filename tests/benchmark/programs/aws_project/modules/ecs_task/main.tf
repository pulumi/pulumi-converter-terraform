data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

locals {
  // making it easier for the end-user to pick a cpu/memory combination
  // by simply entering a t-shirt size option. Users still have the option
  // of overwriting and specifying  a specific value for both of needed.
  cpu = coalesce(
    var.cpu,
    lookup({
      low    = 256
      medium = 512
      high   = 1024
      }, var.resource_allocation, null
  ))

  memory = coalesce(
    var.memory,
    lookup({
      low    = 512
      medium = 1024
      high   = 2048
      }, var.resource_allocation, null
  ))

  logs = {
    logDriver = "awslogs"
    options = {
      "awslogs-group"         = "/ecs/${var.name}-${var.environment}"
      "awslogs-region"        = data.aws_region.current.name
      "awslogs-stream-prefix" = var.name
    }
    secretOptions = []
  }


  // add additional options here as needed
  container_def = {
    name                  = var.name
    image                 = var.container_image
    portMappings          = var.port_mappings
    firelensConfiguration = var.firelens_configuration
    logConfiguration      = coalesce(var.log_configuration, local.logs)
    memory                = local.memory
    cpu                   = local.cpu
  }

  // convert to json
  json_container_def = jsonencode(local.container_def)

  // helps prevent user input issues. Easy true of false instead of is it Fargate or FARGATE or FARGTE??
  compat = var.fargate ? "FARGATE" : "EC2"
}

// create an execution role for the task
// for global resources like IAM roles we should append the region to the name. This helps prevent naming conflicts when
// we provision the same application across multiple regions
resource "aws_iam_role" "execution" {
  name               = "${var.name}-${var.environment}-${data.aws_region.current.name}"
  assume_role_policy = data.aws_iam_policy_document.execution.json
}

data "aws_iam_policy_document" "execution" {
  statement {
    actions = ["sts:AssumeRole"]
    effect  = "Allow"
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_cloudwatch_log_group" "main" {
  name = "/ecs/${var.name}-${var.environment}"

  retention_in_days = var.log_retention_in_days
}

resource "aws_iam_role_policy" "execution_policy" {
  name   = "executionPolicy"
  role   = aws_iam_role.execution.id
  policy = data.aws_iam_policy_document.execution_policy.json
}

// default execution policy to be able to pull images from ecr
// and push logs to cloudwatch
data "aws_iam_policy_document" "execution_policy" {
  statement {
    actions = [
      "ecr:GetAuthorizationToken",
    ]
    effect = "Allow"
    resources = [
      "*"
    ]
  }
  statement {
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    effect = "Allow"
    resources = [
      aws_cloudwatch_log_group.main.arn,
      "${aws_cloudwatch_log_group.main.arn}:log-stream:*"
    ]

  }
  // if I provide ecr_repos then i'll create this statement and allow access
  dynamic statement {
    for_each = length(var.ecr_repos) > 0 ? [1] : []
    content {
      actions = [
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
      ]
      effect = "Allow"
      resources = flatten([
        var.ecr_repos
      ])
    }
  }
}

resource "aws_ecs_task_definition" "def" {
  family                = "${var.name}-${var.environment}"
  container_definitions = "[${local.json_container_def}]"

  //task_role_arn      = var.task_role_arn
  task_role_arn      = aws_iam_role.execution.arn // I should create a seperate task role instead of using the execution role
  execution_role_arn = aws_iam_role.execution.arn
  network_mode       = var.network_mode

  cpu                      = local.cpu
  memory                   = local.memory
  requires_compatibilities = [local.compat]
}
