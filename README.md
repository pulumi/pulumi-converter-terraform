# Pulumi Terraform Converter Plugin

[![Build Status](https://github.com/pulumi/pulumi-converter-terraform/actions/workflows/publish-snapshot.yaml/badge.svg)](https://github.com/pulumi/pulumi-converter-terraform/actions/workflows/publish-snapshot.yaml)

Convert Terraform projects to Pulumi programs written in your favourite languages.

## Goals

The goal of `pulumi-converter-terraform` is to help users efficiently convert Terraform-managed infrastructure
into Pulumi stacks. It translates HCL configuration into Pulumi programs. And Terraform state files into
Pulumi import files.

## Building and Installation

To use `pulumi-converter-terraform` you can build the tool from source or you can use one of the [binary
releases](https://github.com/pulumi/tf2pulumi/releases) hosted on GitHub.

### Install
`pulumi-converter-terraform` can be installed using Pulumi's plugin system:
```console
pulumi plugin install converter terraform
```

### Building

`pulumi-converter-terraform` uses [Go modules](https://github.com/golang/go/wiki/Modules) to manage dependencies. If you want to develop `pulumi-converter-terraform` itself, you'll need to have [Go](https://golang.org/) installed in order to build.
Once this prerequisite is installed, run the following to build the `pulumi-converter-terraform` binary and install it into `$GOPATH/bin`:

```console
$ make install
```

Go should automatically handle pulling the dependencies for you.

If `$GOPATH/bin` is not on your path, you may want to move the `pulumi-converter-terraform` binary from `$GOPATH/bin`
into a directory that is on your path.

## Usage

In order to use `pulumi-converter-terraform` to convert a Terraform project to Pulumi and then deploy it,
you'll first need to install the [Pulumi CLI](https://pulumi.io/quickstart/install.html). Once the
Pulumi CLI has been installed, navigate to the same directory as the Terraform project you'd like to
import and create a new Pulumi stack in your favourite language:

```console
// For a Go project
$ pulumi new go -f

// For a TypeScript project
$ pulumi new typescript -f

// For a Python project
$ pulumi new python -f

// For a C# project
$ pulumi new csharp -f

// For a Java project
$ pulumi new java -f

// For a YAML project
$ pulumi new yaml -f
```

Then run `pulumi convert` which will write a file in the directory that
contains the Pulumi project you just created:

```console
// For a Go project
$ pulumi convert --from terraform --language go

// For a TypeScript project
$ pulumi convert --from terraform --language typescript

// For a Python project
$ pulumi convert --from terraform --language python

// For a C# project
$ pulumi convert --from terraform --language csharp

// For a Java project
$ pulumi convert --from terraform --language java

// For a YAML project
$ pulumi convert --from terraform --language yaml
```

If `pulumi-converter-terraform` complains about missing Terraform resource plugins, install those plugins as
per the instructions in the error message and re-run the command above.

This will generate a Pulumi program that when run with `pulumi update` will deploy the infrastructure
originally described by the Terraform project. Note that if your infrastructure references files or
directories with paths relative to the location of the Terraform project, you will most likely need to update
these paths such that they are relative to the generated file.

## Adopting Resource From TFState

If you would like to adopt resources from an existing `.tfstate` file under management of a Pulumi stack, you
can use `pulumi import`. Again you will need to first install the [Pulumi
CLI](https://pulumi.io/quickstart/install.html). Once the Pulumi CLI has been installed, navigate to the same
directory of your Pulumi project you'd like to import to, probably the directory you created via `pulumi
convert` above.

Then run `pulumi import` which will translate the Terraform state file, and import those resources into Pulumi:

```console
$ pulumi import --from terraform ./terraform.tfstate
```
Once imported, the existing resources in your cloud provider can now be managed by Pulumi going forward. See
the [Adopting Existing Cloud Resources into
Pulumi](https://www.pulumi.com/blog/adopting-existing-cloud-resources-into-pulumi/) blog post for more details
on importing existing resources.

## Limitations

While the majority of Terraform constructs are already supported, there are some gaps.
- Various built-in interpolation functions. Calls to unimplemented functions will throw at
  runtime.
- `self` and `terraform` variable references.