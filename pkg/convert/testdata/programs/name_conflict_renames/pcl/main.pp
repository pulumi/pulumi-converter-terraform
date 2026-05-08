resource "aThingResource" "renames:index/index:resource" {
  __logicalName = "a_thing"
  theResource = {
    theInnerString = "hello"
  }
  theNumber = 1
}

aThing = invoke("renames:index/index:dataSource", {
  theResource = {
    theInnerString = "hello"
  }
  theNumber = 2
})
