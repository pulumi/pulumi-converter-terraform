// A program that uses the tfe provider without explicitly declaring it in required_providers.
// Since this is _not_ a Pulumi provider, the converter should emit a parameterized package block
// with the provider name and a version constraint of "~> <latest version>".

resource "tfe_organization" "test-organization" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "tfe_agent_pool" "test-agent-pool" {
  name         = "my-agent-pool-name"
  organization = tfe_organization.test-organization.name
  organization_scoped = true
}