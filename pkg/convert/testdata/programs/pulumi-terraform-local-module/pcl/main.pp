package "example" {
  baseProviderName    = "terraform-module"
  baseProviderVersion = "0.1.4"
  // encoded parameterization values:
  // module: ../pulumi-terraform-local-module/example
  // packageName: example
  parameterization {
    name    = "example"
    version = "0.0.1"
    value   = "eyJtb2R1bGUiOiIuLi9wdWx1bWktdGVycmFmb3JtLWxvY2FsLW1vZHVsZS9leGFtcGxlIiwicGFja2FnZU5hbWUiOiJleGFtcGxlIn0="
  }
}

resource "exampleModule" "example:index:Module" {
  name = "John"
}

output "name" {
  value = exampleModule.name
}
