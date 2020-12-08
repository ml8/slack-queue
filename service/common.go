package service

import (
	"github.com/golang/glog"

	"io/ioutil"
)

func readFile(fn string) (content string) {
	bytes, err := ioutil.ReadFile(fn)
	content = string(bytes)
	if err != nil {
		glog.Fatalf("Could not open %v: %v", fn, err)
	}
	return
}
