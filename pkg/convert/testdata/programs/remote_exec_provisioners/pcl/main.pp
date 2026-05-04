# Test for remote-exec provisioner conversion. Exercises every connection
# attribute the converter promises to translate, plus all three command-source
# forms: inline, script, and scripts.
config "privateKey" "string" {
}

config "scripts" "list(string)" {
}

resource "remoteExecResource" "simple:index:resource" {
  __logicalName = "remote_exec_resource"
  inputOne      = "hello"
  inputTwo      = true

}
resource "remoteExecResourceProvisioner0" "command:remote:Command" {
  options {
    dependsOn = [remoteExecResource]
    customTimeouts = {
      create = "30s"
      update = "30s"
    }
  }
  connection = {
    host       = "primary.example.com"
    hostKey    = "ssh-ed25519 AAAAtoplevel"
    password   = "ignored-when-private-key-set"
    port       = 2222
    privateKey = privateKey
    user       = "deploy"
    proxy = {
      host     = "bastion.example.com"
      hostKey  = "ssh-ed25519 AAAAbastion"
      password = "bastion-pw"
      port     = 2200
      user     = "jump"
    }
  }
  create = invoke("std:index:join", {
    separator = "\n"
    input     = ["echo first", "echo second"]
  }).result
}
resource "remoteExecResourceProvisioner1Copy" "command:remote:CopyToRemote" {
  options {
    dependsOn = [remoteExecResourceProvisioner0]
    customTimeouts = {
      create = "30s"
      update = "30s"
    }
  }
  connection = {
    host       = "primary.example.com"
    hostKey    = "ssh-ed25519 AAAAtoplevel"
    password   = "ignored-when-private-key-set"
    port       = 2222
    privateKey = privateKey
    user       = "deploy"
    proxy = {
      host     = "bastion.example.com"
      hostKey  = "ssh-ed25519 AAAAbastion"
      password = "bastion-pw"
      port     = 2200
      user     = "jump"
    }
  }
  source     = fileAsset("./script.sh")
  remotePath = "/tmp/remoteExecResourceProvisioner1"
}
resource "remoteExecResourceProvisioner1" "command:remote:Command" {
  options {
    dependsOn = [remoteExecResourceProvisioner1Copy]
    customTimeouts = {
      create = "30s"
      update = "30s"
    }
  }
  connection = {
    host       = "primary.example.com"
    hostKey    = "ssh-ed25519 AAAAtoplevel"
    password   = "ignored-when-private-key-set"
    port       = 2222
    privateKey = privateKey
    user       = "deploy"
    proxy = {
      host     = "bastion.example.com"
      hostKey  = "ssh-ed25519 AAAAbastion"
      password = "bastion-pw"
      port     = 2200
      user     = "jump"
    }
  }
  create = "bash /tmp/remoteExecResourceProvisioner1"
}
resource "remoteExecResourceProvisioner2Copy" "command:remote:CopyToRemote" {
  options {
    range     = scripts
    dependsOn = [remoteExecResourceProvisioner1]
    customTimeouts = {
      create = "30s"
      update = "30s"
    }
  }
  connection = {
    host       = "primary.example.com"
    hostKey    = "ssh-ed25519 AAAAtoplevel"
    password   = "ignored-when-private-key-set"
    port       = 2222
    privateKey = privateKey
    user       = "deploy"
    proxy = {
      host     = "bastion.example.com"
      hostKey  = "ssh-ed25519 AAAAbastion"
      password = "bastion-pw"
      port     = 2200
      user     = "jump"
    }
  }
  source     = fileAsset(range.value)
  remotePath = "/tmp/remoteExecResourceProvisioner2-${range.key}"
}
resource "remoteExecResourceProvisioner2" "command:remote:Command" {
  options {
    dependsOn = remoteExecResourceProvisioner2Copy
    customTimeouts = {
      create = "30s"
      update = "30s"
    }
  }
  connection = {
    host       = "primary.example.com"
    hostKey    = "ssh-ed25519 AAAAtoplevel"
    password   = "ignored-when-private-key-set"
    port       = 2222
    privateKey = privateKey
    user       = "deploy"
    proxy = {
      host     = "bastion.example.com"
      hostKey  = "ssh-ed25519 AAAAbastion"
      password = "bastion-pw"
      port     = 2200
      user     = "jump"
    }
  }
  create = invoke("std:index:join", {
    separator = " && "
    input     = [for k, v in scripts : "bash /tmp/remoteExecResourceProvisioner2-${k}"]
  }).result
}
