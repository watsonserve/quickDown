package main

import (
    "strconv"
    "fmt"
    "os"
    "net/url"
    "net/http"
    "io/ioutil"
)

func help() {
    fmt.Fprintln(os.Stderr, "version 1.0 License MIT")
    fmt.Fprintln(os.Stderr, "(C) watsonserve.com made by James Watson\n")
    fmt.Fprintln(os.Stderr, "use [-b blockSize|-t sumOfThread|-o outputFile] url")
    fmt.Fprintln(os.Stderr, "     -b block Size, will be integer multiples of 64K. default is 1 multiple")
    fmt.Fprintln(os.Stderr, "     -t sum Of Thread. default is 1, max is 512")
    fmt.Fprintln(os.Stderr, "     -o output File. default is stdout, and sum of thread will be 1")
    fmt.Fprintln(os.Stderr, "     -h show this help information\n")
}

func express(file *os.File, urlStr string, pipe chan [2]int64, repipe chan int64, id int) {
    for {
        task := <- pipe
        start := task[0]
        end := task[1]

        client := &http.Client{}
        req, err := http.NewRequest("GET", urlStr, nil)
        if nil != err {
            fmt.Fprintln(os.Stderr, err)
            return
        }
        req.Header.Add("range", strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10))
        resp, err := client.Do(req)
        if nil != err {
            fmt.Fprintln(os.Stderr, err)
            return
        }
        if 200 == resp.StatusCode && 0 != id {
            break
        }
        body, err := ioutil.ReadAll(resp.Body)
        if nil != err {
            fmt.Fprintln(os.Stderr, err)
            return
        }
        resp.Body.Close()
        leng, err := file.WriteAt(body, start)
        if nil != err {
            fmt.Fprintln(os.Stderr, err)
            return
        }
        repipe <- int64(leng)
    }
    return
}

func sendTask(block int64, allSize int64, taskPipe chan [2]int64) {
    var taskBlock [2]int64
    var off int64
    for off = 0; off < allSize; {
        taskBlock[0] = off
        off += block
        taskBlock[1] = off
        fmt.Fprintln(os.Stderr, taskBlock)
        taskPipe <- taskBlock
    }
    return
}

func ByHttp(outFile string, uri string, block *int64, sgmTrd *int64, notifyPipe chan int64) (int64, *os.File, error) {
    client := &http.Client{}
    resp, err := client.Head(uri)
    if nil != err {
        return -1, nil, err
    }
    resp.Body.Close()
    allSize := resp.ContentLength
    if 1 == *sgmTrd {    // signal thread
        *block = allSize
    } else if 0 == *block {    // no repeat
        *block = allSize / *sgmTrd +1
    } else if 0 == *sgmTrd || allSize / *block < *sgmTrd {    // less thread or more thread
        *sgmTrd = allSize / *block
        if 0 != allSize % *block {
            *sgmTrd++
        }
    }
    fmt.Fprintf(os.Stderr, "block: %d\nthread: %d\n", *block, *sgmTrd)
    taskPipe := make(chan [2]int64, *sgmTrd)
    storer, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
      return -1, nil, err
    }
    for i := 0; i < int(*sgmTrd); i++ {
        go express(storer, uri, taskPipe, notifyPipe, i)
    }
    go sendTask(*block, allSize, taskPipe)
    return allSize, storer, nil
}

func main() {
    argv := os.Args
    argc := len(argv)
    if 2 > argc {  // 没有给出任何参数
        help()
        return
    }

    var block int64
    var sgmTrd int64
    var err error
    var uri *url.URL
    var allSize int64
    var storer *os.File
    block = 1
    outFile := ""
    // cfgFile := -1
    sgmTrd = 1
    urlStr := ""
    // get option
    for i := 1; i < argc; i++ {
        argp := argv[i]
        var nextArg string
        nextArg = ""
        if i+1 < argc {
            nextArg = argv[i+1]
        }
        if '-' == argp[0] {  // 一个选项
            switch(argp[1]) {
                case 'b':
                    block, err = strconv.ParseInt(nextArg, 0, 0)
                    if nil != err {
                        fmt.Fprintln(os.Stderr, "ERROR block should be a intger")
                        return
                    }
                    continue
                case 'o':
                    outFile = nextArg
                    continue
                case 't':
                    sgmTrd, err = strconv.ParseInt(nextArg, 0, 0)
                    if nil != err {
                        fmt.Fprintln(os.Stderr, "ERROR number of thread should be a intger")
                        return
                    }
                    continue
                default:
                    fmt.Fprintln(os.Stderr, "ERROR unknow option " + nextArg)
                case 'h':
                    help()
                    return
            }
        } else {
            urlStr = argp  // 预下载文件地址
        }
    }

    if 0 == block && 0 == sgmTrd {
        fmt.Fprintln(os.Stderr, "ERROR block and number of thread can't both be 0")
        return
    }
    block <<= 16    // 64KB
    if sgmTrd > 512 {    // max number of thread
        sgmTrd = 512
    }
    if "" == urlStr {
        help()
        return
    }
    // debug
    fmt.Fprintln(os.Stderr, "url: " + urlStr)

    uri, err = url.Parse(urlStr)
    if nil != err {
        fmt.Fprintln(os.Stderr, "ERROR url")
        return
    }
    notifyPipe := make(chan int64, 8)
    // filter the protocol
    switch uri.Scheme {
        case "thunder":
            fmt.Fprintln(os.Stderr, "parse base64")
            return
        case  "http":
            allSize, storer, err = ByHttp(outFile, urlStr, &block, &sgmTrd, notifyPipe)
            break
        default:
            fmt.Printf("ERROR unsuppored protocol %s\n", uri.Scheme)
            return
    }


    fmt.Fprintf(os.Stderr, "file-size: %d waiting...\n", allSize)
    for {
        finish := <- notifyPipe
        if -1 == finish {
            fmt.Fprintln(os.Stderr, "ERROR")
            return
        }
        allSize -= finish
        if 0 >= allSize {
            break
        }
    }
    storer.Close()
    fmt.Fprintln(os.Stderr, "finish")
    return
}
