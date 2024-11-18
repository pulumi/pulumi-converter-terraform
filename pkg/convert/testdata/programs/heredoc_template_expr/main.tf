locals {
  a_list = ["a", "b", "c"]
  join_template_expr = <<EOT
%{for v in local.a_list~}
${v}
%{endfor~}
EOT
}
