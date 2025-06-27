/**
 * These data sources are used to simplify the inputs required by the module as well as ensure security best practices are
 * being followed.
 */

// pulls the current region based on the credential session being used
data "aws_region" "current" {}
// pulls the current aws account based on the credential session  being used
data "aws_caller_identity" "current" {}


// locals are like variables, but they allow us to perform some functions on the values
locals {
  subnet_tag_value = var.internal ? "app" : "dmz"

  // if the user passes in less elastic ips than there are subnets make sure that we only use that
  // number of subnets
  subnets_slice = slice(tolist(var.subnet_ids), 0, length(var.elastic_ips))

  subnets = [
    for s in var.subnet_ids : s if length(var.elastic_ips) == 0
  ]

  subnet_mapping = [
    for pair in setproduct(local.subnets_slice, var.elastic_ips) : {
      subnet_id     = pair[0].key
      allocation_id = pair[1].key
    }
    if length(var.elastic_ips) != 0
  ]

  protocol = var.load_balancer_type == "application" ? "HTTP" : "TCP"

  health_check_type = var.target_type == "lambda" ? "lambda" : var.load_balancer_type

  health_check = {
    application = var.application_lb_health_check
    network     = var.network_lb_health_check
    lambda      = {}
  }

}

resource "aws_security_group" "main" {
  name        = "${var.name}-${var.environment}-alb-sg"
  description = "alb security group"
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

resource "aws_security_group_rule" "ingress" {
  type        = "ingress"
  from_port   = 443
  to_port     = 443
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"] // you would probably change this based on internal vs external load balancer

  security_group_id = aws_security_group.main.id
}

resource "aws_lb" "main" {
  name                             = "${var.name}-${var.environment}"
  internal                         = var.internal
  load_balancer_type               = var.load_balancer_type
  enable_cross_zone_load_balancing = true                                                                        // some values we hardcode because we want this to always be set, we don't want the user to be able to override
  security_groups                  = var.load_balancer_type == "application" ? [aws_security_group.main.id] : [] // network lbs don't support security groups
  subnets                          = local.subnets

  // dynamically creating access_logs based on whether the user provides the info
  dynamic "access_logs" {
    iterator = a
    for_each = var.access_logs
    content {
      bucket  = a.value["bucket"]
      prefix  = a.value["prefix"]
      enabled = a.value["enabled"]
    }
  }

  // if the user provides elastic ip information then create this block
  dynamic "subnet_mapping" {
    iterator = s
    for_each = {
      for subnet in local.subnet_mapping : "${subnet.subnet_id}.${subnet.allocation_id}" => subnet
    }
    content {
      subnet_id     = s.value.subnet_id
      allocation_id = s.value.allocation_id
    }
  }
}

resource "aws_lb_target_group" "main" {
  depends_on           = [aws_lb.main]
  name                 = "${var.name}-${var.environment}-tg"
  target_type          = var.target_type
  port                 = var.application_port
  protocol             = local.protocol
  vpc_id               = var.vpc_id
  deregistration_delay = var.deregistration_delay

  // dynamically create this health check based on the type of target and type of lb
  dynamic "health_check" {
    iterator = h
    for_each = local.health_check[local.health_check_type]
    content {
      enabled             = h.value.enabled
      interval            = h.value.interval
      port                = h.value.port
      protocol            = h.value.protocol
      path                = lookup(h.value, "path", null)
      healthy_threshold   = h.value.healthy_threshold
      unhealthy_threshold = h.value.unhealthy_threshold
      matcher             = lookup(h.value, "matcher", null)
    }
  }
}

resource "aws_lb_listener" "main" {
  load_balancer_arn = aws_lb.main.arn
  port              = 443
  protocol          = var.load_balancer_type == "application" ? "HTTPS" : "TLS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = var.certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }
}
