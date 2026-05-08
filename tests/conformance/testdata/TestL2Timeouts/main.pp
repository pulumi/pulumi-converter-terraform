resource "withTimeouts" "test:index/resource:Resource" {
  __logicalName = "with_timeouts"
  options {
    customTimeouts = {
      create = "5m"
      update = "10m"
      delete = "30s"
    }
  }
  value = "x"
}
