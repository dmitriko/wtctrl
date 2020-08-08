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
		{
			AttributeName: aws.String("CRTD"),
			AttributeType: aws.String("N"),
		},
		/*	{
			AttributeName: aws.String("TTL"),
			AttributeType: aws.String("N"),
		},*/
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
				ProjectionType:   aws.String("INCLUDE"),
				NonKeyAttributes: []*string{aws.String("PK"), aws.String("K")},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("UMS"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("CRTD"),
					KeyType:       aws.String("RANGE"),
				},
			},
		},
	},
}

var timeToLiveInput = &dynamodb.UpdateTimeToLiveInput{
	TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
		AttributeName: aws.String("TTL"),
		Enabled:       aws.Bool(true),
	},
}
