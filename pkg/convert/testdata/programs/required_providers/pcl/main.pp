package "null" {
  baseProviderName    = "terraform-provider"
  baseProviderVersion = "0.3.0"
  parameterization "name" "version" "value" {
    version = "3.2.3"
    name    = "hashicorp/null"
    value   = "eyJyZW1vdGUiOnsidXJsIjoiaGFzaGljb3JwL251bGwiLCJ2ZXJzaW9uIjoiMy4yLjMifX0="
  }
}

package "random" {
  baseProviderName    = "terraform-provider"
  baseProviderVersion = "0.3.0"
  parameterization "name" "version" "value" {
    version = "3.5.1"
    name    = "hashicorp/random"
    value   = "eyJyZW1vdGUiOnsidXJsIjoiaGFzaGljb3JwL3JhbmRvbSIsInZlcnNpb24iOiIzLjUuMSJ9fQ=="
  }
}


output "asdfasdfasdf" {
  value = 123
}
