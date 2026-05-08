config "timeouts" "object({create=string, delete=string})" {
  default = null
}

resource "withDynamicTimeouts" "test:index/resource:Resource" {
  __logicalName = "with_dynamic_timeouts"
  options {
    customTimeouts = {
      create = try(invoke("std:index:lookup", {
        map     = singleOrNone(timeouts != null ? [timeouts] : [])
        key     = "create"
        default = null
      }).result, null)
      delete = try(invoke("std:index:lookup", {
        map     = singleOrNone(timeouts != null ? [timeouts] : [])
        key     = "delete"
        default = null
      }).result, null)
    }
  }
  value = "x"
}
