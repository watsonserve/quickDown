package main

import (
    "encoding/base64"
    "errors"
    "fmt"
    "net/url"
    "os"
    "strconv"
    "github.com/watsonserve/goutils"
    "github.com/watsonserve/quickDown/downloader"
)

func help() {
    fmt.Fprintln(os.Stderr, "version 0.2.0 License GPL2.0")
    fmt.Fprintf(os.Stderr, "(C) watsonserve.com made by James Watson\n\n")
    fmt.Fprintln(os.Stderr, "use [-b blockSize|-t sumOfThread|-o outputFile|--stdout] url")
    fmt.Fprintln(os.Stderr, "     -c, --config   use a config file")
    fmt.Fprintln(os.Stderr, "     -b, --block    block Size, will be integer multiples of 64K(max: 16). default is 1 multiple")
    fmt.Fprintln(os.Stderr, "     -t, --thread   sum Of Thread. default is 1, max: 256")
    fmt.Fprintln(os.Stderr, "     -o, --output   output file name. auto set")
    fmt.Fprintf(os.Stderr,  "     -h, --help     show this help information\n\n")
}

func getOptions() (*downloader.Options_t, error) {
    var err error
    // 获取命令行参数
    allOptions := []goutils.Option{
        {
            Opt: 'b',
            Option: "block",
            HasParams: true,
        },
        {
            Opt: 'c',
            Option: "config",
            HasParams: true,
        },
        {
            Opt: 't',
            Option: "thread",
            HasParams: true,
        },
        {
            Opt: 'o',
            Option: "output",
            HasParams: true,
        },
        {
            Opt: 'h',
            Option: "help",
            HasParams: false,
        },
    }
    optionMap, urls := goutils.GetOptions(allOptions)

    _, ok := optionMap["help"]
    if ok {
        return nil, errors.New("")
    }

    // block
    var block int64 = 1
    strBlock, ok := optionMap["block"]
    if ok {
        block, err = strconv.ParseInt(strBlock, 0, 0)
        if nil != err {
            return nil, errors.New("Error block should be a intger")
        }
    }

    // thread
    var thread int64 = 1
    strThread, ok := optionMap["thread"]
    if ok {
        thread, err = strconv.ParseInt(strThread, 0, 0)
        if nil != err {
            return nil, errors.New("Error thread should be a intger")
        }
    }

    // url
    if 1 != len(urls) {
        return nil, errors.New("Error download url")
    }

    return &downloader.Options_t {
        SgmTrd: int(thread),
        Block: block,
        OutFile: optionMap["output"],
        RawUrl: urls[0],
    }, nil
}

/**
 * 解析远端资源协议类型，如为包裹协议（比如thunder://）则解除
 * @params Options_t 需要命令行上的url地址，如需要解除包装协议会修改options.RawUrl
 * @return string 真实协议名
 * @return error  错误
 */
func parseResource(options *downloader.Options_t) (string, error) {
    // 解析远端资源类型
    uri, err := url.Parse(options.RawUrl)
    if nil != err {
        return "", errors.New("ERROR url")
    }
    // filter the protocol
    switch uri.Scheme {
    case "thunder":
        data, err := base64.StdEncoding.DecodeString(uri.Host)
        if nil != err {
            return "", err
        }
        length := len(data)
        if 4 < length && "AA" == string(data[0:2]) && "ZZ" == string(data[length - 2 : length]) {
            options.RawUrl = string(data[2:length-2])
            return parseResource(options)
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
