resource "example" "kubernetes:helm.sh/v3:Release" {
  name                     = "redis"
  chart                    = "redis"
  cleanupOnFail            = true
  atomic                   = true
  createNamespace          = true
  dependencyUpdate         = true
  description              = "A Helm chart for Kubernetes"
  devel                    = true
  disableCRDHooks          = true
  disableOpenapiValidation = true
  disableWebhooks          = true
  forceUpdate              = true
  keyring                  = "./keyring.gpg"
  lint                     = true
  maxHistory               = 5
  recreatePods             = true
  renderSubchartNotes      = true
  replace                  = true
  repositoryOpts = {
    repo     = "https://charts.bitnami.com/bitnami"
    caFile   = "./ca.pem"
    certFile = "./cert.pem"
    keyFile  = "./key.pem"
    password = "password"
    username = "username"
  }
  resetValues = true
  reuseValues = true
  values = {
    "set_1"           = "value1"
    "set_list_1"      = "value1"
    "set_sensitive_1" = secret("value1")
  }
  skipCrds    = true
  timeout     = 60
  version     = "10.7.16"
  waitForJobs = true
}
