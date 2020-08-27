package awsapi

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

const TGPicDBEvent = `
{
   "Records":[
      {
         "awsRegion":"us-west-2",
         "dynamodb":{
            "ApproximateCreationDateTime":1598515792,
            "Keys":{
               "PK":{
                  "S":"msg#1gfq3BfAYEAWL8SEzUBc8R8CW8f"
               },
               "SK":{
                  "S":"msg#1gfq3BfAYEAWL8SEzUBc8R8CW8f"
               }
            },
            "NewImage":{
               "A":{
                  "S":"user#1gKk4ltY2UuMc9poYZaCPFArCLY"
               },
               "CRTD":{
                  "N":"1598515792"
               },
               "Ch":{
                  "S":"bot#wtctrlbot#tg"
               },
               "D":{
                  "M":{
                     "orig":{
                        "S":"{\"update_id\":45554225,\n\"message\":{\"message_id\":94,\"from\":{\"id\":123456789,\"is_bot\":false,\"first_name\":\"D\",\"last_name\":\"K\",\"language_code\":\"en\"},\"chat\":{\"id\":123456789,\"first_name\":\"D\",\"last_name\":\"K\",\"type\":\"private\"},\"date\":1598515790,\"photo\":[{\"file_id\":\"AgACAgIAAxkBAANeX0dqThacr_o3xYFITetMDYToYT8AAkStMRtA6TlKQYi8uqvAo2oNCJ-WLgADAQADAgADbQADf9AAAhsE\",\"file_unique_id\":\"AQADDQifli4AA3_QAAI\",\"file_size\":12591,\"width\":180,\"height\":320},{\"file_id\":\"AgACAgIAAxkBAANeX0dqThacr_o3xYFITetMDYToYT8AAkStMRtA6TlKQYi8uqvAo2oNCJ-WLgADAQADAgADeAADgNAAAhsE\",\"file_unique_id\":\"AQADDQifli4AA4DQAAI\",\"file_size\":58243,\"width\":450,\"height\":800},{\"file_id\":\"AgACAgIAAxkBAANeX0dqThacr_o3xYFITetMDYToYT8AAkStMRtA6TlKQYi8uqvAo2oNCJ-WLgADAQADAgADeQADgdAAAhsE\",\"file_unique_id\":\"AQADDQifli4AA4HQAAI\",\"file_size\":119353,\"width\":720,\"height\":1280}]}}"
                     }
                  }
               },
               "K":{
                  "N":"3"
               },
               "PK":{
                  "S":"msg#1gfq3BfAYEAWL8SEzUBc8R8CW8f"
               },
               "SK":{
                  "S":"msg#1gfq3BfAYEAWL8SEzUBc8R8CW8f"
               },
               "UMS":{
                  "S":"user#1gKk4ltY2UuMc9poYZaCPFArCLY#0"
               }
            },
            "SequenceNumber":"36741300000000000483457944",
            "SizeBytes":1069,
            "StreamViewType":"NEW_IMAGE"
         },
         "eventID":"d780d18ce6b08087861462fec66da618",
         "eventName":"INSERT",
         "eventSource":"aws:dynamodb",
         "eventVersion":"1.1",
         "eventSourceARN":"arn:aws:dynamodb:us-west-2:978051452011:table/wtctrlv1/stream/2020-08-19T12:29:22.347"
      }
   ]
}
`
const TGDocDBEvent = `
{
   "Records":[
      {
         "awsRegion":"us-west-2",
         "dynamodb":{
            "ApproximateCreationDateTime":1598518144,
            "Keys":{
               "PK":{
                  "S":"msg#1gfuolrj5pjTMx0oHW9LbBC8zE6"
               },
               "SK":{
                  "S":"msg#1gfuolrj5pjTMx0oHW9LbBC8zE6"
               }
            },
            "NewImage":{
               "A":{
                  "S":"user#1gKk4ltY2UuMc9poYZaCPFArCLY"
               },
               "CRTD":{
                  "N":"1598518144"
               },
               "Ch":{
                  "S":"bot#wtctrlbot#tg"
               },
               "D":{
                  "M":{
                     "orig":{
                        "S":"{\"update_id\":45554229,\n\"message\":{\"message_id\":99,\"from\":{\"id\":123456789,\"is_bot\":false,\"first_name\":\"D\",\"last_name\":\"K\",\"language_code\":\"en\"},\"chat\":{\"id\":123456789,\"first_name\":\"D\",\"last_name\":\"K\",\"type\":\"private\"},\"date\":1598518143,\"document\":{\"file_name\":\"IMGM5160.jpg\",\"mime_type\":\"image/jpeg\",\"thumb\":{\"file_id\":\"AAMCAgADGQEAA2NfR3N-gnsinaTtBJ0n6dK0V1ZgpAACbAcAAkDpOUpEHyHttkiwvvMtdpcuAAMBAAdtAANwAgACGwQ\",\"file_unique_id\":\"AQAD8y12ly4AA3ACAAI\",\"file_size\":17494,\"width\":320,\"height\":213},\"file_id\":\"BQACAgIAAxkBAANjX0dzfoJ7Ip2k7QSdJ-nStFdWYKQAAmwHAAJA6TlKRB8h7bZIsL4bBA\",\"file_unique_id\":\"AgADbAcAAkDpOUo\",\"file_size\":6066767}}}"
                     }
                  }
               },
               "K":{
                  "N":"4"
               },
               "PK":{
                  "S":"msg#1gfuolrj5pjTMx0oHW9LbBC8zE6"
               },
               "SK":{
                  "S":"msg#1gfuolrj5pjTMx0oHW9LbBC8zE6"
               },
               "UMS":{
                  "S":"user#1gKk4ltY2UuMc9poYZaCPFArCLY#0"
               }
            },
            "SequenceNumber":"36964900000000024852890109",
            "SizeBytes":877,
            "StreamViewType":"NEW_IMAGE"
         },
         "eventID":"cc1cc9247793664e81486b5bab24eca8",
         "eventName":"INSERT",
         "eventSource":"aws:dynamodb",
         "eventVersion":"1.1",
         "eventSourceARN":"arn:aws:dynamodb:us-west-2:978051452011:table/wtctrlv1/stream/2020-08-19T12:29:22.347"
      }
   ]
}`

