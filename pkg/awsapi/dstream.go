package awsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/dmitriko/wtctrl/pkg/azr"
	tb "github.com/dmitriko/wtctrl/pkg/telebot"
)

func voiceToText(bot *tb.Bot, voice *tb.Voice) (string, error) {
	if voice.Duration > 59 {
		return "", errors.New(fmt.Sprintf("Record should be less then 60 sec, got %d", voice.Duration))
	}
	file, err := bot.GetFile(&voice.File)
	if err != nil {
		return "", err
	}
	txt, err := azr.DoSpeechRecogn(file)
	if err != nil {
		return "", err
	}
	return txt, nil
}

// Updates Msg.Data with recognized text
func updateMsgData(pk string, table *DTable, txt string) error {
	_, err := table.UpdateItemData(pk, RecognizedTextFieldName, txt)
	return err
}

func replyUser(bot *tb.Bot, tgmsg *tb.Message, txt string) error {
	_, err := bot.Send(tgmsg.Chat, txt)
	return err
}

func runSpeechRecogn(pk string, table *DTable, bot *tb.Bot, upd *tb.Update) {
	txt, err := voiceToText(bot, upd.Message.Voice)
	if err != nil {
		fmt.Printf("ERROR speech recognition: %s", err.Error())
		return
	}
	if txt != "" {
		if err = updateMsgData(pk, table, txt); err != nil {
			fmt.Printf("ERROR updating msg %s with %s", pk, txt)
			return
		}
		if err = replyUser(bot, upd.Message, txt); err != nil {
			fmt.Printf("ERROR responding to user %s", err.Error())
		}
	}
}

func downloadVoice(table *DTable, pk string, voice *tb.Voice, bot *tb.Bot) {
	bucket := os.Getenv("IMG_BUCKET")
	if bucket == "" {
		fmt.Println("IMG_BUCKET evn var is not set")
		return
	}
	sess, _ := session.NewSession()
	file, err := bot.GetFile(&voice.File)
	if err == nil {
		key := fmt.Sprintf("%s.ogg", voice.UniqueID)
		err = storeS3(sess, bucket, key, voice.MIME, file)
		if err == nil {
			createMsgFileVoice(table, pk, voice, key, bucket)
		}
	} else {
		fmt.Println("ERROR", err.Error())
	}
}

func createMsgFileVoice(table *DTable, pk string, voice *tb.Voice, key, bucket string) {
	f, _ := NewMsgFile(pk, FileKindTgVoice, voice.MIME, bucket, key)
	f.Data["duration"] = voice.Duration
	f.Data["size"] = voice.File.FileSize
	err := table.StoreItem(f)
	if err != nil {
		fmt.Println("ERROR storing MsgFile", err.Error())
	}
}

func handleVoiceMsg(pk string, table *DTable, item map[string]events.DynamoDBAttributeValue) {
	bot, err := tb.NewBot(tb.Settings{
		Token:       os.Getenv("TGBOT_SECRET"),
		Synchronous: true,
	})
	if err != nil {
		fmt.Printf("ERROR creating bot %s", err.Error())
		return
	}

	var orig string
	if item["D"].DataType() == events.DataTypeMap {
		data := item["D"].Map()
		orig = data["orig"].String()
	}
	if orig == "" {
		fmt.Println("ERROR orig is empty for msg", pk)
		return
	}

	upd := &tb.Update{}
	err = json.Unmarshal([]byte(orig), upd)
	if err != nil {
		fmt.Printf("ERROR: %s", err.Error())
		return
	}
	if upd.Message == nil || upd.Message.Voice == nil {
		fmt.Printf("ERROR: msg is not well formed %s", orig)
		return
	}
	if upd.Message.Voice.Duration > 59 {
		_ = replyUser(bot, upd.Message, "it's too long")
		return
	}

	runSpeechRecogn(pk, table, bot, upd)
	downloadVoice(table, pk, upd.Message.Voice, bot)
}

