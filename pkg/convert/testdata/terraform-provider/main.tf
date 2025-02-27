terraform {
  required_version = ">= 1.1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    planetscale = {
      source  = "planetscale/planetscale"
      version = "~> 0.1.0"
    }
  }
}

provider "google" {
  project = var.gcp_project
  region  = var.gcp_region
  zone    = var.gcp_zone
}

provider "planetscale" {
  service_token      = var.planetscale_service_token
  service_token_name = "planetscaletoken"
}

resource "planetscale_database" "db" {
  name         = "pulumi-convert-db"
  organization = var.planetscale_org
}
