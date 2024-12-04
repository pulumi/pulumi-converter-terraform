terraform {
  required_providers {
    random = {
      version = "~> 3.5.1"
      source  = "hashicorp/random"
    }
    different-name-null = {
      source = "hashicorp/null"
    }
  }
  required_version = ">= 1.3.5"
}

output "asdfasdfasdf" {
  value = 123
}
