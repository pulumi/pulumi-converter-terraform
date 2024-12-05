resource "complex_resource" "example" {
    innerMapObject = {
        noDots          = true
        ".dotted"       = true
        "dot.in.middle" = true
        "dotAtEnd."     = true
        "..."           = true
    }
}
