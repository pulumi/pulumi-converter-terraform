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
resource "exampleProvisioner0Copy" "command:remote:CopyToRemote" {
  options {
    dependsOn = [example]
  }
  connection = {
    host       = sshHost
    port       = sshPort
    privateKey = sshPrivateKey
    user       = sshUser
  }
  source     = fileAsset("./run.sh")
  remotePath = "/tmp/exampleProvisioner0"
}
resource "exampleProvisioner0" "command:remote:Command" {
  options {
    dependsOn = [exampleProvisioner0Copy]
  }
  connection = {
    host       = sshHost
    port       = sshPort
    privateKey = sshPrivateKey
    user       = sshUser
  }
  create = "bash /tmp/exampleProvisioner0"
}

output "value" {
  value = example.value
}
