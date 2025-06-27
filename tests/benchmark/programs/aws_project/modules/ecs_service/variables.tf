variable vpc_id {
  type = string
}

variable subnet_ids {
  type = list(string)
}

variable target_group_arns {
  type    = list(string)
  default = []
}

variable environment {
  type = string
}

variable service_registry_arns {
  type    = list(string)
  default = []
}

variable capacity_provider_strategies {
  type = map(object({
    weight = number
    base   = any
  }))
  default = {
    FARGATE = {
      weight = 100
      base   = null
    }
  }
}

variable fargate {
  type    = bool
  default = true
}

variable cluster_arn {
  type = string
}

variable task_definition_arn {
  type = string
}

variable desired_count {
  type    = number
  default = 2
}

variable container_port {
  type    = number
  default = 3000
}

variable name {
  type = string
}

variable project {
  type = string
}


variable ingress_security_groups {
  type    = list(string)
  default = []
}

variable deployment_maximum_percent {
  type    = number
  default = 200
}

variable deployment_minimum_healthy_percent {
  type    = number
  default = 100
}

variable lb_security_group {
  type    = string
  default = ""
}
