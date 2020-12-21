package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

func GenerateActionValue(pos int, token int64) string {
	return fmt.Sprintf("%d_%d", pos, token)
}

func ParseActionValue(value string) (pos int, token int64, err error) {
	arr := strings.Split(value, "_")
	if len(arr) != 2 {
		err = errors.New(fmt.Sprintf("Invalid value string '%v'", value))
		return
	}
	pos64, err := strconv.ParseInt(arr[0], 10, strconv.IntSize)
	pos = int(pos64)
	token, err = strconv.ParseInt(arr[1], 10, 64)
	return
}

func readFile(fn string) (content string) {
	bytes, err := ioutil.ReadFile(fn)
	content = string(bytes)
	if err != nil {
		glog.Fatalf("Could not open %v: %v", fn, err)
	}
	return
}

func userToLink(user *slack.User) string {
	return fmt.Sprintf("<slack://user?id=%s&team=%s|%s>", user.ID, user.TeamID, user.Name)
}
