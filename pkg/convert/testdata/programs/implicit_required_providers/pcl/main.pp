package "tfe" {
  baseProviderName    = "terraform-provider"
  baseProviderVersion = "0.8.1"
  parameterization {
    version = "0.70.0"
    name    = "tfe"
    value   = "eyJyZW1vdGUiOnsidXJsIjoicmVnaXN0cnkudGVycmFmb3JtLmlvL2hhc2hpY29ycC90ZmUiLCJ2ZXJzaW9uIjoiMC43MC4wIn19"
  }
}

// A program that uses the tfe provider without explicitly declaring it in required_providers.
// Since this is _not_ a Pulumi provider, the converter should emit a parameterized package block
// with the provider name and a version constraint of "~> <latest version>".
resource "test-organization" "tfe:index/organization:Organization" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "test-agent-pool" "tfe:index/agentPool:AgentPool" {
  name               = "my-agent-pool-name"
  organization       = test-organization.name
  organizationScoped = true
}
