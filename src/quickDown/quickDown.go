package main

import (
    "strconv"
    "fmt"
    "os"
    "net/url"
    "quickDown/downloaders"
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

func express(file *os.File, downer downloaders.Downloader, pipe chan [2]int64, repipe chan int64, id int) {
    for {
        task := <- pipe
        start := task[0]
        end := task[1]
        httpRange := make(map[string]string)
        httpRange["range"] = strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10)
        httpRange["method"] = "GET"
        response, err := downer.Load(nil, &httpRange)
        if nil != err {
            return
        }
        size, err := strconv.ParseInt((*response)["Content-Length"], 0, 0)
        var length int64
        fmt.Fprintf(os.Stderr, "id %d have load %d byte.\n", id, size)
        length = 0
        content := []byte((*response)["content"])
        for length < size {

            fmt.Fprintf(os.Stderr, "write\n")
            leng, err := file.WriteAt(content[length:], start + length)
            if nil != err {
              fmt.Fprintln(os.Stderr, err)
                return
            }
            length += int64(leng)
            fmt.Fprintf(os.Stderr, "write length: %d\n", length)
        }
        repipe <- length
    }
}
func sendTask(block int64, allsize int64, taskPipe chan [2]int64) {
    var taskBlock [2]int64
    var off int64
    for off = 0; off < allsize; {
        taskBlock[0] = off
        off += block
        taskBlock[1] = off
        fmt.Fprintln(os.Stderr, taskBlock)
        taskPipe <- taskBlock
    }
    return
}

func ByHttp(outFile string, uri *url.URL, block *int64, sgmTrd *int64, notifyPipe chan int64) (int64, *os.File, error) {
    downloader := downloaders.NewHttpDownloader(uri)
    response, err := downloader.Load(nil, nil)
    if nil != err {
        return -1, nil, err
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
    fmt.Fprintf(os.Stderr, "block: %d\nthread: %d\n", *block, *sgmTrd)
    taskPipe := make(chan [2]int64, *sgmTrd)
    storer, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
      return -1, nil, err
    }
    for i := 0; i < int(*sgmTrd); i++ {
        go express(storer, downloader, taskPipe, notifyPipe, i)
    }
    go sendTask(*block, allsize, taskPipe)
    return allsize, storer, nil
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
    var allsize int64
    var storer *os.File
    block = 1
    outFile := ""
    // cfgFile := -1
    sgmTrd = 1
    urlstr := ""
    // get option
    for i := 2; i < argc; i++ {
        if '-' == argv[i-1][0] {
            switch(argv[i-1][1]) {
                case 'b':
                    block, err = strconv.ParseInt(argv[i], 0, 0)
                    if nil != err {
                        fmt.Fprintln(os.Stderr, "ERROR block should be a intger")
                        return
                    }
                    continue
                case 'o':
                    outFile = argv[i]
                    continue
                case 't':
                    sgmTrd, err = strconv.ParseInt(argv[i], 0, 0)
                    if nil != err {
                        fmt.Fprintln(os.Stderr, "ERROR number of thread should be a intger")
                        return
                    }
                    continue
                default:
                    fmt.Fprintln(os.Stderr, "ERROR unknow option " + argv[i])
                case 'h':
                    help()
                    return
            }
        } else {
            urlstr = argv[i]
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
    if "" == urlstr {
        help()
        return
    }
    fmt.Fprintln(os.Stderr, "url: " + urlstr)

    uri, err = url.Parse(urlstr)
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
            allsize, storer, err = ByHttp(outFile, uri, &block, &sgmTrd, notifyPipe)
            break
        default:
            fmt.Printf("ERROR unsuppored protocol %s\n", uri.Scheme)
            return
    }
    fmt.Fprintf(os.Stderr, "file-size: %d\nwaiting...", allsize)
    for {
        finish := <- notifyPipe
        if -1 == finish {
            fmt.Fprintln(os.Stderr, "ERROR")
            return
        }
        allsize -= finish
        if 0 >= allsize {
            break
        }
    }
    storer.Close()
    fmt.Fprintln(os.Stderr, "finish")
    return
}
