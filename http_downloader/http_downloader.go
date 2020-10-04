package http_downloader

import (
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"
    "github.com/watsonserve/quickDown/downloader"
    "github.com/watsonserve/quickDown/http_downloader/remote"
    "github.com/watsonserve/quickDown/link_table"
)

type HttpTask_t struct {
    BlockSlice_t
    httpResource *remote.HttpResource
    store        *downloader.Store_t
}

func resume(cfgFileName string) (*downloader.Store_t, []link_table.Line_t, int64, string, error) {
    lines, err := downloader.ReadLineN(cfgFileName, 4)
    if nil != err {
        return nil, nil, 0, "", errors.New("Read Config file: " + err.Error())
    }
    rawUrl := lines[0]
    outFile := lines[1]
    size, err := strconv.ParseInt(lines[2], 10, 64)
	if nil != err {
		return nil, nil, 0, "", err
    }
    arr := downloader.Reduction(lines[3])
    store, err := downloader.NewStore(rawUrl, size, outFile, cfgFileName)
    return store, arr, size, outFile, err
}

/**
 * 构造函数
 */
func New(options *downloader.Options_t) (downloader.Task_t, error) {
    var store *downloader.Store_t = nil
    var linker []link_table.Line_t = nil
    var size int64 = 0
    fileName := ""
    parallelable := true

    // 一个远端资源对象
    httpResource, err := remote.NewHttpResource(options.RawUrl)
    if nil != err {
        return nil, err
    }

    // 如果存在配置文件，读取之
    if "" != options.ConfigFile {
        store, linker, size, fileName, err = resume(options.ConfigFile)
    } else {
        // 没有配置则拉取元信息
        err = httpResource.GetMeta()
        size = httpResource.Size()
        fileName = httpResource.Filename()

        // 若没有指定文件名，自动设定文件名
        if 0 < len(options.OutFile) || len(fileName) < 1 {
            fileName = options.OutFile
        }
        store, err = downloader.NewStore(options.RawUrl, size, fileName, "")
        parallelable = httpResource.Parallelable()
    }
    if nil != err {
        return nil, err
    }

    // 计算分片
    trd := 1
    block := size
    if parallelable {
        block, trd = GetBlockSlice(size, options.SgmTrd, options.Block)
    }
    // debug
    fmt.Printf("%s\nblock: %d\nthread: %d\r\n", store.FileInfo, block, trd)

    // 一个下载器实例
    return &HttpTask_t {
        BlockSlice_t: *NewBlockSlice(size, trd, block, linker),
        httpResource: httpResource,
        store:        store,
    }, nil
}

/**
 * 生产者
 */
func (this *HttpTask_t) Download() error {
    id := int64(0)
    if this.size < 1 {
        return errors.New("unknow origin file size")
    }
    pool := NewPool(this, this.sgmTrd)

    fmt.Fprintf(os.Stderr, ".")
    for ; id < int64(this.sgmTrd); id++ {
        // 入队
        foo := this.Pice()
        if nil == foo {
            break
        }
        foo.Id = id
        pool.Push(foo)
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
        this.Fill(ranger)
        foo := this.Pice()
        if nil != foo {
            id++
            foo.Id = id
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
        arr := this.done.ToArray()
        for i := 0; i < len(arr); i++ {
            fmt.Printf("\n{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
        }
    case "quit":
        arr := this.done.ToArray()
        this.store.Sync(arr)
        this.store.Close()
        os.Exit(0)
    }
}

func (this *HttpTask_t) EmitError(err error) {
    fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
    os.Exit(0)
}
