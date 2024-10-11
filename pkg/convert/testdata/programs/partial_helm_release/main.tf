resource "helm_release" "example" {
  name  = "redis"
  chart = "redis"
  cleanup_on_fail = true
  atomic = true
  create_namespace = true
  dependency_update = true
  description = "A Helm chart for Kubernetes"
  devel = true
  disable_crd_hooks = true
  disable_openapi_validation = true
  disable_webhooks = true
  force_update = true
  keyring = "./keyring.gpg"
  lint = true
  max_history = 5
  pass_credentials = true
  postrender {
    binary_path = "echo"
    args = [ "foo", "bar" ]
  }
  recreate_pods = true
  render_subchart_notes = true
  replace = true
  repository = "https://charts.bitnami.com/bitnami"
  repository_ca_file = "./ca.pem"
  repository_cert_file = "./cert.pem"
  repository_key_file = "./key.pem"
  repository_password = "password"
  repository_username = "username"
  reset_values = true
  reuse_values = true
  set {
    name  = "set_1"
    value = "value1"
  }
  set_list {
    name  = "set_list_1"
    value = "value1"
  }
  set_sensitive {
    name  = "set_sensitive_1"
    value = "value1"
    type = "string"
  }
  skip_crds = true
  timeout = 60
  values  = [<<-EOT
    fullnameOverride: foo
    EOT
  ]
  version = "10.7.16"
  wait = true
  wait_for_jobs = true
}