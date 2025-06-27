resource "aws_acm_certificate" "cert" {
  domain_name       = "*.mb-dev.example.com"
  validation_method = "DNS"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}


resource "aws_subnet" "main" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}

module "sample-app_task_def" {
  source = "./modules/ecs_task"

  resource_allocation = "low"
  container_image     = "corymhall/hello-world-go:latest"
  name                = "sample-app"
  environment         = local.env
}

module "sample-app_s3_iam" {
  source = "./modules/s3-iam"

  role         = module.sample-app_task_def.execution_role
  access_level = ["read"]
  bucket_name  = "my-sample-bucket"
}

module "sample-app_alb" {
  source = "./modules/alb"

  project         = local.project
  name            = "sample-app"
  environment     = local.env
  certificate_arn = aws_acm_certificate.cert.arn
  vpc_id          = aws_vpc.main.id
  subnet_ids      = [aws_subnet.main.id]
}

module "sample-app_ecs_service" {
  source                  = "./modules/ecs_service"
  cluster_arn             = aws_ecs_cluster.main.arn
  environment             = local.env
  task_definition_arn     = module.sample-app_task_def.arn
  name                    = "sample-app"
  project                 = local.project
  target_group_arns       = [module.sample-app_alb.tg_arn]
  ingress_security_groups = [module.sample-app_alb.security_group_id]
  vpc_id                  = aws_vpc.main.id
  subnet_ids              = [aws_subnet.main.id]
}
