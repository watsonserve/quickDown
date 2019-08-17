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
    BlockSlice_t
    transportor *BlockStorer
}

/**
 * 构造函数
 */
func New(file *os.File, reader *remote.HttpResource, block int64, sgmTrd int) *DownTask_t {
    this := &DownTask_t{
        BlockSlice_t: BlockSlice_t {},
        transportor: NewTransportor(file, reader),
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
    fmt.Printf("{size: %d, block: %d, thread: %d}\r\n", size, this.block, this.sgmTrd)
    return this
}

/**
 * 生产者
 */
func (this *DownTask_t) Download() error {
    offset := int64(0)
    id := int64(0)
    if this.size < 1 {
        return errors.New("unknow origin file size")
    }
    pool := NewPool(this.transportor, this.sgmTrd)

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
    this.transportor.Close()
    fmt.Fprintf(os.Stderr, "----\n{\"cost\": \"%ds\"}\n", time.Now().Unix() - this.startTime)

    return nil
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
