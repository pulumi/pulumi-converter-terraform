/**
 * These data sources are used to simplify the inputs required by the module as well as ensure security best practices are
 * being followed.
 */

// pulls the current region based on the credential session being used
data "aws_region" "current" {}

// pulls the current aws account based on the credential session  being used
data "aws_caller_identity" "current" {}


resource "aws_security_group" "main" {
  name        = "${var.name}-${var.environment}-sg"
  description = "security group"
  vpc_id      = var.vpc_id
}

resource "aws_security_group_rule" "egress" {
  type        = "egress"
  from_port   = 0
  to_port     = 65535
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = aws_security_group.main.id
}

// we are only allowing ingress from other security groups (i.e. security group attached to the load balancer)
resource "aws_security_group_rule" "ingress" {
  count                    = length(var.ingress_security_groups)
  type                     = "ingress"
  from_port                = var.container_port
  to_port                  = var.container_port
  protocol                 = "tcp"
  source_security_group_id = var.ingress_security_groups[count.index]

  security_group_id = aws_security_group.main.id
}

resource "aws_ecs_service" "main" {
  // we ignore changes to the desired_count because we are ideally using autoscaling
  // we also ignore changes to the task_definition because we need to make updates to the task
  // definition outside of terraform (i.e. during deployments)
  lifecycle {
    ignore_changes = [desired_count, task_definition]
  }
  name                               = "${var.name}-${var.environment}"
  cluster                            = var.cluster_arn
  task_definition                    = var.task_definition_arn
  desired_count                      = var.desired_count
  deployment_maximum_percent         = var.deployment_maximum_percent
  deployment_minimum_healthy_percent = var.deployment_minimum_healthy_percent

  // dynamically attach to a load balancer based on whether a target group is provided
  dynamic "load_balancer" {
    iterator = t
    for_each = var.target_group_arns
    content {
      target_group_arn = t.value
      container_name   = var.name
      container_port   = var.container_port
    }
  }

  // if this is a fargate service then provide the network configuration otherwise don't
  dynamic "network_configuration" {
    for_each = var.fargate ? [1] : []
    content {
      subnets          = var.subnet_ids
      security_groups  = [aws_security_group.main.id]
      assign_public_ip = false
    }
  }

  // If we provide service discovery information then create otherwise don't
  dynamic "service_registries" {
    for_each = var.service_registry_arns
    content {
      registry_arn   = each.value
      container_port = var.container_port
      container_name = var.name
    }
  }

  // setup the capacity provider strategy based on user input
  dynamic "capacity_provider_strategy" {
    iterator = s
    for_each = var.capacity_provider_strategies
    content {
      capacity_provider = s.key
      weight            = s.value["weight"]
      base              = s.value["base"]
    }
  }
}
