resource "main" "maxItemsOne:index/index:resource" {
  innerResource = singleOrNone([for entry in [true] : {
    someInput = true
  }])
}
