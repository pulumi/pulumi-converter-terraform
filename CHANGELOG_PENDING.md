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
- Fix overlapping dynamic scopes shadowing names, making accessing the shadowed `entry` impossible.
- Add path.root and path.cwd to the converter to pcl intrinsics projectRoot and cwd respectively.
- Allow mapper package hint when upstream package was not found

### Bug Fixes

- Fix the order of arguments to `substr`
- Fix conversion in the presence of dynamically bridged Terraform providers
