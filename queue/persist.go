package queue

import (
	"github.com/golang/glog"

	"encoding/json"
	"io"
	"os"
)

type Persister interface {
	Write(state []Element) (err error)
	Read() (state []Element)
}

type FilePersister struct {
	fn string
}

func (fp FilePersister) Write(state []Element) (err error) {
	glog.Infoln("Writing to ", fp.fn)
	glog.Infoln("Backing up ", fp.fn)
	err = os.Rename(fp.fn, fp.fn+".bak")
	if err != nil && !os.IsNotExist(err) {
		glog.Fatalln("Could not rename ", fp.fn, ": ", err)
	}
	f, err := os.Create(fp.fn)
	if err != nil {
		glog.Fatalln("Error opening ", fp.fn, ": ", err)
	}

	writer := json.NewEncoder(f)
	for _, el := range state {
		err = writer.Encode(el)
		if err != nil {
			glog.Errorf("Error encoding %v", el)
		}
	}
	return
}

func (fp FilePersister) Read() (state []Element) {
	f, err := os.Open(fp.fn)
	if err != nil {
		glog.Fatalln(err)
	}

	reader := json.NewDecoder(f)
	var el Element
	for {
		err := reader.Decode(&el)
		if err != nil {
			if err != io.EOF {
				glog.Fatalln(err)
			}
			break
		}
		state = append(state, el)
	}
	return
}
