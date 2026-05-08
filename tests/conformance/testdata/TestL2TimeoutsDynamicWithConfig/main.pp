config "timeoutCreate" "string" {
  default = "5m"
}

config "timeoutDelete" "string" {
  default = "30s"
}

resource "withDynamicTimeouts" "test:index/resource:Resource" {
  __logicalName = "with_dynamic_timeouts"
  options {
    customTimeouts = {
      create = invoke("std:index:lookup", {
        map = {
          create = timeoutCreate
          delete = timeoutDelete
        }
        key     = "create"
        default = null
      }).result
      delete = invoke("std:index:lookup", {
        map = {
          create = timeoutCreate
          delete = timeoutDelete
        }
        key     = "delete"
        default = null
      }).result
    }
  }
  value = "x"
}
