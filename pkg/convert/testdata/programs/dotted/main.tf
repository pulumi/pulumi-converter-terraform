resource "complex_resource" "example" {
    innerObject = {
        noDots          = true
        ".dotted"       = true
        "dot.in.middle" = true
        "dotAtEnd."     = true
        "..."           = true
    }
}
