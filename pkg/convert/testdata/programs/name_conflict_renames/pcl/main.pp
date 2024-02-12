resource "aThingResource" "renames:index/index:resource" {
  __logicalName = "a_thing"
  theNumber     = 1
  theResource = {
    theInnerString = "hello"
  }
}

aThing = invoke("renames:index/index:dataSource", {
  theNumber = 2
  theResource = {
    theInnerString = "hello"
  }
})
