terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      my_tag = "my_value"
    }
  }
}

resource "random_string" "bucket_name" {
  length  = 8
  special = false
  upper   = false
}

resource "aws_s3_bucket" "example" {
  bucket = random_string.bucket_name.result
  tags = {
    Name = "My bucket"
  }
}

output "name" {
  value = aws_s3_bucket.example.bucket
}