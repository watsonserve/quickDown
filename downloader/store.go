package downloader

import (
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "strings"
    "github.com/watsonserve/quickDown/link_table"
)

// 从一个文件中读取前n行和后面所有内容
func ReadLineN(filename string, n int) ([]string, error) {
    content, err := ioutil.ReadFile(filename)
    if nil != err {
        return make([]string, 0), err
    }

    lines := strings.SplitN(string(content), "\n", n)
    return lines, nil
}

func Reduction(txt string) []link_table.Line_t {
    lines := strings.Split(txt, "\n")
    length := len(lines)
    arr := make([]link_table.Line_t, length)
    for i := 0; i < length; i++ {
        fmt.Sscanf(lines[i], "{start: %d, end: %d}", &arr[i].Start, &arr[i].End)
    }
    return arr
}

type Store_t struct {
    FileInfo  string
    outStream *os.File
    cfgStream *os.File
}

func newStore(fileInfo string, outFile string, cfgFileName string) (*Store_t, error) {
    // 创建本地文件
    outStream, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
        return nil, err
    }
    cfgStream, err := os.OpenFile(cfgFileName, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
        return nil, err
    }
    return &Store_t {
        FileInfo:  fileInfo,
        outStream: outStream,
        cfgStream: cfgStream,
    }, nil
}

func Resume(cfgFileName string) (*Store_t, []link_table.Line_t, error) {
    lines, err := ReadLineN(cfgFileName, 4)
    if nil != err {
        return nil, nil, errors.New("Read Config file: " + err.Error())
    }
    rawUrl := lines[0]
    outFile := lines[1]
    size := lines[2]
    arr := Reduction(lines[3])
    fileInfo := fmt.Sprintf("%s\n%s\n%s\n", rawUrl, outFile, size)
    store, err := newStore(fileInfo, outFile, cfgFileName)
    return store, arr, err
}

func CreateStore(meta *Meta_t) (*Store_t, error) {
    fileInfo := fmt.Sprintf("%s\n%s\n%d\n", meta.RawUrl, meta.OutFile, meta.Size)
    return newStore(fileInfo, meta.OutFile, meta.OutFile + ".qdt")
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

func (this *Store_t) Sync(arr []link_table.Line_t) {
    this.cfgStream.Truncate(0)
    fmt.Fprintf(this.cfgStream, "%s\n", this.FileInfo)
    for i := 0; i < len(arr); i++ {
        fmt.Fprintf(this.cfgStream, "{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
    }
}

func (this *Store_t) Close() {
    this.outStream.Close()
    this.cfgStream.Close()
}
