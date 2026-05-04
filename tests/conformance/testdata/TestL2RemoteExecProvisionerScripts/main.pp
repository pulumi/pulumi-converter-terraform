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
scripts = ["./a.sh", "./b.sh"]

resource "example" "test:index/resource:Resource" {
  value = "hello"
}
resource "exampleProvisioner0Copy" "command:remote:CopyToRemote" {
  options {
    range     = scripts
    dependsOn = [example]
  }
  connection = {
    host       = sshHost
    port       = sshPort
    privateKey = sshPrivateKey
    user       = sshUser
  }
  source     = fileAsset(range.value)
  remotePath = "/tmp/exampleProvisioner0-${range.key}"
}
resource "exampleProvisioner0" "command:remote:Command" {
  options {
    dependsOn = exampleProvisioner0Copy
  }
  connection = {
    host       = sshHost
    port       = sshPort
    privateKey = sshPrivateKey
    user       = sshUser
  }
  create = invoke("std:index:join", {
    separator = " && "
    input     = [for k, v in scripts : "bash /tmp/exampleProvisioner0-${k}"]
  }).result
}

output "value" {
  value = example.value
}
