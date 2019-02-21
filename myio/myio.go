package myio

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func SendFileAt(rs io.Reader, ws io.WriterAt, w_off int64) error {
	buf, err := ioutil.ReadAll(rs)
	if nil != err {
		return err
	}
	length, err := ws.WriteAt(buf, w_off)
	if nil != err {
		return err
	}
	if len(buf) != length {
		fmt.Fprintf(os.Stderr, "write not complete, len: %d", length)
	}
	return nil
}
