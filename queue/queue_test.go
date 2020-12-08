package queue

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"testing"
	"time"
)

// TODO: Test error cases.

var q Queue

func validateList(t *testing.T, els []Element, expected []int) {
	lst := make([]string, len(els))
	for i, el := range els {
		lst[i] = el.Id
	}
	for i, el := range els {
		v, _ := strconv.Atoi(el.Id)
		if expected[i] != v {
			t.Fatalf("Expected %d, got %d: %v", expected[i], v, lst)
		}
	}
}

func validate(t *testing.T, expected []int) {
	els := q.List()
	validateList(t, els, expected)
}

func clear(t *testing.T, errs []error) {
	for i, err := range errs {
		if err != nil {
			t.Fatalf("Error %d non-nil: %s", i, err)
		}
	}
}

func testPutList(t *testing.T) {
	validate(t, []int{0, 1, 2})
}

func testPutDuplicates(t *testing.T) {
	q.Put(Element{Id: "1"})
	_, err := q.Put(Element{Id: "1"})
	_, ok := err.(AlreadyExistsError)
	if !ok {
		t.Fatalf("Expected AlreadyExistsError, got %#v", err)
	}
}

func testPutTakeFront(t *testing.T) {
	for i := 0; i < 3; i++ {
		el, err := q.TakeFront()
		if err != nil {
			t.Fatalf("Expected no error")
		}
		v, _ := strconv.Atoi(el.Id)
		if i != v {
			t.Fatalf("Expected %d, got %d", i, v)
		}
	}
}

func TestPut(t *testing.T) {
	q = MakeQueue(nil)
	q.Put(Element{Id: "0", QTime: time.Now()})
	q.Put(Element{Id: "1", QTime: time.Now()})
	q.Put(Element{Id: "2", QTime: time.Now()})

	t.Run("PutList", testPutList)
	t.Run("PutTakeFront", testPutTakeFront)
	t.Run("PutDuplicates", testPutDuplicates)
}

func testMove(t *testing.T) {
	errs := make([]error, 6)
	// 0 1 2 3 4 5 6 7 8 9
	errs[0] = q.Move(0, 2)
	// 1 2 0 3 4 5 6 7 8 9
	errs[1] = q.Move(9, 1)
	// 1 9 2 0 3 4 5 6 7 8
	errs[2] = q.Move(5, 7)
	// 1 9 2 0 3 5 6 4 7 8
	errs[3] = q.Move(9, 0)
	// 8 1 9 2 0 3 5 6 4 7
	errs[4] = q.Move(0, 2)
	// 1 9 8 2 0 3 5 6 4 7
	errs[5] = q.Move(0, 9)
	// 9 8 2 0 3 5 6 4 7 1
	clear(t, errs)
	validate(t, []int{9, 8, 2, 0, 3, 5, 6, 4, 7, 1})
}

func testRemove(t *testing.T) {
	errs := make([]error, 4)
	errs[0] = q.Remove(0)
	// 1 2 3 4 5 6 7 8 9
	errs[1] = q.Remove(8)
	// 1 2 3 4 5 6 7 8
	errs[2] = q.Remove(2)
	// 1 2 4 5 6 7 8
	errs[3] = q.Remove(4)
	// 1 2 4 5 7 8
	clear(t, errs)
	validate(t, []int{1, 2, 4, 5, 7, 8})
}

func testMoveRemove(t *testing.T) {
	errs := make([]error, 10)
	// 0 1 2 3 4 5 6 7 8 9
	errs[0] = q.Remove(1)
	// 0 2 3 4 5 6 7 8 9
	errs[1] = q.Move(3, 0)
	// 4 0 2 3 5 6 7 8 9
	errs[2] = q.Move(3, 8)
	// 4 0 2 5 6 7 8 9 3
	errs[3] = q.Move(5, 2)
	// 4 0 7 2 5 6 8 9 3
	errs[4] = q.Remove(8)
	// 4 0 7 2 5 6 8 9
	errs[5] = q.Remove(2)
	// 4 0 2 5 6 8 9
	errs[6] = q.Remove(3)
	// 4 0 2 6 8 9
	errs[7] = q.Move(4, 5)
	// 4 0 2 6 9 8
	errs[8] = q.Remove(5)
	// 4 0 2 6 9 3
	errs[9] = q.Move(1, 2)
	// 4 2 0 6 9
	clear(t, errs)
	validate(t, []int{4, 2, 0, 6, 9})
}

func TestReorder(t *testing.T) {
	qi := &queueImpl{}
	els := make([]Element, 10)
	qi.els = make([]Element, 10)
	for i := 0; i < 10; i++ {
		els[i] = Element{Id: strconv.Itoa(i), QTime: time.Now()}
	}
	copy(qi.els, els)
	q = qi

	t.Run("Remove", testRemove)
	qi.els = make([]Element, 10)
	copy(qi.els, els)
	t.Run("Move", testMove)
	qi.els = make([]Element, 10)
	copy(qi.els, els)
	t.Run("MoveRemove", testMoveRemove)
}

func TestGetTake(t *testing.T) {
	qi := &queueImpl{}
	els := make([]Element, 10)
	qi.els = make([]Element, 10)
	for i := 0; i < 10; i++ {
		els[i] = Element{Id: strconv.Itoa(i), QTime: time.Now()}
	}
	copy(qi.els, els)
	q = qi

	vals := make([]Element, 4)
	errs := make([]error, 4)
	expected := []int{0, 9, 2, 5}

	vals[0], errs[0] = q.Get(0)
	vals[1], errs[1] = q.Get(9)
	vals[2], errs[2] = q.Get(2)
	vals[3], errs[3] = q.Get(5)

	clear(t, errs)
	validateList(t, vals, expected)

	vals[0], errs[0] = q.Take(0)
	vals[1], errs[1] = q.Take(8)
	vals[2], errs[2] = q.Take(1)
	vals[3], errs[3] = q.Take(3)

	clear(t, errs)
	validateList(t, vals, expected)
}

func PersistTest(t *testing.T) {
	fn := t.TempDir() + "/state"
	fp := FilePersister{fn: fn}
	q = MakeQueue(fp)

	expected := make([]int, 100)
	for i := 0; i < 100; i++ {
		expected[i] = i
		q.Put(Element{Id: strconv.Itoa(i), QTime: time.Now()})
	}

	q.Persist()
	q.Persist()
	if !deepCompare(t, fn, fn+".bak") {
		t.Fatalf("Failed to copy backup file")
	}

	q = MakeQueue(fp)
	q.Recover()

	validate(t, expected)
}

const chunkSize = 64000

// From https://stackoverflow.com/a/30038571
func deepCompare(t *testing.T, file1, file2 string) bool {
	// Check file size ...

	f1, err := os.Open(file1)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	for {
		b1 := make([]byte, chunkSize)
		_, err1 := f1.Read(b1)

		b2 := make([]byte, chunkSize)
		_, err2 := f2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true
			} else if err1 == io.EOF || err2 == io.EOF {
				return false
			} else {
				t.Fatal(err1, err2)
			}
		}

		if !bytes.Equal(b1, b2) {
			return false
		}
	}
}
