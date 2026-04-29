### Improvements

- Convert `file` provisioners to `command:remote:CopyToRemote`. The `source` form
  emits `fileAsset(...)` or `fileArchive(...)` when the path can be resolved at convert
  time, falling back to `try(fileAsset(p), fileArchive(p))` for non-literal paths so
  Pulumi picks the correct asset shape at runtime. The `content` form emits
  `stringAsset(...)`.

### Bug Fixes
