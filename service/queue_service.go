package service

import (
	"github.com/matthewlang/slack-queue/persister"
	"github.com/matthewlang/slack-queue/queue"

	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"time"
)

type UserLookup interface {
	Lookup(id string) (user *slack.User, err error)
}

type UserLookupImpl struct {
	api *slack.Client
}

func (ul *UserLookupImpl) Lookup(id string) (user *slack.User, err error) {
	user, err = ul.api.GetUserInfo(id)
	return
}

type QueueService struct {
	q *queue.VersionedQueue
	u UserLookup
}

func InMemoryTS(api *slack.Client) *QueueService {
	u := &UserLookupImpl{api}
	return TS(u, nil)
}

func PersistentTS(api *slack.Client, persist persister.Persister) *QueueService {
	u := &UserLookupImpl{api}
	return TS(u, persist)
}

func TS(u UserLookup, persist persister.Persister) *QueueService {
	s := &QueueService{}
	s.q = queue.VQ(persist)
	s.u = u
	return s
}

func (s *QueueService) Enqueue(req *EnqueueRequest, resp *EnqueueResponse) (err error) {
	user := req.User
	resp.User = user
	now := time.Now()
	pos, seq, e := s.q.Put(queue.Element{Id: user.ID, Metadata: req.Metadata, QTime: now})
	resp.Pos = pos
	if e != nil {
		ae, ok := e.(queue.AlreadyExistsError)
		if !ok {
			// Unknown error
			glog.Errorf("Unknown error on Put: %v", err)
			err = e
		} else {
			glog.Infof("User (%v) %v already in queue at time %v (v %v)", user.ID, user.Name, ae.Timestamp, seq)
			resp.Ok = false
			resp.Timestamp = ae.Timestamp
			return
		}
	}
	resp.Ok = true
	resp.Timestamp = now
	return
}

func (s *QueueService) Dequeue(req *DequeueRequest, resp *DequeueResponse) (err error) {
	var el queue.Element
	var seq int64
	var e error
	if req.Place == 0 {
		el, seq, e = s.q.TakeFront()
	} else {
		el, seq, e = s.q.Take(req.Place, req.Token)
	}
	if e != nil {
		resp.Token = seq
		resp.User = nil
		err = nil
		glog.Infof("Error taking %d from queue: %v", req.Place, e)
		return
	}
	glog.Infof("Dequeueing %v (v %v)", el.Id, seq)
	user, err := s.u.Lookup(el.Id)
	if err != nil {
		glog.Errorf("Dequeued user %v but could not get user info, requeueing (v %v): %v", el.Id, seq, err)
		s.q.Put(el)
		return
	}
	resp.User = user
	resp.Metadata = el.Metadata
	resp.Timestamp = el.QTime
	return
}

func (s *QueueService) List(req *ListRequest, resp *ListResponse) (err error) {
	lst, seq := s.q.List()
	resp.Token = seq
	for _, el := range lst {
		user, err := s.u.Lookup(el.Id)
		if err != nil {
			glog.Errorf("Failed to lookup user %v in queue (v %v): %v", el.Id, seq, err)
			// Return error, but also return users that were looked up.
		} else {
			resp.Users = append(resp.Users, user)
			resp.Times = append(resp.Times, el.QTime)
			resp.Metadata = append(resp.Metadata, el.Metadata)
		}
	}
	return
}

func (s *QueueService) Remove(req *RemoveRequest, resp *RemoveResponse) (err error) {
	seq, e := s.q.Remove(req.Pos, req.Token)
	resp.Token = seq
	if e != nil {
		ae, ok := e.(queue.VersionError)
		if !ok {
			glog.Errorf("Unknown error on remove at %d with token %d: %+v:", req.Pos, req.Token, ae)
			err = ae
			return
		}
	}
	glog.Infof("Remove at %d with token %d, error: %+v", req.Pos, req.Token, err)
	resp.Err = e
	return
}

func (s *QueueService) Recover() {
	s.q.Recover()
	return
}
