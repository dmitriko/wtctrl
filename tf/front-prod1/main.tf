terraform {
  backend "s3" {
    key    = "frontprod1"
    bucket = "webtectrl-terra"
    region = "us-west-2"
  }
}

provider "aws" {
  alias  = "acm"
  region = "us-east-1"
}

data "aws_route53_zone" "wtctrl" {
  zone_id = "Z03979483BDPVDJTLTZKY"
}

data "aws_iam_role" "api" {
  name = "apiprod1"
}

variable domain {
  default = "wtctrl.com"
}

variable www_bucket_name {
  default = "www-wtctrl-com"
}

variable app_bucket_name {
  default = "app-wtctrl-com"
}

variable app_domain_name {
  default = "app.wtctrl.com"
}

variable "table_name" {}

resource "aws_acm_certificate" "wtctrl_east" {
  provider                  = aws.acm
  domain_name               = var.domain
  subject_alternative_names = ["*.${var.domain}"]
  validation_method         = "DNS"
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "validation_east" {
  zone_id = data.aws_route53_zone.wtctrl.zone_id
  name    = tolist(aws_acm_certificate.wtctrl_east.domain_validation_options)[0].resource_record_name
  type    = tolist(aws_acm_certificate.wtctrl_east.domain_validation_options)[0].resource_record_type
  records = [tolist(aws_acm_certificate.wtctrl_east.domain_validation_options)[0].resource_record_value]
  ttl     = "300"
}

resource "aws_acm_certificate_validation" "wtctrl_east" {
  provider        = aws.acm
  certificate_arn = aws_acm_certificate.wtctrl_east.arn
  validation_record_fqdns = [
    aws_route53_record.validation_east.fqdn
  ]
}


data "archive_file" "webapp" {
  type        = "zip"
  source_dir  = "${path.root}/../../lambda/webapp"
  output_path = "/tmp/webapp.zip"
}

resource "aws_lambda_function" "webapp" {
  function_name    = "webapp_prod1"
  runtime          = "go1.x"
  handler          = "webapp"
  memory_size      = 128
  timeout          = 10
  role             = data.aws_iam_role.api.arn
  filename         = data.archive_file.webapp.output_path
  source_code_hash = data.archive_file.webapp.output_base64sha256
  environment {
    variables = {
      TABLE_NAME = var.table_name
    }
  }
}

resource "aws_apigatewayv2_api" "webapp" {
  name          = "webapp-prod1"
  protocol_type = "HTTP"
  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["*"]
    allow_headers = ["Content-Type"]
    max_age = 1000
    expose_headers = ["Access-Control-Allow-Origin"]
  }
}

resource "aws_apigatewayv2_integration" "webapp" {
  api_id           = aws_apigatewayv2_api.webapp.id
  integration_type = "AWS_PROXY"

  integration_method = "POST"
  integration_uri    = aws_lambda_function.webapp.invoke_arn
}


resource "aws_apigatewayv2_route" "webapp" {
  api_id    = aws_apigatewayv2_api.webapp.id
  target    = "integrations/${aws_apigatewayv2_integration.webapp.id}"
  route_key = "$default"
}

resource "aws_lambda_permission" "webapp" {
  statement_id  = "${aws_lambda_function.webapp.function_name}Lambda"
  function_name = aws_lambda_function.webapp.function_name
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.webapp.execution_arn}/*"
}

resource "aws_apigatewayv2_stage" "webapp" {
  api_id        = aws_apigatewayv2_api.webapp.id
  deployment_id = aws_apigatewayv2_deployment.webapp.id
  name          = "prod1"
}

resource "aws_apigatewayv2_deployment" "webapp" {
  api_id      = aws_apigatewayv2_api.webapp.id
  description = "prod1"
  depends_on = [aws_apigatewayv2_route.webapp]
  triggers = {
    redeployment = sha1(jsonencode(aws_lambda_function.webapp))
  }

  lifecycle {
    create_before_destroy = true
  }
}

data "aws_iam_policy_document" "webapp_policy" {
  statement {
    actions = [
      "s3:GetObject"
    ]
    principals {
      identifiers = ["*"]
      type        = "AWS"
    }
    resources = [
      "arn:aws:s3:::${var.app_bucket_name}/*"
    ]
  }
}

resource "aws_s3_bucket" "webapp_bucket" {
  bucket = var.app_bucket_name
  acl    = "public-read"
  policy = data.aws_iam_policy_document.webapp_policy.json
  website {
    index_document = "index.html"
    error_document = "index.html"
  }
}


resource "aws_cloudfront_distribution" "app_wtctrl_com" {
  origin {
    custom_origin_config {
      http_port              = "80"
      https_port             = "443"
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1", "TLSv1.1", "TLSv1.2"]
    }
    domain_name = replace("${aws_apigatewayv2_api.webapp.api_endpoint}", "https://", "")
    origin_id   = "app_wtctrl_com"
    origin_path = "/${aws_apigatewayv2_stage.webapp.name}"
  }

  enabled             = true
  default_root_object = "index.html"

  default_cache_behavior {
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
    allowed_methods        = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "app_wtctrl_com"
    min_ttl                = 0
    default_ttl            = 0
    max_ttl                = 0

    forwarded_values {
      query_string = true
      cookies {
        forward = "none"
      }
    }
  }

  aliases = [var.app_domain_name]

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn = aws_acm_certificate.wtctrl_east.arn
    ssl_support_method  = "sni-only"
  }
}

resource "aws_route53_record" "webapp" {
  zone_id = data.aws_route53_zone.wtctrl.zone_id
  name    = var.app_domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.app_wtctrl_com.domain_name
    zone_id                = aws_cloudfront_distribution.app_wtctrl_com.hosted_zone_id
    evaluate_target_health = false
  }
}

data "aws_iam_policy_document" "website_policy" {
  statement {
    actions = [
      "s3:GetObject"
    ]
    principals {
      identifiers = ["*"]
      type        = "AWS"
    }
    resources = [
      "arn:aws:s3:::${var.www_bucket_name}/*"
    ]
  }
}

resource "aws_s3_bucket" "website_bucket" {
  bucket = var.www_bucket_name
  acl    = "public-read"
  policy = data.aws_iam_policy_document.website_policy.json
  website {
    index_document = "index.html"
    error_document = "index.html"
  }
}

output "website_endpoint" {
  value = "${aws_s3_bucket.webapp_bucket.bucket_regional_domain_name}"
}

output "app_url" {
  value = "${aws_apigatewayv2_api.webapp.api_endpoint}/prod1"
}

