data "aws_iam_policy_document" "read_only" {
  statement {
    effect = "Allow"
    actions = [
      "s3:GetObject"
    ]
    resources = [
      "arn:aws:s3:::${var.bucket_name}/*"
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "s3:ListBucket"
    ]
    resources = [
      "arn:aws:s3:::${var.bucket_name}"
    ]
  }
}

data "aws_iam_policy_document" "write_only" {
  statement {
    effect = "Allow"
    actions = [
      "s3:PutObject"
    ]
    resources = [
      "arn:aws:s3:::${var.bucket_name}/*"
    ]
  }
}
resource "aws_iam_role_policy" "read_bucket_access" {
  count  = contains(var.access_level, "read") ? 1 : 0
  name   = "${var.bucket_name}-read-access"
  role   = "${var.role}"
  policy = data.aws_iam_policy_document.read_only.json
}

resource "aws_iam_role_policy" "write_bucket_access" {
  count  = contains(var.access_level, "write") ? 1 : 0
  name   = "${var.bucket_name}-write-access"
  role   = "${var.role}"
  policy = data.aws_iam_policy_document.write_only.json
}
