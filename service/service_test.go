package service

import (
	"github.com/slack-go/slack"

	"testing"
)

type MockUserLookup struct {
	i         int
	responses []struct {
		User *slack.User
		Err  error
	}
}

func (ml *MockUserLookup) Lookup(id string) (user *slack.User, err error) {
	r := ml.responses[ml.i]
	user = r.User
	err = r.Err
	ml.i++
	return
}

func TestEnqueue(t *testing.T) {
	mul := &MockUserLookup{}
	ts := TS(mul, nil)

	req := &EnqueueRequest{}
	resp := &EnqueueResponse{}

	req.User = &slack.User{}
	req.User.ID = "user123"

	err := ts.Enqueue(req, resp)

	if err != nil {
		t.Fatalf("No expected failure on first put: %v", err)
	}

	if !resp.Ok {
		t.Fatalf("Expected success.")
	}
}

func TestMultiEnqueue(t *testing.T) {
	mul := &MockUserLookup{}
	ts := TS(mul, nil)

	req := &EnqueueRequest{}
	resp := &EnqueueResponse{}

	req.User = &slack.User{}
	req.User.ID = "user123"

	err := ts.Enqueue(req, resp)

	if err != nil {
		t.Fatalf("No expected failure on put: %v", err)
	}

	if !resp.Ok {
		t.Fatalf("Expected success.")
	}

	req.User.ID = "user456"

	err = ts.Enqueue(req, resp)

	if err != nil {
		t.Fatalf("No expected failure on put: %v", err)
	}

	if !resp.Ok {
		t.Fatalf("Expected success.")
	}

	req.User.ID = "user123"

	err = ts.Enqueue(req, resp)

	if err != nil {
		t.Fatalf("No expected failure on put: %v", err)
	}

	if resp.Ok {
		t.Fatalf("Expected failure.")
	}
}
