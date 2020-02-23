package main

/*/ #cgo CFLAGS: -O3 */

import (
    //	"C"
    "fmt"
    "github.com/watsonserve/quickDown/downloader"
    "github.com/watsonserve/quickDown/http_downloader"
    "os"
)

func main() {
    options, err := getOptions()
    if nil != err {
        fmt.Fprintln(os.Stderr, err.Error())
        help()
        return
    }

    if "" != options.OutPath {
        // 变更到输出目录
        err = os.Chdir(options.OutPath)
        if nil != err {
            fmt.Fprintln(os.Stderr, err.Error())
            return
        }
    }

    // filter the protocol
    proto, err := parseResource(options)
    for nil == err {
        var subject downloader.Subject_t
        var loader downloader.Task_t
        var store *downloader.Store_t

        // 创建下载任务
        switch proto {
        case "http":
            subject, err = http_downloader.New(options)
        // case "ftp":
        // case "p2p":
        default:
            return
        }
        if nil != err {
            break
        }
        fmt.Printf("meta loading...\r\n")
        meta := subject.GetMeta()
        // debug
        fmt.Printf("{size: %d, block: %d, thread: %d}\r\n", meta.Size, meta.Block, meta.SgmTrd)
        store, err = downloader.CreateStore(meta)
        if nil != err {
            break
        }
        loader, err = subject.CreateTask(store)
        if nil != err {
            break
        }

        // 注册信号监听
        go downloader.ListenSign(loader)
        // 监听标准输入流
        go downloader.ListenCmd(loader, os.Stdin)

        err = loader.Download()
        if nil != err {
            break
        }
        return
    }
    fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
    return
}
