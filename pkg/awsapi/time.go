package awsapi

import (
	"regexp"
	"strconv"
	"time"
)

func StrToTime(input string) (time.Time, error) {
	if input == "now" {
		return time.Now(), nil
	}
	daysAgoR := regexp.MustCompile(`-(\d+)d`)
	r := daysAgoR.FindStringSubmatch(input)
	if r != nil && len(r) == 2 {
		daysAgo, err := strconv.Atoi(r[1])
		if err != nil {
			return time.Time{}, nil
		}
		return time.Now().AddDate(0, 0, daysAgo*-1), nil
	}
	return ParseISODatetime(input)
}
