resource "example" "complex:index/index:resource" {
  innerMapObject = {
    noDots          = true
    ".dotted"       = true
    "dot.in.middle" = true
    "dotAtEnd."     = true
    "..."           = true
  }
}
