output "exposed" {
  value = unsecret(secret("hello"))
}
