package httpDownloader

const MAX_THREAD_COUNT int64 = 256
const MAX_BLOCK_SIZE int64 = 1 << 20
const DEFAULT_BLOCK_SIZE int64 = 65536

type BlockSlice_t struct {
    size          int64
    doneSeek      int64
    block         int64
    sgmTrd        int
}
/**
 * 分片
 * @params {int64} 总大小
 * @params {int}   线程数
 * @params {int64} 块大小
 * @return {int}   线程数
 * @return {int64} 块大小
 */
func NewBlockSlice(size int64, intTrd int, block int64) *BlockSlice_t {
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
    return &BlockSlice_t {
        size:          size,
        doneSeek:      0,
        block:         block,
        sgmTrd:        int(trd),
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
    return this.size < this.doneSeek
}
