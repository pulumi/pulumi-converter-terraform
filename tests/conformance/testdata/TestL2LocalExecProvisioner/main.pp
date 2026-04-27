config "conformanceKind" "string" {
}

config "outputPath" "string" {
}

resource "example" "test:index/resource:Resource" {
  value = "hello"
}
resource "exampleProvisioner0" "command:local:Command" {
  options {
    dependsOn = [example]
  }
  create = "printf %s \"${example.computedValue}\" > \"${outputPath}/${conformanceKind}.txt\""
}

output "value" {
  value = example.value
}
