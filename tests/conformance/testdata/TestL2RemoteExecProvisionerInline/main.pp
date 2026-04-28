config "conformanceKind" "string" {
}

config "sshHost" "string" {
}

config "sshPort" "number" {
}

config "sshUser" "string" {
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
    user       = sshUser
  }
  create = invoke("std:index:join", {
    separator = "\n"
    input     = ["mkdir -p /tmp/conformance", "printf %s ${example.computedValue} > /tmp/conformance/inline-${conformanceKind}.txt"]
  }).result
}

output "value" {
  value = example.value
}
