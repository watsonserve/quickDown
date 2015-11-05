package main

import (
    "strings"
    "strconv"
    "fmt"
    "os"
    "net/url"
    "quickDown/downloaders"
)

func help() {
    fmt.Println("version 1.0 License MIT")
    fmt.Println("(C) watsonserve.com made by James Watson")
    fmt.Println("use [-b blockSize|-t sumOfThread|-o outputFile] url")
    fmt.Println("     -b block Size, will be integer multiples of 64K. default is 1 multiple")
    fmt.Println("     -t sum Of Thread. default is 1, max is 512")
    fmt.Println("     -o output File. default is stdout, and sum of thread will be 1")
    fmt.Println("     -h show this help information\n")
}

func express(file *os.File, downer downloaders.Downloader, pipe chan string, repipe chan int64) {
    for {
        task := <- pipe
        if "" == task {
            return
        }
        se := strings.Split(task, "-")
        offset, err := strconv.ParseInt(se[0], 0, 0)
        length, err := strconv.ParseInt(se[1], 0, 0)
        httpRange := make(map[string]string)
        httpRange["range"] = se[0] + "-" + strconv.FormatInt(offset + length, 10)
        response, err := downer.Load(&httpRange)
        if nil != err {
            return
        }
        size, err := strconv.ParseInt((*response)["Content-Length"], 0, 0)
        for length < size {
            leng, err := file.WriteAt([]byte((*response)["content"]), offset)
            if nil != err {
                return
            }
            length = int64(leng)
        }
        repipe <- length
    }
}

func ByHttp(outFile string, uri *url.URL, block *int64, sgmTrd *int64, notifyPipe chan int64) (int64, error) {
    downloader := downloaders.NewHttpDownloader(uri)
    response, err := downloader.Load(nil)
    if nil != err {
        return -1, err
    }
    allsize, err := strconv.ParseInt((*response)["Content-Length"], 0, 0)
    if 1 == *sgmTrd {    // signal thread
        *block = allsize
    } else if 0 == *block {    // no repeat
        *block = allsize / *sgmTrd +1
    } else if 0 == *sgmTrd || allsize / *block < *sgmTrd {    // less thread or more thread
        *sgmTrd = allsize / *block
        if 0 != allsize % *block {
            *sgmTrd++
        }
    }
    taskPipe := make(chan string, *sgmTrd)
    storer, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
      return -1, err
    }
    defer storer.Close()
    for i := 0; i < int(*sgmTrd); i++ {
        go express(storer, downloader, taskPipe, notifyPipe)
    }
    return allsize, nil
}

func main() {
    argv := os.Args
    argc := len(argv)
    if 2 > argc {
        help()
        return
    }

    var block int64
    var sgmTrd int64
    var err error
    var uri *url.URL
    block = 1
    outFile := ""
    //cfgFile := -1
    sgmTrd = 1
    urlstr := ""

    for i := 1; i < argc; i++ {
        if '-' == argv[i-1][0] {
            switch(argv[i-1][1]) {
                case 'b':
                    block, err = strconv.ParseInt(argv[i], 0, 0)
                    if nil != err {
                        fmt.Println("ERROR block should be a intger")
                        return
                    }
                    continue
                case 'o':
                    outFile = argv[i]
                    continue
                case 't':
                    sgmTrd, err = strconv.ParseInt(argv[i], 0, 0)
                    if nil != err {
                        fmt.Println("ERROR number of thread should be a intger")
                        return
                    }
                    continue
                default:
                    fmt.Println("ERROR unknow option " + argv[i])
                case 'h':
                    help()
                    return
            }
        } else {
            urlstr = argv[i]
        }
    }
    if 0 == block && 0 == sgmTrd {
        fmt.Println("ERROR block and number of thread can't both be 0")
        return
    }
    block <<= 16
    if sgmTrd > 512 {
        sgmTrd = 512
    }
    if "" == urlstr {
        help()
        return
    }

    uri, err = url.Parse(urlstr)
    if nil != err {
        fmt.Println("ERROR url")
        return
    }
    notifyPipe := make(chan int64, 8)
    var allsize int64
    switch uri.Scheme {
        case "thunder":
            fmt.Println("parse base64")
            break
        case  "http":
            allsize, err = ByHttp(outFile, uri, &block, &sgmTrd, notifyPipe)
            break
        default:
            fmt.Printf("ERROR unsuppored protocol %s\n", uri.Scheme)
            return
    }
    for {
        finish := <- notifyPipe
        if -1 == finish {
            fmt.Println("ERROR")
            return
        }
        allsize -= finish
        if 0 >= allsize {
            break
        }
    }
    fmt.Println("finish")
    return
}
