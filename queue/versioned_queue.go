package queue

import (
	"github.com/golang/glog"
	"github.com/matthewlang/slack-queue/persister"

	"fmt"
	"sync"
)

// Versioned queue for concurrent and asynchronous modifications.
//
// All operations return a version number associated with the state of an
// underlying queue. Modification operations must supply a version number,
// except for blind writes (e.g., Put and TakeFront). All possible modifications
// to the backing queue increase the version number, even on error.
//
// Version mismatches return a VersionError.
//
// Thread safe.
type VersionedQueue struct {
	q   Queue // wrapped queue
	seq int64 // sequence number
	mu  sync.Mutex
}

func VQ(persist persister.Persister) (vq *VersionedQueue) {
	vq = &VersionedQueue{}
	vq.q = MakeQueue(persist)
	return
}

type VersionError struct {
	Current   int64 // current sequence number
	Attempted int64
}

func (ve VersionError) Error() string {
	return fmt.Sprintf("Version mismatch, attempted %d for current version %d\n", ve.Attempted, ve.Current)
}

func (vq *VersionedQueue) checkSeq(seq int64) (err error) {
	if seq != vq.seq {
		glog.Errorf("Sequence number mismatch %d for current gen %d", seq, vq.seq)
		err = VersionError{Current: vq.seq, Attempted: seq}
	}
	return
}

func (vq *VersionedQueue) Put(el Element) (pos int, seq int64, err error) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	pos, err = vq.q.Put(el)
	vq.seq += 1
	seq = vq.seq
	return
}

func (vq *VersionedQueue) TakeFront() (el Element, seq int64, err error) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	el, err = vq.q.TakeFront()
	vq.seq += 1
	seq = vq.seq
	return
}

func (vq *VersionedQueue) Take(i int, seq int64) (el Element, nseq int64, err error) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	err = vq.checkSeq(seq)
	if err != nil {
		nseq = vq.seq
		return
	}
	el, err = vq.q.Take(i)
	vq.seq += 1
	nseq = vq.seq
	return
}

func (vq *VersionedQueue) Get(i int, seq int64) (el Element, nseq int64, err error) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	nseq = vq.seq
	err = vq.checkSeq(seq)
	if err != nil {
		return
	}
	el, err = vq.q.Get(i)
	return
}

func (vq *VersionedQueue) Remove(i int, seq int64) (nseq int64, err error) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	err = vq.checkSeq(seq)
	if err != nil {
		nseq = vq.seq
		return
	}
	err = vq.q.Remove(i)
	vq.seq += 1
	nseq = vq.seq
	return
}

func (vq *VersionedQueue) Move(i int, npos int, seq int64) (nseq int64, err error) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	err = vq.checkSeq(seq)
	if err != nil {
		return
	}
	err = vq.q.Move(i, npos)
	vq.seq += 1
	nseq = vq.seq
	return
}

func (vq *VersionedQueue) Find(id string) (pos int, seq int64, err error) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	pos, err = vq.q.Find(id)
	seq = vq.seq
	return
}

func (vq *VersionedQueue) List() (els []Element, seq int64) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	els = vq.q.List()
	seq = vq.seq
	return
}

func (vq *VersionedQueue) Size() (size int, seq int64) {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	size = vq.q.Size()
	seq = vq.seq
	return
}

func (vq *VersionedQueue) Persist() {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	vq.q.Persist()
}

func (vq *VersionedQueue) Recover() {
	vq.mu.Lock()
	defer vq.mu.Unlock()
	vq.q.Recover()
	vq.seq = 0
}
