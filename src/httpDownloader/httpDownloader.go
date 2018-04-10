package httpDownloader

import (
    "strconv"
    "strings"
    "fmt"
    "os"
    "path"
    "net/url"
    "errors"
    "net/http"
    "io"
    "io/ioutil"
    "httpUtils"
    "time"
)

type Range_t struct {
    Id    int64
    Start int64
    End   int64
    Err   error
}

type Resp_t struct {
    Length int64
    Body   io.ReadCloser
}

type OriginFile_t struct {
    File  DownTask_t
    Range Range_t
    Body  []byte
}

var units = []string{
    "B", "KB", "MB", "GB",
}

func sendFileAt(rs io.Reader, ws io.WriterAt, w_off int64) error {
    buf, err := ioutil.ReadAll(rs)
    if nil != err {
        return err
    }
    ws.WriteAt(buf, w_off)
    return nil
}

/**
 * 分片
 * @params {int64} 总大小
 * @params {int}   线程数
 * @params {int64} 块大小
 * @return {int}   线程数
 * @return {int64} 块大小
 */
 func cut(size int64, intTrd int, block int64) (int, int64) {
    maxTrd := int64(128)
    maxBlock := int64(1 << 20)
    defaultBlock := int64(65536)
    trd := int64(intTrd)
    block <<= 16

    for {
        // TODO
        if 0 == trd && 0 == block {
            trd = maxTrd
            block = defaultBlock
        }

        // 块大小不为0
        if 0 != block && 0 == trd {
            trd = size / block
            // less thread or more thread
            if 0 != size % block {
                trd++
            }
            // 如果size小于block，则会出现单线程模式
        }

        // 单线程模式
        if 1 == trd {
            block = size
            break
        }

        // 指定线程数，计算分块大小
        if 0 != trd && 0 == block {
            // no repeat
            block = size / trd +1
        }
        break
    }

    // 最大值限制
    if maxTrd < trd {
        trd = maxTrd
    }
    if maxBlock < block {
        block = maxBlock
    }
    return int(trd), block
}

type DownTask_t struct {
    uri            *url.URL
    Url            string
    LocalFileName  string
    ContentLength  int64
    maxSeek        int64
    Block          int64
    SgmTrd         int
    Tls            bool
    CanRange       bool
    Store          *os.File
}

/**
 * 构造函数
 */
func New(url_raw string, fileName string, block int64, sgmTrd int) *DownTask_t {
    uri, err := url.Parse(url_raw)
    if nil != err {
        return nil
    }
    this := &DownTask_t{
        Url: url_raw,
        uri: uri,
        Block: block,
        SgmTrd: sgmTrd,
        Tls: "https" == uri.Scheme,
    }
    return this
}

func (this *DownTask_t) push(start int64) *Range_t {
    max := this.maxSeek
    if max < start {
        return nil
    }
    end := start + this.Block
    if max < end {
        end = max
    }
    return &Range_t {
        Start: start,
        End: end,
    }
}

/**
 * 试着获取远端信息，文件名和内容长度
 * @return {error}
 */
func (this *DownTask_t) originInfo() error {
    resp, err := httpUtils.Dail(this.Url, "HEAD", nil, this.Tls)
    if nil != err {
        return err
    }
    // 应答错误
    if 200 != resp.StatusCode {
        return errors.New(resp.Status)
    }
    resp.Body.Close()

    acceptRanges := resp.Header.Get("Accept-Ranges")
    contentLength := resp.Header.Get("Content-Length")

    this.CanRange = "" != acceptRanges && "none" != acceptRanges
    fmt.Fprintf(os.Stderr, "content-length: %s\n", contentLength)
    i, err := strconv.ParseInt(contentLength, 10, 64)
    if nil != err {
        return err
    }
    this.ContentLength = i
    this.maxSeek = i - 1

    // 若没有指定文件名，自动设定文件名
    if len(this.LocalFileName) < 1 {
        // 优先使用应答头里的文件名
        fileName := resp.Header.Get("Content-Disposition")
        if 0 < len(fileName) {
            foo := strings.Split(fileName, "filename=")
            if 0 < len(foo) {
                fileName = foo[1]
            }
            if '"' == fileName[0] {
                fileName = fileName[1:len(fileName) - 1]
            }
        }
        // 使用url的文件名
        if len(fileName) < 1 {
            fileName = path.Base(this.uri.Path)
        }
        this.LocalFileName = fileName
    }
    return nil
}

