config "timeouts" "map(string)" {
  default = {
    create = "5m"
    delete = "30s"
  }
}

resource "withDynamicTimeouts" "test:index/resource:Resource" {
  __logicalName = "with_dynamic_timeouts"
  options {
    customTimeouts = singleOrNone([for entry in [timeouts] : {
      create = invoke("std:index:lookup", {
        map     = entry
        key     = "create"
        default = null
      }).result
      delete = invoke("std:index:lookup", {
        map     = entry
        key     = "delete"
        default = null
      }).result
    }])
  }
  value = "x"
}
