output "plain_string" {
    value = "hello world"
}

output "escaped_string" {
    value = "\"\thello\nworld\r1\\2\""
}

output "unicode_escape_string" {
    value = "\u1111"
}

output "unicode_string" {
    value = "Ǝ"
}