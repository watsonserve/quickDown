package main

/*/ #cgo CFLAGS: -O3 */

import (
    //	"C"
    "encoding/base64"
    "errors"
    "fmt"
    "github.com/watsonserve/quickDown/ctrler"
    "github.com/watsonserve/quickDown/http_downloader"
    "github.com/watsonserve/quickDown/remote"
    "net/url"
    "os"
)

func help() {
    fmt.Fprintln(os.Stderr, "version 1.0 License GPL2.0")
    fmt.Fprintln(os.Stderr, "(C) watsonserve.com made by James Watson\n")
    fmt.Fprintln(os.Stderr, "use [-b blockSize|-t sumOfThread|-o outputFile|--stdout] url")
    fmt.Fprintln(os.Stderr, "     -b block Size, will be integer multiples of 64K(max: 16). default is 1 multiple")
    fmt.Fprintln(os.Stderr, "     -t sum Of Thread. default is 1, max: 128")
    fmt.Fprintln(os.Stderr, "     -o output file name. auto set")
    fmt.Fprintln(os.Stderr, "     -h show this help information\n")
}

//export Http_download
// func Http_download(urlStr *C.char, outFile *C.char, block int64, sgmTrd int) C.int {
// 	downloader := httpDownloader.New(C.GoString(urlStr), C.GoString(outFile), block, sgmTrd)
// 	ret := downloader.Download()
// 	if nil != ret {
// 		return -1
// 	}
// 	return 0
// }

func httpDownload(options *Options_t) (*http_downloader.DownTask_t, error) {
    // 一个远端资源对象
    httpResource, err := remote.NewHttpResource(options.RawUrl)
    if nil == err {
        // 读取远端资源的元数据
        err = httpResource.GetMeta()
    }
    if nil != err {
        return nil, err
    }
    fileName := httpResource.Filename()
    // 若没有指定文件名，自动设定文件名
    if len(options.OutFile) < 1 && 0 < len(fileName) {
        options.OutFile = fileName
    }
    // 创建本地文件
    outStream, err := os.OpenFile(options.OutFile, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
        return nil, err
    }
    // 一个下载器实例
    downloader := http_downloader.New(outStream, httpResource, options.Block, options.SgmTrd)
    return downloader, nil
}

func parseResource(raw_url string) (string, error) {
    // 解析远端资源类型
    uri, err := url.Parse(raw_url)
    if nil != err {
        return "", errors.New("ERROR url")
    }
    // filter the protocol
    switch uri.Scheme {
    case "thunder":
        data, err := base64.StdEncoding.DecodeString(uri.Opaque)
        if nil != err {
            return "", err
        }
        length := len(data)
        if "AA" == string(data[0:2]) && "ZZ" == string(data[length-2:length]) {
            return parseResource(string(data[2:length-2]))
        }
    case "http":
        fallthrough
    case "https":
        return "http", nil
    default:
        return "", errors.New("ERROR unsuppored protocol " + uri.Scheme)
    }
    return "", errors.New("ERROR unknow")
}

func main() {
    var err     error
    var options *Options_t

    // 获取命令行参数
    options, err = getOptions()
    if nil != err {
        fmt.Fprintln(os.Stderr, err.Error())
        help()
        return
    }

    // filter the protocol
    _, err = parseResource(options.RawUrl)
    if nil != err {
        fmt.Fprintln(os.Stderr, err.Error())
        return
    }

    downloader, err := httpDownload(options)
    if nil != err {
        fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
    }

    // 注册信号监听
    go ctrler.ListenSign(downloader)
    // 监听标准输入流
    go ctrler.ListenCmd(downloader, os.Stdin)

    downloader.Download()
    return
}