const TGVoiceDBEvent = `
{
   "Records":[
      {
         "awsRegion":"us-west-2",
         "dynamodb":{
            "ApproximateCreationDateTime":1598516817,
            "Keys":{
               "PK":{
                  "S":"msg#1gfs7wcpPIidtx7QqVjeiYzbZz1"
               },
               "SK":{
                  "S":"msg#1gfs7wcpPIidtx7QqVjeiYzbZz1"
               }
            },
            "NewImage":{
               "A":{
                  "S":"user#1gKk4ltY2UuMc9poYZaCPFArCLY"
               },
               "CRTD":{
                  "N":"1598516817"
               },
               "Ch":{
                  "S":"bot#wtctrlbot#tg"
               },
               "D":{
                  "M":{
                     "orig":{
                        "S":"{\"update_id\":45554226,\n\"message\":{\"message_id\":95,\"from\":{\"id\":123456789,\"is_bot\":false,\"first_name\":\"D\",\"last_name\":\"K\",\"language_code\":\"en\"},\"chat\":{\"id\":123456789,\"first_name\":\"D\",\"last_name\":\"K\",\"type\":\"private\"},\"date\":1598516815,\"voice\":{\"duration\":1,\"mime_type\":\"audio/ogg\",\"file_id\":\"AwACAgIAAxkBAANfX0duT7WVEu0ugtgblUp6OgLLwFwAAmgHAAJA6TlKZxLsai1Iqv0bBA\",\"file_unique_id\":\"AgADaAcAAkDpOUo\",\"file_size\":6528}}}"
                     }
                  }
               },
               "K":{
                  "N":"2"
               },
               "PK":{
                  "S":"msg#1gfs7wcpPIidtx7QqVjeiYzbZz1"
               },
               "SK":{
                  "S":"msg#1gfs7wcpPIidtx7QqVjeiYzbZz1"
               },
               "UMS":{
                  "S":"user#1gKk4ltY2UuMc9poYZaCPFArCLY#0"
               }
            },
            "SequenceNumber":"36964700000000024852592515",
            "SizeBytes":660,
            "StreamViewType":"NEW_IMAGE"
         },
         "eventID":"50798f90ea38b01cd1362649f68bbb66",
         "eventName":"INSERT",
         "eventSource":"aws:dynamodb",
         "eventVersion":"1.1",
         "eventSourceARN":"arn:aws:dynamodb:us-west-2:978051452011:table/wtctrlv1/stream/2020-08-19T12:29:22.347"
      }
   ]
}`

func GetDBEvent(data string) (*events.DynamoDBEvent, error) {
	var e events.DynamoDBEvent
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func TestDBeventUnmarhal(t *testing.T) {
	_, err := GetDBEvent(TGPicDBEvent)
	assert.Nil(t, err)
	_, err = GetDBEvent(TGVoiceDBEvent)
	assert.Nil(t, err)
}
