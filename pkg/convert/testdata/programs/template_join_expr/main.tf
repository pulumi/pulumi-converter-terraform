locals {
  a_list = ["a", "b", "c"]
  join_template_expr = "%{for v in local.a_list~}${v}%{endfor~}"
  if_template_expr = "%{if true~}true%{else~}false%{endif~}"
}
