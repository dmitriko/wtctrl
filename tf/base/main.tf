terraform {
  backend "s3" {
    key = "base"
  }
}

resource "aws_s3_bucket" "deploy" {
    bucket = "webtectrl-deploy"
    acl    = "private"
}

