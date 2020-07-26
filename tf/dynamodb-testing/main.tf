provider aws {
    endpoints {
        dynamodb = "http://127.0.0.1:8000"
    }
}

resource "aws_dynamodb_table" "webtectrlv1" {
    name          = "main"
    billing_mode  = "PAY_PER_REQUEST"
    hash_key      = "PK"

    attribute {
        name = "PK"
        type = "S"
    }

    attribute {
        name = "USM"
        type = "S"
    }

    global_secondary_index {
        name            = "main_usm"
        hash_key        = "USM"
        range_key       = "PK"
        projection_type = "ALL"
    }
    
}


