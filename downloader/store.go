package downloader

import (
    "errors"
    "fmt"
    "io"
    "io/ioutil"
	"os"
)

type Store_t struct {
    outStream *os.File
    cfgStream *os.File
}

func CreateStore(options *Meta_t) (*Store_t, error) {
    fmt.Printf("filename: %s\n", options.OutFile)
    // 创建本地文件
    outStream, err := os.OpenFile(options.OutFile, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
        return nil, err
    }
    cfgStream, err := os.OpenFile(options.OutFile + ".qdt", os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
        return nil, err
    }
    return &Store_t {
        outStream: outStream,
        cfgStream: cfgStream,
    }, nil
}

/**
 * 线程安全
 */
func (this *Store_t) SendFileAt(rs io.ReadCloser, w_off int64) error {
	buf, err := ioutil.ReadAll(rs)
	if nil != err {
		return err
    }
    bugLen := len(buf)
	length, err := this.outStream.WriteAt(buf, w_off)
	if nil != err {
		return err
    }
    
	if bugLen != length {
		return errors.New(fmt.Sprintf("write faild, len: %d", length))
	}
	return nil
}

func (this *Store_t) Close() {
    this.outStream.Close()
    this.cfgStream.Close()
}
