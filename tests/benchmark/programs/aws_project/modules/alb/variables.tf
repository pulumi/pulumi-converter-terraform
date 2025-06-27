// TODO: these all need to be udpated with descriptions
variable network_lb_health_check {
  type = list(object({
    enabled             = bool
    interval            = number
    port                = string
    protocol            = string
    healthy_threshold   = number
    unhealthy_threshold = number
  }))
  default = [{
    enabled             = true
    interval            = 30
    port                = "traffic-port"
    protocol            = "TCP"
    healthy_threshold   = 3
    unhealthy_threshold = 3
  }]
}

variable application_lb_health_check {
  type = list(object({
    enabled             = bool
    interval            = number
    port                = string
    path                = string
    protocol            = string
    healthy_threshold   = number
    unhealthy_threshold = number
    matcher             = string
  }))
  default = [{
    enabled             = true
    interval            = 10
    port                = "traffic-port"
    path                = "/ping"
    protocol            = "HTTP"
    healthy_threshold   = 2
    unhealthy_threshold = 3
    matcher             = "200,299"
  }]
}

variable project {
  type = string
}

variable name {
  type = string
}

variable environment {
  type = string
}

variable elastic_ips {
  type    = list(string)
  default = []
}

variable load_balancer_type {
  type    = string
  default = "application"
}

variable internal {
  type    = bool
  default = false
}

variable access_logs {
  type = list(object({
    bucket  = string
    prefix  = any
    enabled = bool
  }))
  default = []
}

variable application_port {
  type    = number
  default = 3000
}

variable target_type {
  type    = string
  default = "ip"
}

variable deregistration_delay {
  type    = number
  default = 10
}

variable certificate_arn {
  type = string
}

variable vpc_id {
  type = string
}

variable subnet_ids {
  type = list(string)
}