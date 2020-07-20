variable "deploy_bucket" {
    default = "webtectrl-deploy"
}

variable "viber_webhook_s3_key" {
    default = "lambda/viber-webhook.zip"
}

variable "lambda_binary_name" {
    default = "main"
}

variable "viber_lambda_binary_hash" {
    default = "0SHjD+E5jGTjKWLZMbdqQCzm+vYOk4VBTfNSZd9UJVg="
}
