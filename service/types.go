package service

import (
	"github.com/slack-go/slack"

	"time"
)

type EnqueueRequest struct {
	User *slack.User
}

type EnqueueResponse struct {
	User      *slack.User
	Ok        bool
	Pos       int
	Timestamp time.Time
}

type DequeueRequest struct {
	Place int
}

type DequeueResponse struct {
	User      *slack.User
	Timestamp time.Time
}

type ListRequest struct {
}

type ListResponse struct {
	Users []*slack.User
	Times []time.Time
	Token int64
}
