package queue

import (
	"strconv"
	"testing"
)

var vq *VersionedQueue
var seq int64

func populate(vq *VersionedQueue, n int) {
	for i := 0; i < n; i++ {
		vq.Put(Element{Id: strconv.Itoa(i)})
	}
}

func TestPutIncreases(t *testing.T) {
	vq = VQ(nil)
	oseq := vq.seq
	_, seq, _ = vq.Put(Element{Id: "1"})
	if oseq >= vq.seq {
		t.Fatal("Failed to increase sequence number on write.")
	}
	if seq != vq.seq {
		t.Fatal("Incorrect sequence number returned.")
	}
}

func TestTakeFrontIncreases(t *testing.T) {
	vq = VQ(nil)
	populate(vq, 1)
	oseq := vq.seq
	_, seq, _ = vq.TakeFront()
	if oseq >= vq.seq {
		t.Fatal("Failed to increase sequence number on write.")
	}
	if seq != vq.seq {
		t.Fatal("Incorrect sequence number returned.")
	}
}

func testTakeSucceeds(t *testing.T) {
	_, _, err := vq.Take(0, seq)
	if err != nil {
		t.Fatalf("Take failed with correct sequence number: %v", err)
	}
}

func testTakeFails(t *testing.T) {
	_, _, err := vq.Take(0, seq+1)
	if err == nil {
		t.Fatalf("Take succeeded with incorrect sequence number %d vs %d", seq, seq+1)
	}
	_, ok := err.(VersionError)
	if !ok {
		t.Fatalf("Incorrect error type returned: %#v", err)
	}
}

func testTakeIncreases(t *testing.T) {
	_, nseq, _ := vq.Take(0, seq)
	_, _, err := vq.Take(0, nseq)
	if err != nil {
		t.Fatalf("Take failed with correct sequence number %d vs %d", nseq, vq.seq)
	}
}

func TestTake(t *testing.T) {
	vq = VQ(nil)
	populate(vq, 10)

	seq = vq.seq
	t.Run("TakeSucceeds", testTakeSucceeds)
	seq = vq.seq
	t.Run("TakeFails", testTakeSucceeds)
	seq = vq.seq
	t.Run("TakeIncreases", testTakeIncreases)
}

func testRemoveSucceeds(t *testing.T) {
	_, err := vq.Remove(0, seq)
	if err != nil {
		t.Fatalf("Remove failed with correct sequence number: %v", err)
	}
}

func testRemoveFails(t *testing.T) {
	_, err := vq.Remove(0, seq+1)
	if err == nil {
		t.Fatalf("Remove succeeded with incorrect sequence number %d vs %d", seq, seq+1)
	}
	_, ok := err.(VersionError)
	if !ok {
		t.Fatalf("Incorrect error type returned: %#v", err)
	}
}

func testRemoveIncreases(t *testing.T) {
	nseq, _ := vq.Remove(0, seq)
	_, err := vq.Remove(0, nseq)
	if err != nil {
		t.Fatalf("Remove failed with correct sequence number %d vs %d", nseq, vq.seq)
	}
}

func TestRemove(t *testing.T) {
	vq = VQ(nil)
	populate(vq, 10)

	seq = vq.seq
	t.Run("RemoveSucceeds", testRemoveSucceeds)
	seq = vq.seq
	t.Run("RemoveFails", testRemoveSucceeds)
	seq = vq.seq
	t.Run("RemoveIncreases", testRemoveIncreases)
}

func testMoveSucceeds(t *testing.T) {
	_, err := vq.Move(0, 1, seq)
	if err != nil {
		t.Fatalf("Move failed with correct sequence number: %v", err)
	}
}

func testMoveFails(t *testing.T) {
	_, err := vq.Move(0, 1, seq+1)
	if err == nil {
		t.Fatalf("Move succeeded with incorrect sequence number %d vs %d", seq, seq+1)
	}
	_, ok := err.(VersionError)
	if !ok {
		t.Fatalf("Incorrect error type returned: %#v", err)
	}
}

func testMoveIncreases(t *testing.T) {
	nseq, _ := vq.Move(0, 1, seq)
	_, err := vq.Move(0, 1, nseq)
	if err != nil {
		t.Fatalf("Move failed with correct sequence number %d vs %d", nseq, vq.seq)
	}
}

func TestMove(t *testing.T) {
	vq = VQ(nil)
	populate(vq, 10)

	seq = vq.seq
	t.Run("MoveSucceeds", testMoveSucceeds)
	seq = vq.seq
	t.Run("MoveFails", testMoveSucceeds)
	seq = vq.seq
	t.Run("MoveIncreases", testMoveIncreases)
}

func TestFind(t *testing.T) {
	vq = VQ(nil)
	populate(vq, 10)

	seq = vq.seq

	_, oseq, _ := vq.Find("3")
	_, nseq, _ := vq.Find("4")
	if seq != oseq || seq != nseq || seq != vq.seq {
		t.Fatalf("Find should not modify the sequence number (start (%d), iteration 1 (%d), iteration 2 (%d), current (%d))", seq, oseq, nseq, vq.seq)
	}
}

func TestSize(t *testing.T) {
	vq = VQ(nil)
	populate(vq, 10)

	seq = vq.seq

	_, oseq := vq.Size()
	_, nseq := vq.Size()
	if seq != oseq || seq != nseq || seq != vq.seq {
		t.Fatalf("Size should not modify the sequence number (start (%d), iteration 1 (%d), iteration 2 (%d), current (%d))", seq, oseq, nseq, vq.seq)
	}
}

func TestRecoverResets(t *testing.T) {
	vq = VQ(nil)
	populate(vq, 10)

	if vq.seq == 0 {
		t.Fatal("zero starting value before recovery")
	}
	vq.Recover()
	if vq.seq != 0 {
		t.Fatal("Sequence number not resest upon recovery")
	}
}
