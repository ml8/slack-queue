package persister

import (
	"github.com/golang/glog"

	"encoding/json"
	"os"
)

type Persister interface {
	Write(state interface{}) (err error)
	Read(state interface{})
	Id() string // TODO this is a hack, remove.
}

type FilePersister struct {
	Fn string
}

func (fp FilePersister) Id() string {
	return fp.Fn
}

func (fp FilePersister) Write(state interface{}) (err error) {
	glog.V(3).Infof("Writing to %v", fp.Fn)
	glog.V(3).Infof("Backing up %v", fp.Fn)
	err = os.Rename(fp.Fn, fp.Fn+".bak")
	if err != nil && !os.IsNotExist(err) {
		glog.Fatalln("Could not rename %v: %v", fp.Fn, err)
	}
	f, err := os.Create(fp.Fn)
	if err != nil {
		glog.Fatalln("Error opening %v: %v", fp.Fn, err)
	}

	glog.Infof("%+v", state)

	writer := json.NewEncoder(f)
	err = writer.Encode(state)

	if err != nil {
		glog.Fatalf("Error encoding %v", state)
	}
	return
}

func (fp FilePersister) Read(state interface{}) {
	glog.V(3).Infof("Reading from %v", fp.Fn)
	f, err := os.Open(fp.Fn)
	if err != nil {
		if os.IsNotExist(err) {
			glog.Infof("Nothing to recover...")
			return
		} else {
			glog.Fatalln(err)
		}
	}
	glog.Infof("%+v", state)

	reader := json.NewDecoder(f)
	err = reader.Decode(state)
	if err != nil {
		glog.Fatalln(err)
	}
	return
}
