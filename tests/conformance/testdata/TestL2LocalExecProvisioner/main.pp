config "outputPath" "string" {
}

resource "example" "test:index/resource:Resource" {
  value = "hello"
}
resource "exampleProvisioner0" "command:local:Command" {
  options {
    dependsOn = [example]
  }
  create = "printf %s \"${example.computedValue}\" > \"${outputPath}/$PULUMI_CONVERTER_CONFORMANCE_KIND.txt\""
}

output "value" {
  value = example.value
}