func handleTGPhotoMsg(pk string, table *DTable, item map[string]events.DynamoDBAttributeValue) {
	fmt.Println("Handling photo message.")
	bucket := os.Getenv("IMG_BUCKET")
	if bucket == "" {
		fmt.Println("IMG_BUCKET evn var is not set")
	}
	bot, err := tb.NewBot(tb.Settings{
		Token:       os.Getenv("TGBOT_SECRET"),
		Synchronous: true,
	})
	if err != nil {
		fmt.Printf("ERROR creating bot %s", err.Error())
		return
	}
	var upd tb.Update
	if item["D"].DataType() == events.DataTypeMap {
		data := item["D"].Map()
		orig := data["orig"].String()
		err := json.Unmarshal([]byte(orig), &upd)
		if err != nil {
			fmt.Printf("ERROR: %s", err.Error())
			return
		}
		if upd.Message != nil && upd.Message.Photo != nil {
			downloadPics(table, pk, upd.Message.Photo, bot, bucket)
		}
	}
}

func storeS3(sess *session.Session, bucket, key, contentType string, file io.ReadCloser) error {
	defer file.Close()
	uploader := s3manager.NewUploader(sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	return err
}

func createMsgFilePic(table *DTable, pk string, pic *tb.PhotoSize, key, bucket string, i int) {
	kindMap := map[int]string{
		0: FileKindTgThumb,
		1: FileKindTgMediumPic,
		2: FileKindTgBigPic,
	}
	fkind := kindMap[i]
	if fkind == "" {
		fkind = "unknown"
	}
	f, _ := NewMsgFile(pk, fkind, "image/jpeg", bucket, key)
	f.Data["height"] = pic.Height
	f.Data["width"] = pic.Width
	f.Data["size"] = pic.FileSize
	err := table.StoreItem(f)
	if err != nil {
		fmt.Println("ERROR storing MsgFile", err.Error())
	}
}

func downloadPics(table *DTable, pk string, photo *tb.Photo, bot *tb.Bot, bucket string) {
	fmt.Printf("Going to download pics for %#v", photo)
	sess, _ := session.NewSession()

	for i, pic := range photo.Sizes {
		file, err := bot.GetFile(&pic.File)
		if err == nil {
			key := fmt.Sprintf("%s.jpg", pic.UniqueID)
			err = storeS3(sess, bucket, key, "image/jpeg", file)
			if err == nil {
				createMsgFilePic(table, pk, &pic, key, bucket, i)
			}
		} else {
			fmt.Println("ERROR", err.Error())
		}
	}
}

func handleNewMsg(pk string, table *DTable, item map[string]events.DynamoDBAttributeValue) {
	if item["K"].DataType() == events.DataTypeNumber {
		kind, _ := item["K"].Integer()
		if kind == TGVoiceMsgKind {
			handleVoiceMsg(pk, table, item)
			return
		}
		if kind == TGPhotoMsgKind {
			handleTGPhotoMsg(pk, table, item)
			return
		}
	}
}

func notifySubsciptions(table *DTable, pk, eventName string, item map[string]events.DynamoDBAttributeValue) {
	var kind int64
	if item["K"].DataType() == events.DataTypeNumber {
		kind, _ = item["K"].Integer()
	}
	var subs Subscriptions
	if item["UMS"].DataType() == events.DataTypeString {
		ums := item["UMS"].String()
		err := table.FetchItemsWithPrefix(ums, SubscriptionKeyPrefix, &subs)
		if err != nil {
			fmt.Println("ERROR", err.Error())
			return
		}
		for _, s := range subs {
			err = s.SendDBEvent(pk, eventName, ums, kind)
			if err != nil {
				fmt.Println("ERROR", err.Error())
			}
		}
	}
}

func HandleDBEvent(ctx context.Context, table *DTable, e events.DynamoDBEvent) {
	for _, record := range e.Records {
		pk := record.Change.Keys["PK"].String()
		sk := record.Change.Keys["SK"].String()
		fmt.Println("Processing", pk, sk)
		if strings.HasPrefix(pk, MsgKeyPrefix) && strings.HasPrefix(sk, MsgKeyPrefix) {
			notifySubsciptions(table, pk, record.EventName, record.Change.NewImage)
			if record.EventName == "INSERT" {
				handleNewMsg(pk, table, record.Change.NewImage)
			}
		}

	}
}
