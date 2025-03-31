// Copyright 2024-2025, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package convert

import "slices"

// isTerraformProvider returns true if and only if the given provider name is *not* one in the "Pulumi universe". This
// means that this function should return true for any provider that must be dynamically bridged. Note that the given
// provider name must be a *Pulumi package name*, not (for instance) a Terraform provider name.
func isTerraformProvider(name string) bool {
	return !slices.Contains(pulumiSupportedProviders, name)
}

// pulumiRenamedProviderNames is a map whose keys are Terraform provider names and whose values are the corresponding
// (managed) Pulumi provider names, in the cases where they differ.
var pulumiRenamedProviderNames = map[string]string{
	"azurerm":  "azure",
	"bigip":    "f5bigip",
	"google":   "gcp",
	"template": "terraform-template",
}

var pulumiSupportedProviders = []string{
	"acme",
	"aiven",
	"akamai",
	"alicloud",
	"aquasec",
	"archive",
	"artifactory",
	"astra",
	"auth0",
	"aws",
	"aws-eksa",
	"azure",
	"azuread",
	"azuredevops",
	"buildkite",
	"cloudamqp",
	"cloudflare",
	"cloudinit",
	"cloudngfwaws",
	"concourse",
	"configcat",
	"confluentcloud",
	"consul",
	"databricks",
	"datadog",
	"dbtcloud",
	"digitalocean",
	"dnsimple",
	"docker",
	"doppler",
	"ec",
	"exoscale",
	"external",
	"f5bigip",
	"fastly",
	"gandi",
	"gcp",
	"github",
	"github-credentials",
	"gitlab",
	"googleworkspace",
	"harbor",
	"harness",
	"hcloud",
	"hcp",
	"heroku",
	"http",
	"ise",
	"junipermist",
	"kafka",
	"keycloak",
	"kong",
	"kubernetes",
	"linode",
	"mailgun",
	"matchbox",
	"meraki",
	"minio",
	"mongodbatlas",
	"mssql",
	"mysql",
	"newrelic",
	"ngrok",
	"nomad",
	"ns1",
	"null",
	"oci",
	"okta",
	"openstack",
	"opsgenie",
	"pagerduty",
	"postgresql",
	"purrl",
	"rabbitmq",
	"rancher2",
	"random",
	"scm",
	"sdwan",
	"sentry",
	"signalfx",
	"slack",
	"snowflake",
	"splunk",
	"spotinst",
	"statuscake",
	"sumologic",
	"tailscale",
	"tf-provider-boilerplate",
	"time",
	"tls",
	"unifi",
	"vault",
	"venafi",
	"vra",
	"vsphere",
	"wavefront",
	"xyz",
	"zitadel",
}