/**
 * 请求一个分片
 * @params {int64}   start
 * @params {int64}   end
 * @return {*Resp_t}
 * @return {error}
 */
func (this *DownTask_t) express(start int64, end int64) (*Resp_t, error) {
    headers := &http.Header{}
    headers.Add("Range", "bytes=" + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10))
    resp, err := httpUtils.Dail(this.Url, "GET", headers, this.Tls)
    if nil != err {
        return nil, err
    }
    // 应答错误
    if 2 != resp.StatusCode / 100 {
        return nil, errors.New(resp.Status)
    }
    // TODO
    // rangeLength := resp.Header.Get("Content-Range")
    // fmt.Fprintf(os.Stderr, "content-range: %s\n", rangeLength)

    // 直接返回流
    return &Resp_t{
        Length: 0,
        Body:   resp.Body,
    }, nil
}

/**
 * 消费者
 */
func (this *DownTask_t) worker(taskPipe chan *Range_t, notifyPipe chan *Range_t) {
    for ranger := <- taskPipe; nil != ranger; ranger = <- taskPipe {
        rsp, err := this.express(ranger.Start, ranger.End)
        if nil == err {
            err = sendFileAt(rsp.Body, this.Store, ranger.Start)
            rsp.Body.Close()
        }
        if nil != err {
            fmt.Fprintf(os.Stderr, "Error in worker:\nrange: %d-%d\n", ranger.Start, ranger.End)
        }
        ranger.Err = err
        notifyPipe <- ranger
    }
}

/**
 * 生产者
 */
func (this *DownTask_t) Download() error {
    // 获取远端文件信息
    err := this.originInfo()
    if nil != err {
        return err
    }
    // 创建本地文件
    storer, err := os.OpenFile(this.LocalFileName, os.O_WRONLY|os.O_CREATE, 0666)
    if nil != err {
      return err
    }
    this.Store = storer
    // 计算分片
    trd, block := cut(this.ContentLength, this.SgmTrd, this.Block)
    this.SgmTrd = trd
    this.Block = block

    // debug
    fmt.Fprintf(os.Stderr, "block: %d\nthread: %d\n", this.Block, this.SgmTrd)
    fmt.Fprintln(os.Stderr, "waiting...")

    // 准备工作已经完成
    err = this.load()
    if nil != err {
        return err
    }

    storer.Close()
    fmt.Fprintln(os.Stderr, "----\nfinish")

    return nil
}

func (this *DownTask_t) load() error {
    taskPipe := make(chan *Range_t, this.SgmTrd)
    notifyPipe := make(chan *Range_t, 8)

    size := this.ContentLength
    block := this.Block
    offset := int64(0)
    id := int64(0)
    doneSeek := int64(0)

    startTime := time.Now().Unix()
    for ; id < int64(this.SgmTrd); id++ {
        // 入队
        foo := this.push(offset)
        if nil == foo {
            break
        }
        foo.Id = id
        taskPipe <- foo
        offset = foo.End
        go this.worker(taskPipe, notifyPipe)
    }

    for doneSeek < size {
        ranger := <- notifyPipe
        if nil != ranger.Err {
            return ranger.Err
        }
        doneSeek += block
        id++
        foo := this.push(offset)
        if nil != foo {
            foo.Id = id
            offset = foo.End
        }
        taskPipe <- foo

        // 统计
        progress, velocity, unit := statistic(startTime, doneSeek, size)
        fmt.Fprintf(os.Stderr, "完成: %0.2f%%\t速度: %0.2f%s/s\n", progress, velocity, unit)
    }

    return nil
}

/**
 * 统计
 */
func statistic(startTime int64, doneSeek int64, size int64) (float32, float32, string) {
    var unit_p byte
    progress := float32(100 * float64(doneSeek) / float64(size))
    if 100 < progress {
        progress = 100
    }
    velocity := float32(float64(doneSeek) / float64(time.Now().Unix() - startTime))
    for unit_p = 0; 1024 < velocity; unit_p++ {
        velocity /= 1024
    }
    return progress, velocity, units[unit_p]
}