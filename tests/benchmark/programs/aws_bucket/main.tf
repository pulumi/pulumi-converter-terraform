resource "random_string" "bucket_name" {
  length  = 8
  special = false
  upper   = false
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      "my_tag" = "my_value"
    }
  }
}

resource "aws_s3_bucket" "example" {
  bucket = random_string.bucket_name.result
  tags = {
    Name = "My bucket"
  }
}

resource "aws_s3_bucket_object" "object" {
  bucket = aws_s3_bucket.example.bucket
  key    = "index.html"
  source = "index.html"
}

output "url" {
  value = "s3://${aws_s3_bucket.example.bucket}/${aws_s3_bucket_object.object.key}"
}

output "name" {
  value = aws_s3_bucket.example.bucket
}
