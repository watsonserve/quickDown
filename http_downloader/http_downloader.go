package http_downloader

import (
    "errors"
    "fmt"
    "os"
    "time"
    "github.com/watsonserve/quickDown/downloader"
    "github.com/watsonserve/quickDown/http_downloader/remote"
    "github.com/watsonserve/quickDown/link_table"
)

type HttpSuject_t struct {
    sgmTrd       int
    size         int64
    block        int64
    rawUrl       string
    outFileName  string
    httpResource *remote.HttpResource
}

type HttpTask_t struct {
    BlockSlice_t
    httpResource *remote.HttpResource
    store        *downloader.Store_t
}

/**
 * 构造函数
 */
func New(options *downloader.Options_t) (*HttpSuject_t, error) {
    // 一个远端资源对象
    httpResource, err := remote.NewHttpResource(options.RawUrl)
    if nil == err {
        // 读取远端资源的元数据
        err = httpResource.GetMeta()
    }
    if nil != err {
        return nil, err
    }
    size := httpResource.Size()
    fileName := httpResource.Filename()
    // 若没有指定文件名，自动设定文件名
    if 0 < len(options.OutFile) || len(fileName) < 1 {
        fileName = options.OutFile
    }
    // 计算分片
    trd := 1
    block := size
    if httpResource.Parallelable() {
        block, trd = GetBlockSlice(size, options.SgmTrd, options.Block)
    }

    return &HttpSuject_t {
        size:         size,
        sgmTrd:       trd,
        block:        block,
        outFileName:  fileName,
        rawUrl:       options.RawUrl,
        httpResource: httpResource,
    }, nil
}

func (this *HttpSuject_t) GetMeta() *downloader.Meta_t {
    return &downloader.Meta_t {
        Size:    this.size,
        SgmTrd:  this.sgmTrd,
        Block:   this.block,
        OutFile: this.outFileName,
        RawUrl:  this.rawUrl,
    }
}

func (this *HttpSuject_t) CreateTask(store *downloader.Store_t, linker []link_table.Line_t) (downloader.Task_t, error) {
    // 一个下载器实例
    return &HttpTask_t {
        BlockSlice_t: *NewBlockSlice(this.size, this.sgmTrd, this.block, linker),
        httpResource: this.httpResource,
        store:        store,
    }, nil
}

/**
 * 生产者
 */
func (this *HttpTask_t) Download() error {
    offset := int64(0)
    id := int64(0)
    if this.size < 1 {
        return errors.New("unknow origin file size")
    }
    pool := NewPool(this, this.sgmTrd)

    fmt.Fprintf(os.Stderr, ".")
    for ; id < int64(this.sgmTrd); id++ {
        // 入队
        foo := this.Cut(offset)
        if nil == foo {
            break
        }
        foo.Id = id
        pool.Push(foo)
        offset = foo.End
    }

    fmt.Fprintf(os.Stderr, ".\n")
    // 任务发放完成 并且 全部线程均已关闭
    for 0 < this.sgmTrd {
        ranger := pool.Wait()
        // 线程退出
        if nil == ranger {
            this.sgmTrd--
            if 0 == this.sgmTrd {
                break
            }
            continue
        }
        // 错误
        if nil != ranger.Err {
            fmt.Fprintf(os.Stderr, "Error in worker: range: %d-%d\n%s", ranger.Start, ranger.End, ranger.Err.Error())
            // TODO
            continue
        }
        id++
        foo := this.Cut(offset)
        if this.Fill(ranger) {
            foo = nil
        }
        if nil != foo {
            foo.Id = id
            offset = foo.End
        }
        pool.Push(foo)
    }
    this.store.Close()
    fmt.Fprintf(os.Stderr, "----\n{\"cost\": \"%ds\"}\n", time.Now().Unix() - this.startTime)

    return nil
}

/**
 * 消费者
 * 传入nil使线程结束
 */
func (this *HttpTask_t) Worker(taskPipe chan *Range_t, notifyPipe chan *Range_t) {
    httpRequester, err := this.httpResource.NewHttpReader()
    if nil != err {
        notifyPipe <- nil
        return
    }
    for ranger := <- taskPipe; nil != ranger; ranger = <- taskPipe {
        rsp, err := httpRequester.Read(ranger.Start, ranger.End, 3)
        if nil == err {
            err = this.store.SendFileAt(rsp.Body, ranger.Start)
            rsp.Body.Close()
        }
        ranger.Err = err
        notifyPipe <- ranger
    }
    // 得到的任务为nil则传出nil
    notifyPipe <- nil
}

func (this *HttpTask_t) Emit(cmd string) {
    switch cmd {
    case "check":
        arr := this.completedLink.ToArray()
        for i := 0; i < len(arr); i++ {
            fmt.Printf("{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
        }
    case "quit":
        arr := this.completedLink.ToArray()
        this.store.Sync(arr)
        this.store.Close()
        os.Exit(0)
    }
}

func (this *HttpTask_t) EmitError(err error) {
    fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
    os.Exit(0)
}
