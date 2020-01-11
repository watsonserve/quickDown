package http_downloader

import (
    "fmt"
    "github.com/watsonserve/quickDown/data_struct"
    "os"
    "time"
)

const MAX_THREAD_COUNT int64 = 256
const MAX_BLOCK_SIZE int64 = 1 << 20
const DEFAULT_BLOCK_SIZE int64 = 65536

var units = []string{
    "B", "KB", "MB", "GB",
}

type BlockSlice_t struct {
    sgmTrd        int
    size          int64
    block         int64
    doneSeek      int64
    startTime     int64
    completedLink *data_struct.TaskLink
}
/**
 * 分片
 * @params {int64} 总大小
 * @params {int}   线程数
 * @params {int64} 块大小
 * @return {int}   线程数
 * @return {int64} 块大小
 */
func GetBlockSlice(size int64, intTrd int, block int64) (int64, int) {
    maxTrd := MAX_THREAD_COUNT
    maxBlock := MAX_BLOCK_SIZE
    defaultBlock := DEFAULT_BLOCK_SIZE
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
        if size < block {
            trd = 1
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
    return block, int(trd)
}

func NewBlockSlice(size int64, trd int, block int64) *BlockSlice_t {
    return &BlockSlice_t {
        size:          size,
        block:         block,
        sgmTrd:        trd,
        doneSeek:      0,
        startTime:     time.Now().Unix(),
        completedLink: data_struct.NewList(nil),
    }
}

func (this *BlockSlice_t) Cut(start int64) *Range_t {
    max := this.size - 1
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

func (this *BlockSlice_t) Pice() *Range_t {
    return this.Cut(this.doneSeek)
}

func (this *BlockSlice_t) Fill(ranger *Range_t) bool {
    this.doneSeek += this.block
    this.completedLink.Mount(ranger.Start, ranger.End)
    // 统计
    progress, velocity, unit, planTime := statistic(this.startTime, this.doneSeek, this.size)
    fmt.Fprintf(
        os.Stderr,
        "{\"finish\": \"%0.2f%%\", \"speed\": \"%0.2f%s/s\", \"planTime\": \"%ds\"}\n",
        progress, velocity, unit, planTime,
    )
    return this.size < this.doneSeek
}

/**
 * 非线程安全
 */
func (this *BlockSlice_t) Check() []data_struct.Line_t {
    return this.completedLink.ToArray()
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

    planTime := -1
    if 0 != doneSeek {
        planTime = int((size - doneSeek) / int64(velocity))
    }

    for unit_p = 0; 1024 < velocity; unit_p++ {
        velocity /= 1024
    }
    return progress, velocity, units[unit_p], planTime
}
