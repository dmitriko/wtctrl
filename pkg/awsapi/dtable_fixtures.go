package awsapi

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var createTableInput = &dynamodb.CreateTableInput{
	BillingMode: aws.String("PAY_PER_REQUEST"),
	AttributeDefinitions: []*dynamodb.AttributeDefinition{
		{
			AttributeName: aws.String("PK"),
			AttributeType: aws.String("S"),
		},
		{
			AttributeName: aws.String("UMS"),
			AttributeType: aws.String("S"),
		},
	},
	KeySchema: []*dynamodb.KeySchemaElement{
		{
			AttributeName: aws.String("PK"),
			KeyType:       aws.String("HASH"),
		},
	},
	GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
		{
			IndexName: aws.String("UMSIndex"),
			Projection: &dynamodb.Projection{
				ProjectionType: aws.String("KEYS_ONLY"),
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("UMS"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("PK"),
					KeyType:       aws.String("RANGE"),
				},
			},
		},
	},
}
