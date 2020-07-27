package awsapi

import (
	"testing"
	"time"
)

func TestStrToTime(t *testing.T) {
	tm, err := StrToTime("now")
	if err != nil {
		t.Error(err)
	}
	dur := time.Now().Sub(tm)
	if int(dur.Seconds()) != 0 {
		t.Errorf("Could no parse now, difference %f sec", dur.Seconds())
	}
	tm, err = StrToTime("-2d")
	if err != nil {
		t.Error(err)
	}
	dur = time.Now().Sub(tm)
	if int(dur.Hours()) != 48 {
		t.Errorf("Could not parse -2d, difference %d hours", int(dur.Hours()))
	}
	tm, err = StrToTime("2020-07-27T19:58:51.535597")
	if err != nil {
		t.Error(err)
	}
	if tm.Year() != 2020 && tm.Month() != 7 && tm.Day() != 27 && tm.Hour() != 19 && tm.Minute() != 51 {
		t.Errorf("Got wrong date %+v for %s", tm, "2020-07-27T19:58:51.535597")
	}
}
