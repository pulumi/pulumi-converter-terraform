// Copyright 2016-2024, Pulumi Corporation.
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

package shim

import (
	"context"
	"fmt"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/hcl/v2"
	tfregaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-svchost/disco"
	tfaddr "github.com/opentofu/registry-address"
	"github.com/pulumi/terraform/pkg/configs"
	"github.com/pulumi/terraform/pkg/getproviders"
)

// A simple shim to access internals of opentofu for resolving versions.

// FindTfPackageVersion finds an appropriate version of an opentofu/tf package.
func FindTfPackageVersion(prov *configs.RequiredProvider) (versions.Version, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}
	p, err := tfaddr.ParseProviderSource(prov.Source)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid terraform provider name",
			Detail:   fmt.Sprintf("invalid provider name: %s", err.Error()),
			Subject:  &prov.DeclRange,
		})
	}
	ver, err := getproviders.ParseVersionConstraints(prov.Requirement.Required.String())
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid terraform provider version",
			Detail:   fmt.Sprintf("invalid provider version: %s", err.Error()),
			Subject:  &prov.DeclRange,
		})
	}
	source := getproviders.NewRegistrySource(disco.New())

	ptf := tfregaddr.Provider{
		Type:      p.Type,
		Namespace: p.Namespace,
		Hostname:  p.Hostname,
	}
	availableVersions, warnings, err := source.AvailableVersions(context.TODO(), ptf)
	for _, warning := range warnings {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "provider version warning",
			Detail:   warning,
			Subject:  &prov.DeclRange,
		})
	}
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "provider version error",
			Detail:   err.Error(),
			Subject:  &prov.DeclRange,
		})
	}
	desiredVersion := availableVersions.NewestInSet(versions.MeetingConstraints(ver))
	if desiredVersion == versions.Unspecified {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "could not resolve provider version",
			Detail:   fmt.Sprintf("Could not resolve a version from %s: %s", p, ver),
			Subject:  &prov.DeclRange,
		})
	}
	return desiredVersion, diags
}
