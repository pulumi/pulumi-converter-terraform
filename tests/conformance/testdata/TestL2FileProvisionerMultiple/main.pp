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
  source     = stringAsset("first ${conformanceKind}\n")
  remotePath = "/tmp/conformance/multi-first-${conformanceKind}.txt"
}
resource "exampleProvisioner1" "command:remote:CopyToRemote" {
  options {
    dependsOn = [exampleProvisioner0]
  }
  connection = {
    host       = sshHost
    port       = sshPort
    privateKey = sshPrivateKey
    user       = sshUser
  }
  source     = stringAsset("second ${conformanceKind}\n")
  remotePath = "/tmp/conformance/multi-second-${conformanceKind}.txt"
}

output "value" {
  value = example.value
}
