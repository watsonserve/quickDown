package myio

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type ReadStream struct {
	scanner *bufio.Scanner
	EOF     bool
}

// 用回车换行符分隔
func (readStream *ReadStream) SplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	length := len(data)
	// 有且只有atEOF为true时，data可能是空数组
	var i int
	for i = 1; i < length; i++ {
		if '\n' == data[i] {
			lst := i
			if '\r' == data[i-1] {
				lst = i - 1
			}
			return i + 1, data[0:lst], nil
		}
	}
	// 没有找到分隔符
	if atEOF {
		return length, data, nil
	}
	return 0, nil, nil
}

func InitReadStream(sock io.Reader) *ReadStream {
	scanner := bufio.NewScanner(sock)
	this := &ReadStream{
		scanner: scanner,
		EOF:     false,
	}
	scanner.Split(this.SplitFunc)
	return this
}

func (readStream *ReadStream) ReadLine() (string, error) {
	hasCuted := readStream.scanner.Scan()
	err := readStream.scanner.Err()
	if nil != err {
		return "", err
	}
	if !hasCuted {
		readStream.EOF = true
	}
	msg := readStream.scanner.Text()
	return msg, nil
}

/**
 * 线程安全
 */
func SendFileAt(of *os.File, rs io.ReadCloser, wOff int64) error {
	buf, err := ioutil.ReadAll(rs)
	if nil != err {
		return err
	}
	bufLen := len(buf)
	length, err := of.WriteAt(buf, wOff)
	if nil != err {
		return err
	}

	if bufLen != length {
		return fmt.Errorf("write faild, len: %d", length)
	}
	return nil
}
