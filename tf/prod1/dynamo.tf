provider aws {}

variable "table_name" {}

resource "aws_dynamodb_table" "main" {
    name          = var.table_name
    billing_mode  = "PAY_PER_REQUEST"
    hash_key      = "PK"

    attribute {
        name = "PK"
        type = "S"
    }

    attribute {
        name = "UMS"
        type = "S"
    }

    attribute {
        name = "CRTD"
        type = "N"
    }
    
    ttl {
        attribute_name = "TTL"
        enabled        = true
    }

    stream_enabled   = true
    stream_view_type = "NEW_IMAGE"

    global_secondary_index {
        name               = "UMSIndex"
        hash_key           = "UMS"
        range_key          = "CRTD"
        projection_type    = "INCLUDE"
        non_key_attributes = ["PK", "K"]
    }
    
}

output "stream_arn" {
    value = aws_dynamodb_table.main.stream_arn
}
