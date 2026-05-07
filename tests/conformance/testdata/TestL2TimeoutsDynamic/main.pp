config "timeouts" "map(string)" {
  default = {
    create = "5m"
    delete = "30s"
  }
}

resource "withDynamicTimeouts" "test:index/resource:Resource" {
  __logicalName = "with_dynamic_timeouts"
  options {
    customTimeouts = {
      create = invoke("std:index:lookup", {
        map     = timeouts
        key     = "create"
        default = null
      }).result
      delete = invoke("std:index:lookup", {
        map     = timeouts
        key     = "delete"
        default = null
      }).result
    }
  }
  value = "x"
}
