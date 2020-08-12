package azr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const SPEECH_ENDPOINT_TMPL = "https://%s.stt.speech.microsoft.com/speech/recognition/conversation/cognitiveservices/v1"

var REGION = "eastus"
var LANG = "ru-RU"

type Response struct {
	RecognitionStatus string
	DisplayText       string
	Offset            int64
	Duration          int64
}

func getUrl() (*url.URL, error) {
	u := fmt.Sprintf(SPEECH_ENDPOINT_TMPL, REGION) + "?language=" + LANG
	return url.Parse(u)
}

func DoSpeechRecogn(body io.ReadCloser) (string, error) {
	if os.Getenv("AZURE_REGION") != "" {
		REGION = os.Getenv("AZURE_REGION")
	}
	key := os.Getenv("AZURE_SPEECH2TEXT_KEY")
	if key == "" {
		return "", errors.New("AZURE_SPEECH2TEXT_KEY is not set")
	}
	u, err := getUrl()
	if err != nil {
		return "", err
	}
	req := &http.Request{
		Method: "POST",
		URL:    u,
		Body:   body,
		Header: map[string][]string{
			"Content-Type":              {"audio/ogg; codecs=opus"},
			"Ocp-Apim-Subscription-Key": {key},
		},
	}
	clnt := &http.Client{
		Timeout: time.Second * 10,
	}
	res, err := clnt.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("Got error response from Azure, status %s", res.Status))
	}
	data, _ := ioutil.ReadAll(res.Body)
	resp := &Response{}
	err = json.Unmarshal(data, resp)
	if err != nil {
		return "", err
	}
	if resp.RecognitionStatus != "Success" {
		return "", errors.New(resp.RecognitionStatus)
	}
	return resp.DisplayText, nil
}
