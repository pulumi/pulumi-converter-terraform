CHANGELOG
=========

## 1.2.4

### Bug Fixes
 - Handle attributes with complex indexer parts by falling back to camelCasing the attribute name if the name cannot be determined.

## 1.2.3

### Bug Fixes
 - Handle terraform state files that contain `check_results` field by removing it before parsing the rest of the state file.

## 1.2.2

### Improvements

 - Sanitize resource names during terraform state conversion to handle special characters and make them valid identifiers.


## 1.2.1

- Support converting remote and local modules to resources parameterized by
  [terraform-module](https://github.com/pulumi/pulumi-terraform-module) provider. To use this feature, annotate the
  sources with a special comment: `// @pulumi-terraform-module <pulumi-package-name>`

  For example: `// @pulumi-terraform-module vpc`

## 1.2.0

### Improvements

- Fix overlapping dynamic scopes shadowing names, making accessing the shadowed `entry` impossible.
- Add path.root and path.cwd to the converter to pcl intrinsics projectRoot and cwd respectively.
- Allow mapper package hint when upstream package was not found
- Support converting remote modules to resources parameterized by terraform-module

### Bug Fixes

- Fix heredoc parsing when inside a typle cons expr

## 1.1.0

### Improvements

- Allow generating of tobool invocation.
- Enable jsondecode which is already in pulumi-std
- Enable lookup which is already in pulumi-std
- Enable merge which is already in pulumi-std
- Enable flatten which is already in pulumi-std
- Implement `coalesce` through the `pulumi-std` invoke of the same name
- Implement `compact` through the `pulumi-std` invoke of the same name
- Implement `coalescelist` through the `pulumi-std` invoke of the same name
- Implement `distinct` through the `pulumi-std` invoke of the same name
- Implement `format` through the `pulumi-std` invoke of the same name
- Implement `keys` through the `pulumi-std` invoke of the same name
- Implement `setintersection` through the `pulumi-std` invoke of the same name
- Implement `alltrue` through the `pulumi-std` invoke of the same name
- Implement `anytrue` through the `pulumi-std` invoke of the same name
- Implement `contains` through the `pulumi-std` invoke of the same name
- Implement `chunklist` through the `pulumi-std` invoke of the same name
- Implement `slice` through the `pulumi-std` invoke of the same name
- Implement `regex(all)` through the `pulumi-std` invokes of the same name
- Implement `toset` through the `pulumi-std` invoke of the same name
- Implement `cidrsubnets` through the `pulumi-std` invoke of the same name
- Implement `formatlist` through the `pulumi-std` invoke of the same name

### Bug Fixes

- Fix the order of arguments to `substr`
- Fix conversion in the presence of dynamically bridged Terraform providers

## 1.0.22

- Bump generated provider-terraform version

## 1.0.21

### Improvements

- Change inferred resource names to pascal case
- Add parameterization block to "package" blocks
- Add generation of pcl "package" blocks

## 1.0.20

### Improvements

- Add EOT (heredoc) style string delimiter handling.
- Add template join expression to convert expression
- Add references to issues for missing functions in output
- Add code generation rename workarounds for pcl keywords

### Bug Fixes

- Fix errors being encountered but not reported to the user

- Fix using a module multiple times via different constraints

- Fix conversion of object blocks

- Fix conversion of object attributes

## 1.0.18

### Improvements

- Support custom translation of appautoscaling ids

## 1.0.17

### Improvements

- Support the standard `tolist` function translation by the rewrite rule `tolist(x) ==> x`
- Don't attempt to rename object keys if the object contains any non-identifiers

## 1.0.16

### Improvements

- Support the `depends_on` option
- Emit warnings when encountering non-supported lifecycle hooks `create_before_destroy` and `replace_triggered_by`

### Bug Fixes

- Convert expressions inside `jsonencode` calls without rewriting object keys to camelCase

## 1.0.15

### Improvements

 - Ensure block and attribute iteration is consistent.

## 1.0.14

### Improvements

 - Sort generated properties by the position of their keys to get deterministic output.

## 1.0.13

### Improvements

  - Better conflict resolution for conflicting source names. They'll now use the objects type name to help build a unique name.

## 1.0.12

### Improvements

  - Support the `replace` function.

  - Don't rename map keys.

### Bug Fixes

 - Fix order of parameters for the `parseint` function
