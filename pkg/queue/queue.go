package queue

import (
	"github.com/golang/glog"
	"github.com/matthewlang/slack-queue/pkg/persister"

	"errors"
	"fmt"
	"time"
)

type Element struct {
	Id       string    `json:"Id"`
	Metadata string    `json:"Metadata"`
	QTime    time.Time `json:"QTime"`
}

type Queue interface {
	Put(el Element) (pos int, err error)
	TakeFront() (el Element, err error)
	Take(i int) (el Element, err error)
	Get(i int) (el Element, err error)
	Remove(i int) (err error)
	Move(i int, npos int) (err error)
	Find(id string) (pos int, err error)
	List() (els []Element)
	Size() int

	Persist()
	Recover()
}

type queueImpl struct {
	els     []Element
	persist persister.Persister
}

type QueueState struct {
	Elements []Element `json:"Elements"`
}

func MakeQueue(persist persister.Persister) Queue {
	q := &queueImpl{}
	q.persist = persist
	return q
}

type AlreadyExistsError struct {
	Id        string
	Timestamp time.Time
}

func (ae AlreadyExistsError) Error() string {
	return fmt.Sprintf("%v already exists at time %v", ae.Id, ae.Timestamp)
}

func (q *queueImpl) Recover() {
	if q.persist == nil {
		glog.Infof("In-memory -- nothing to recover.")
		return
	}
	state := QueueState{}
	q.persist.Read(&state)
	q.els = state.Elements
	glog.Infof("Recovered queue: %v", q.els)
}

func (q *queueImpl) Persist() {
	if q.persist == nil {
		return
	}
	err := q.persist.Write(QueueState{q.els})
	if err != nil {
		glog.Errorln("Error encoding elements: ", err)
	}
	glog.V(2).Infof("Persisted.")
}

func (q *queueImpl) findInternal(id string) (pos int) {
	pos = -1
	for i, el := range q.els {
		if el.Id == id {
			pos = i
			break
		}
	}
	return
}

func (q *queueImpl) removeInternal(i int) (err error) {
	q.els = append(q.els[:i], q.els[i+1:]...)
	return
}

func (q *queueImpl) Find(id string) (pos int, err error) {
	pos = q.findInternal(id)
	if pos < 0 {
		err = errors.New("Element does not exist.")
	}
	return
}

func (q *queueImpl) Put(el Element) (pos int, err error) {
	pos = q.findInternal(el.Id)
	if pos > -1 {
		glog.Infof("Duplicate put for id %v", el.Id)
		err = AlreadyExistsError{Id: el.Id, Timestamp: q.els[pos].QTime}
		return
	}
	glog.Infof("Put %s", el.Id)
	q.els = append(q.els, el)
	pos = len(q.els) - 1
	q.Persist()
	return
}

func (q *queueImpl) TakeFront() (el Element, err error) {
	if len(q.els) == 0 {
		glog.Infof("Take %s", el.Id)
		err = errors.New("empty queue")
		return
	}
	el = q.els[0]
	q.els = q.els[1:]
	q.Persist()
	return
}

func (q *queueImpl) takeInternal(i int) (el Element, err error) {
	if i < 0 || i >= len(q.els) {
		glog.Errorf("Fail to take element %d for queue length %d (%v)", i, len(q.els), q.els)
		err = errors.New("No such element")
		return
	}
	el = q.els[i]
	q.removeInternal(i)
	return
}

func (q *queueImpl) Take(i int) (el Element, err error) {
	el, err = q.takeInternal(i)
	if err == nil {
		q.Persist()
	}
	return
}

func (q *queueImpl) Size() int {
	return len(q.els)
}

func (q *queueImpl) dbg() {
	glog.V(2).Infof("%v", q.dlist())
}

func (q *queueImpl) Get(i int) (el Element, err error) {
	if i < 0 || i >= len(q.els) {
		err = errors.New("No such element")
		return
	}
	el = q.els[i]
	return
}

func (q *queueImpl) Remove(i int) (err error) {
	glog.V(2).Infof("Remove %d", i)
	if i < 0 || i >= len(q.els) {
		err = errors.New("No such element")
		return
	}
	q.removeInternal(i)
	q.Persist()
	return
}

func (q *queueImpl) dlist() (lst []string) {
	lst = make([]string, len(q.els))
	for i, el := range q.els {
		lst[i] = el.Id
	}
	return
}

func (q *queueImpl) Move(i int, npos int) (err error) {
	glog.V(0).Infof("Move %d -> %d", i, npos)
	if i < 0 || i >= len(q.els) || npos < 0 || npos >= len(q.els) {
		err = errors.New("No such element")
		return
	}
	if i == npos {
		return
	}
	el, _ := q.takeInternal(i)
	tmp := q.els
	q.els = make([]Element, len(tmp)+1)
	copy(q.els, tmp[:npos])
	q.els[npos] = el
	copy(q.els[npos+1:], tmp[npos:])
	q.Persist()
	return
}

func (q *queueImpl) List() (els []Element) {
	els = make([]Element, len(q.els))
	n := copy(els, q.els)
	if n != len(q.els) {
		glog.Errorf("Of %d, only %d copied", len(q.els), n)
	}
	return
}
