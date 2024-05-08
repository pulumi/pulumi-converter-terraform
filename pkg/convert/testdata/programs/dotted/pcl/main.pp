resource "example" "complex:index/index:resource" {
  innerObject = {
    noDots          = true
    ".dotted"       = true
    "dot.in.middle" = true
    "dotAtEnd."     = true
    "..."           = true
  }
}
