package "null" {
  baseProviderName    = "terraform-provider"
  baseProviderVersion = "0.3.0"
  parameterization = {
    name    = "hashicorp/null"
    value   = "eyJyZW1vdGUiOnsidXJsIjoicmVnaXN0cnkub3BlbnRvZnUub3JnL2hhc2hpY29ycC9udWxsIiwidmVyc2lvbiI6IjMuMi4zIn19"
    version = ""
  }
}

package "random" {
  baseProviderName    = "terraform-provider"
  baseProviderVersion = "0.3.0"
  parameterization = {
    name    = "hashicorp/random"
    value   = "eyJyZW1vdGUiOnsidXJsIjoicmVnaXN0cnkub3BlbnRvZnUub3JnL2hhc2hpY29ycC9yYW5kb20iLCJ2ZXJzaW9uIjoiMy41LjEifX0="
    version = "~> 3.5.1"
  }
}


output "asdfasdfasdf" {
  value = 123
}
