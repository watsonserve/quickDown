package downloader

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/watsonserve/goutils"
)

func Reduction(txt string) []goutils.Range_t {
	txt = strings.Trim(txt, "\n")
	lines := strings.Split(txt, "\n")
	length := len(lines)
	arr := make([]goutils.Range_t, length)
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

func NewStore(rawUrl string, size int64, outFile string, cfgFileName string) (*Store_t, error) {
	// 创建本地文件
	outStream, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE, 0666)
	if nil != err {
		return nil, err
	}
	if "" == cfgFileName {
		cfgFileName = outFile + ".qdt"
	}
	cfgStream, err := os.OpenFile(cfgFileName, os.O_WRONLY|os.O_CREATE, 0666)
	if nil != err {
		return nil, err
	}
	fileInfo := fmt.Sprintf("%s\n%s\n%d\n", rawUrl, outFile, size)
	return &Store_t{
		FileInfo:  fileInfo,
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

func (this *Store_t) Sync(arr []goutils.Range_t) {
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
