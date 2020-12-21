package service

import (
	"github.com/slack-go/slack"

	"time"
)

type EnqueueRequest struct {
	User     *slack.User
	Metadata string
}

type EnqueueResponse struct {
	User      *slack.User
	Metadata  string
	Ok        bool
	Pos       int
	Timestamp time.Time
}

type DequeueRequest struct {
	Place int
	Token int64
}

type DequeueResponse struct {
	User      *slack.User
	Metadata  string
	Timestamp time.Time
	Token     int64
}

type ListRequest struct {
}

type ListResponse struct {
	Users    []*slack.User
	Metadata []string
	Times    []time.Time
	Token    int64
}

type RemoveRequest struct {
	Pos   int
	Token int64
}

type RemoveResponse struct {
	Err   error
	Token int64
}

type MoveRequest struct {
	Pos   int
	NPos  int
	Token int64
}

type MoveResponse struct {
	Ok    bool
	Token int64
}
