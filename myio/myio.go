package myio

import (
    "bufio"
    "io"
)

type ReadStream struct {
    scanner *bufio.Scanner
    EOF     bool
}

// 用回车换行符分隔
func (this *ReadStream) SplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
    length := len(data)
    // 有且只有atEOF为true时，data可能是空数组
    var i int
    for i = 1; i < length; i++ {
        lst := i - 1
        if '\n' == data[i] && '\r' == data[lst] {
            return i + 1, data[0 : lst], nil
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
    this := &ReadStream {
        scanner: scanner,
        EOF: false,
    }
    scanner.Split(this.SplitFunc)
    return this
}

func (this *ReadStream) ReadLine() (string, error) {
    hasCuted := this.scanner.Scan()
    err := this.scanner.Err()
    if nil != err {
        return "", err
    }
    if !hasCuted {
        this.EOF = true
    }
    msg := this.scanner.Text()
    return msg, nil
}