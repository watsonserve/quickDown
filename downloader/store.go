package downloader

import (
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "github.com/watsonserve/quickDown/data_struct"
)

type Store_t struct {
    fileInfo  string
    outStream *os.File
    cfgStream *os.File
}

func CreateStore(meta *Meta_t) (*Store_t, error) {
    fileInfo := fmt.Sprintf("%s\n%s\n%d\n", meta.RawUrl, meta.OutFile, meta.Size)
    // 创建本地文件
    outStream, err := os.OpenFile(meta.OutFile, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
        return nil, err
    }
    cfgStream, err := os.OpenFile(meta.OutFile + ".qdt", os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
        return nil, err
    }
    return &Store_t {
        fileInfo:  fileInfo,
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
    bufLen := len(buf)
	length, err := this.outStream.WriteAt(buf, w_off)
	if nil != err {
		return err
    }
    
	if bufLen != length {
		return errors.New(fmt.Sprintf("write faild, len: %d", length))
	}
	return nil
}

func (this *Store_t) Sync(arr []data_struct.Line_t) {
    this.cfgStream.Truncate(0)
    fmt.Fprintf(this.cfgStream, "%s\n", this.fileInfo)
    for i := 0; i < len(arr); i++ {
        fmt.Fprintf(this.cfgStream, "{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
    }
}

func (this *Store_t) Close() {
    this.outStream.Close()
    this.cfgStream.Close()
}
