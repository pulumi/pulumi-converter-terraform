terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    cloudflare = {
      source = "cloudflare/cloudflare"
    }
    random = {
      source = "hashicorp/random"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

provider "cloudflare" {}

locals {
  site_domain = "pulumi-cloudflare-demo.com"
}

resource "aws_s3_bucket" "site" {
  bucket = local.site_domain
}

resource "aws_s3_bucket_public_access_block" "site" {
  bucket = aws_s3_bucket.site.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_website_configuration" "site" {
  bucket = aws_s3_bucket.site.id

  index_document {
    suffix = "index.html"
  }

  error_document {
    key = "error.html"
  }
}

resource "aws_s3_bucket_ownership_controls" "site" {
  bucket = aws_s3_bucket.site.id
  rule {
    object_ownership = "BucketOwnerPreferred"
  }
}

resource "aws_s3_bucket_acl" "site" {
  bucket = aws_s3_bucket.site.id

  acl = "public-read"
  depends_on = [
    aws_s3_bucket_ownership_controls.site,
    aws_s3_bucket_public_access_block.site
  ]
}

resource "aws_s3_bucket_policy" "site" {
  bucket = aws_s3_bucket.site.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadGetObject"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource = [
          aws_s3_bucket.site.arn,
          "${aws_s3_bucket.site.arn}/*",
        ]
      },
    ]
  })

  depends_on = [
    aws_s3_bucket_public_access_block.site
  ]
}

data "cloudflare_zones" "domain" {
  name = local.site_domain
}

resource "cloudflare_dns_record" "site_cname" {
  zone_id = data.cloudflare_zones.domain.result[0].id
  name    = local.site_domain
  content = aws_s3_bucket_website_configuration.site.website_endpoint
  type    = "CNAME"

  ttl     = 1
  proxied = true
}

resource "cloudflare_dns_record" "www" {
  zone_id = data.cloudflare_zones.domain.result[0].id
  name    = "www"
  content = local.site_domain
  type    = "CNAME"

  ttl     = 1
  proxied = true
}
