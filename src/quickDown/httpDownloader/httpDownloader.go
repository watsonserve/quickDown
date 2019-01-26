package httpDownloader

import (
    "fmt"
    "io"
    "os"
    "quickDown/link"
    "quickDown/myio"
    "quickDown/httpUtils"
    "syscall"
    "time"
)

type Range_t struct {
    Id    int64
    Start int64
    End   int64
    Err   error
}


type OriginFile_t struct {
    File  DownTask_t
    Range Range_t
    Body  []byte
}

var units = []string{
    "B", "KB", "MB", "GB",
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
            if 0 != size%block {
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
            block = size/trd + 1
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
    httpRequester    *httpUtils.HTTPRequest
    completedLink    *link.TaskLink
    store            *io.WriterAt
    maxSeek          int64
    contentLength    int64
    block            int64
    sgmTrd           int
    canRange         bool
    startTime        int64
}

/**
 * 构造函数
 */
func New(url_raw string, block int64, sgmTrd int) *DownTask_t {
    httpRequester := httpUtils.New(url_raw)
    if nil == httpRequester {
        return nil
    }

    return &DownTask_t{
        httpRequester: httpRequester,
        completedLink: link.New(nil),
        store: nil,
        contentLength: 0,
        block:  block,
        sgmTrd: sgmTrd,
    }
}

func (this *DownTask_t) push(start int64) *Range_t {
    max := this.maxSeek
    if max <= start {
        return nil
    }
    end := start + this.block
    if max < end {
        end = max
    }
    return &Range_t{
        Start: start,
        End:   end,
    }
}

/**
 * 试着获取远端信息，文件名和内容长度
 * @return {error}
 */
func (this *DownTask_t) OriginInfo() (error, string) {
    err, canRange, contentLength, fileName := this.httpRequester.OriginInfo()
    if nil != err {
        return err, nil
    }
    this.canRange = canRange
    this.contentLength = contentLength
    this.maxSeek = contentLength - 1

    return nil, fileName
}

/**
 * 消费者
 * 传入nil使线程结束
 */
func (this *DownTask_t) worker(taskPipe chan *Range_t, notifyPipe chan *Range_t) {
    for ranger := <-taskPipe; nil != ranger; ranger = <-taskPipe {
        rsp, err := this.httpRequester.RequestRange(ranger.Start, ranger.End, 3)
        if nil == err {
            err = myio.SendFileAt(rsp.Body, this.store, ranger.Start)
            rsp.Body.Close()
        }
        ranger.Err = err
        notifyPipe <- ranger
    }
    // 得到的任务为nil则传出nil
    notifyPipe <- nil
}

func (this *DownTask_t) On(sigChannel chan os.Signal) {
    for {
        s := <- sigChannel
        arr := this.completedLink.ToArray()
        for i := 0; i < len(arr); i++ {
            fmt.Printf("{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
        }
        switch s {
        // case syscall.SIGTSTP:
        //     fallthrough
        case syscall.SIGINT:
            fallthrough
        case syscall.SIGQUIT:
            fallthrough
        case syscall.SIGTERM:
            os.Exit(0)
        }
    }
}

/**
 * 生产者
 */
func (this *DownTask_t) Download(outStream *io.WriterAt) error {
    if this.contentLength < 1 {
        return errors.New("unknow origin file size")
    }
    if nil == outStream {
        return errors.New("no out stream")
    }
    this.store = outStream;

    // 计算分片
    trd, block := cut(this.contentLength, this.sgmTrd, this.block)
    this.sgmTrd = trd
    this.block = block

    // debug
    fmt.Fprintf(os.Stderr, "block: %d\nthread: %d\n", this.block, this.sgmTrd)

    // 准备工作已经完成
    err = this.load()
    if nil != err {
        return err
    }

    outStream.Close()
    fmt.Fprintf(os.Stderr, "----\n{\"cost\": \"%ds\"}\n", time.Now().Unix() - this.startTime)
    return nil
}

func (this *DownTask_t) load() error {
    fmt.Fprintf(os.Stderr, "waiting")
    taskPipe := make(chan *Range_t, this.sgmTrd)
    notifyPipe := make(chan *Range_t, this.sgmTrd << 1)

    fmt.Fprintf(os.Stderr, ".")
    size := this.contentLength
    block := this.block
    offset := int64(0)
    id := int64(0)
    doneSeek := int64(0)
    startTime := time.Now().Unix()
    this.startTime = startTime

    fmt.Fprintf(os.Stderr, ".")
    for ; id < int64(this.sgmTrd); id++ {
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

    fmt.Fprintln(os.Stderr, ".\n")
    // 任务发放完成 并且 全部线程均已关闭
    for 0 < this.sgmTrd {
        ranger := <-notifyPipe
        // 线程退出
        if nil == ranger {
            this.sgmTrd--
            continue
        }
        // 错误
        if nil != ranger.Err {
            fmt.Fprintf(os.Stderr, "Error in worker: range: %d-%d\n%s", ranger.Start, ranger.End, ranger.Err.Error())
            // TODO
            continue
        }
        this.completedLink.Mount(ranger.Start, ranger.End)
        doneSeek += block
        id++
        foo := this.push(offset)
        if size < doneSeek {
            foo = nil
        }
        if nil != foo {
            foo.Id = id
            offset = foo.End
        }
        taskPipe <- foo

        // 统计
        progress, velocity, unit, planTime := statistic(startTime, doneSeek, size)
        fmt.Fprintf(
            os.Stderr,
            "{\"finish\": \"%0.2f%%\", \"speed\": \"%0.2f%s/s\", \"planTime\": \"%ds\"}\n",
            progress, velocity, unit, planTime,
        )
    }

    return nil
}

/**
 * 统计
 */
func statistic(startTime int64, doneSeek int64, size int64) (float32, float32, string, int) {
    var unit_p byte
    progress := float32(100 * float64(doneSeek) / float64(size))
    if 100 < progress {
        progress = 100
    }
    delta := float64(time.Now().Unix() - startTime)
    if delta < 0.1 {
        delta = 0.1
    }
    velocity := float32(float64(doneSeek) / delta)

    planTime := int((size - doneSeek) / int64(velocity))

    for unit_p = 0; 1024 < velocity; unit_p++ {
        velocity /= 1024
    }
    return progress, velocity, units[unit_p], planTime
}
