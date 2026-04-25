config "conformanceKind" "string" {
}

config "sshHost" "string" {
}

config "sshPort" "number" {
}

config "sshPrivateKey" "string" {
}

resource "example" "test:index/resource:Resource" {
  value = "hello"
}
resource "exampleProvisioner0" "command:remote:Command" {
  options {
    dependsOn = [example]
  }
  connection = {
    host       = sshHost
    port       = sshPort
    privateKey = sshPrivateKey
    user       = conformanceKind
  }
  create = join("\n", ["echo ${example.computedValue}"])
}

output "value" {
  value = example.value
}
