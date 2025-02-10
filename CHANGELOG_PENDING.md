### Improvements

- Allow generating of tobool invocation.
- Enable jsondecode which is already in pulumi-std
- Enable lookup which is already in pulumi-std
- Enable merge which is already in pulumi-std
- Enable flatten which is already in pulumi-std
- Implement `coalesce` through the `pulumi-std` invoke of the same name

### Bug Fixes

 - Require v0.6.0 of the `terraform-provider` instead of v0.5.4 because the latter is no longer available.
