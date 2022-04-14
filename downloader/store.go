package downloader

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/watsonserve/goutils"
	"github.com/watsonserve/quickDown/myio"
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
func (that *Store_t) SendFileAt(rs io.ReadCloser, w_off int64) error {
	return myio.SendFileAt(that.outStream, rs, w_off)
}

func (that *Store_t) Sync(arr []goutils.Range_t) {
	that.cfgStream.Truncate(0)
	fmt.Fprintf(that.cfgStream, "%s\n", that.FileInfo)
	for i := 0; i < len(arr); i++ {
		fmt.Fprintf(that.cfgStream, "{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
	}
}

func (that *Store_t) Close() {
	that.outStream.Close()
	that.cfgStream.Close()
}
