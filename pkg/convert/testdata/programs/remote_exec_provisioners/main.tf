# Test for remote-exec provisioner conversion. Exercises every connection
# attribute the converter promises to translate, plus all three command-source
# forms: inline, script, and scripts.

variable "private_key" {
  type = string
}

variable "scripts" {
  type = list(string)
}

resource "simple_resource" "remote_exec_resource" {
  input_one = "hello"
  input_two = true

  connection {
    host             = "primary.example.com"
    port             = 2222
    user             = "deploy"
    password         = "ignored-when-private-key-set"
    private_key      = var.private_key
    host_key         = "ssh-ed25519 AAAAtoplevel"
    bastion_host     = "bastion.example.com"
    bastion_port     = 2200
    bastion_user     = "jump"
    bastion_password = "bastion-pw"
    bastion_host_key = "ssh-ed25519 AAAAbastion"
    timeout          = "30s"
    agent            = false
  }

  provisioner "remote-exec" {
    inline = ["echo first", "echo second"]
  }

  provisioner "remote-exec" {
    script = "./script.sh"
  }

  provisioner "remote-exec" {
    scripts = var.scripts
  }
}
