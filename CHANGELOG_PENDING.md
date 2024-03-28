### Improvements

- Support the `depends_on` option
- Emit warnings when encountering non-supported lifecycle hooks `create_before_destroy` and `replace_triggered_by` 

### Bug Fixes
 - Convert expressions inside `jsonencode` calls without rewriting object keys to camelCase