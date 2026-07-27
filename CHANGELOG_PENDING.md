### Improvements

- Convert the Terraform `can` builtin to the PCL `can` intrinsic, now that the
  PCL code generator supports it.
  [#295](https://github.com/pulumi/pulumi-converter-terraform/pull/295)

- Convert the `nonsensitive` TF function to the PCL `unsecret` intrinsic.
  [#139](https://github.com/pulumi/pulumi-converter-terraform/issues/139)

- Convert `file` provisioners to `command:remote:CopyToRemote`. The `source` form
  emits `fileAsset(...)` or `fileArchive(...)` when the path can be resolved at convert
  time, falling back to `try(fileAsset(p), fileArchive(p))` for non-literal paths so
  Pulumi picks the correct asset shape at runtime. The `content` form emits
  `stringAsset(...)`.

### Bug Fixes

- Convert resource `timeouts` blocks to a `customTimeouts` resource option instead of
  silently dropping them or emitting them as a resource attribute. Both static
  `timeouts {}` blocks and `dynamic "timeouts" {}` blocks are now lifted to the
  `customTimeouts` option on the generated resource.
  [#104](https://github.com/pulumi/pulumi-converter-terraform/issues/104)
  [#184](https://github.com/pulumi/pulumi-converter-terraform/issues/184)

- Stop emitting deprecated package block labels in generated PCL, using the
  `baseProviderName` attribute instead.  [#405](https://github.com/pulumi/pulumi-converter-terraform/pull/405)

- Convert `for_each = toset(<list>)` to a string-keyed PCL for-object and lower the
  `<cond> ? [<x>] : []` gate idiom on `dynamic` blocks to a conditional list with
  `each.value`/`each.key` inlined.
  [#228](https://github.com/pulumi/pulumi-converter-terraform/issues/228)
