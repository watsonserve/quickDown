package http_downloader

import (
    "errors"
    "fmt"
    "os"
    "github.com/watsonserve/quickDown/remote"
    "time"
)

type OriginFile_t struct {
    File  DownTask_t
    Range Range_t
    Body  []byte
}

type DownTask_t struct {
    Pool_t
    IOProcess_t
    certValid    bool
    httpResource *remote.HttpResource
    writer       *os.File
}

/**
 * 构造函数
 */
func New(writer *os.File, reader *remote.HttpResource, block int64, sgmTrd int) *DownTask_t {
    this := &DownTask_t{
        IOProcess_t: IOProcess_t {},
        httpResource: reader,
        writer: writer,
    }
    size := reader.Size()
    // 计算分片
    if reader.Parallelable() {
        this.BlockSlice_t = *NewBlockSlice(size, sgmTrd, block)
    } else {
        this.sgmTrd = 1
        this.size = size
        this.block = size
    }
    // debug
    fmt.Fprintf(os.Stderr, "block: %d\nthread: %d\n", this.block, this.sgmTrd)
    return this
}

/**
 * 生产者
 */
func (this *DownTask_t) Download() error {
    if this.size < 1 {
        return errors.New("unknow origin file size")
    }
    if nil == this.writer {
        return errors.New("no out stream")
    }
    this.IOProcess_t = *NewIOProcess(&this.BlockSlice_t, this.writer)
    this.Pool_t = *this.initPool()

    // 准备工作已经完成
    err := this.load()
    if nil != err {
        return err
    }

    this.writer.Close()
    fmt.Fprintf(os.Stderr, "----\n{\"cost\": \"%ds\"}\n", time.Now().Unix() - this.startTime)
    return nil
}

func (this *DownTask_t) initPool() *Pool_t {
    taskPipe := make(chan *Range_t, this.sgmTrd)
    notifyPipe := make(chan *Range_t, this.sgmTrd << 1)

    for i := 0; i < this.sgmTrd; i++ {
        go this.worker(taskPipe, notifyPipe)
    }
    return &Pool_t {
        taskPipe: taskPipe,
        notifyPipe: notifyPipe,
    }
}

func (this *DownTask_t) load() error {
    offset := int64(0)
    id := int64(0)

    fmt.Fprintf(os.Stderr, ".")
    for ; id < int64(this.sgmTrd); id++ {
        // 入队
        foo := this.Cut(offset)
        if nil == foo {
            break
        }
        foo.Id = id
        this.Push(foo)
        offset = foo.End
    }

    fmt.Fprintf(os.Stderr, ".\n")
    // 任务发放完成 并且 全部线程均已关闭
    for 0 < this.sgmTrd {
        ranger := this.Wait()
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
        this.Record(ranger)

        id++
        foo := this.Cut(offset)
        if this.Fill() {
            foo = nil
        }
        if nil != foo {
            foo.Id = id
            offset = foo.End
        }
        this.Push(foo)
    }

    return nil
}

/**
 * 消费者
 * 传入nil使线程结束
 */
func (this *DownTask_t) worker(taskPipe chan *Range_t, notifyPipe chan *Range_t) {
    httpRequester, err := this.httpResource.NewHttpReader()
    if nil != err {
        notifyPipe <- nil
        return
    }
    for ranger := <- taskPipe; nil != ranger; ranger = <- taskPipe {
        rsp, err := httpRequester.Read(ranger.Start, ranger.End, 3)
        if nil == err {
            err = this.Write(rsp.Body, ranger.Start)
        }
        ranger.Err = err
        notifyPipe <- ranger
    }
    // 得到的任务为nil则传出nil
    notifyPipe <- nil
}

func (this *DownTask_t) Emit(cmd string) {
    switch cmd {
    case "check":
        arr := this.completedLink.ToArray()
        for i := 0; i < len(arr); i++ {
            fmt.Printf("{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
        }
    case "quit":
        os.Exit(0)
    }
}

func (this *DownTask_t) EmitError(err error) {
    fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
    os.Exit(0)
}
