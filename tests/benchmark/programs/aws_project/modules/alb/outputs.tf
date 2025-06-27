output lb_arn {
  value = aws_lb.main.arn
}

output lb_url {
  value = aws_lb.main.dns_name
}

output tg_arn {
  value = aws_lb_target_group.main.arn
}

output security_group_id {
  value = aws_security_group.main.id
}
