variable resource_allocation {
  type    = string
  default = "low"
}

variable name {
  type = string
}

variable environment {
  type = string
}

variable fargate {
  type    = bool
  default = true
}

variable network_mode {
  type    = string
  default = "awsvpc"
}

variable log_configuration {
  type = object({
    logDriver = string
    options   = map(string)
    secretOptions = list(object({
      name      = string
      valueFrom = string
    }))
  })
  default = null
}

variable firelens_configuration {
  type = object({
    type    = string
    options = map(string)
  })
  default = null
}

variable cpu {
  type    = number
  default = null
}

variable memory {
  type    = number
  default = null
}

variable port_mappings {
  type = list(object({
    containerPort = number
    hostPort      = number
    protocol      = string
  }))

  default = [
    {
      containerPort = 3000
      hostPort      = 3000
      protocol      = "tcp"
    }
  ]
}

variable container_image {
  type = string
}

variable task_role_arn {
  type    = string
  default = ""
}

variable ecr_repos {
  type    = list(string)
  default = []
}

variable log_retention_in_days {
  type    = number
  default = 1
}
