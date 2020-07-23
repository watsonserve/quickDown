package main

/*/ #cgo CFLAGS: -O3 */

import (
    //	"C"
    "fmt"
    "os"
    "github.com/watsonserve/quickDown/downloader"
)

func main() {
    options, err := getOptions()
    if nil != err {
        fmt.Fprintln(os.Stderr, err.Error())
        help()
        return
    }

    if "" == options.ConfigFile && "" != options.OutPath {
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
        loader, create_err := create(proto, options)
        if nil != err {
            err = create_err
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
