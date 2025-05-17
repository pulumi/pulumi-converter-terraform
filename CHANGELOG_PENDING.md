### Improvements

- Fix overlapping dynamic scopes shadowing names, making accessing the shadowed `entry` impossible.
- Add path.root and path.cwd to the converter to pcl intrinsics projectRoot and cwd respectively.
- Allow mapper package hint when upstream package was not found
- Support converting remote modules to resources parameterized by terraform-module
- Support converting local modules to resources parameterized by terraform-module

### Bug Fixes

- Fix heredoc parsing when inside a typle cons expr
