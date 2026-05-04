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
resource "exampleProvisioner0" "command:remote:CopyToRemote" {
  options {
    dependsOn = [example]
  }
  connection = {
    host       = sshHost
    port       = sshPort
    privateKey = sshPrivateKey
    user       = sshUser
  }
  source     = stringAsset("override ${conformanceKind}\n")
  remotePath = "/tmp/conformance/override-${conformanceKind}.txt"
}

output "value" {
  value = example.value
}
