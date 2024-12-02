terraform {
  required_providers {
    null = {
      source = "hashicorp/null"
    }
    random = {
      version = "~> 3.5.1"
      source  = "hashicorp/random"
    }
  }
  required_version = ">= 1.3.5"
}

output "asdfasdfasdf" {
  value = 123
}
