package http_downloader

import (
	"fmt"
	"os"
	"time"

	"github.com/watsonserve/goutils"
)

const MAX_THREAD_COUNT int64 = 256
const MAX_BLOCK_SIZE int64 = 1 << 20
const DEFAULT_BLOCK_SIZE int64 = 65536

var units = []string{
	"B", "KB", "MB", "GB",
}

type BlockSlice_t struct {
	sgmTrd    int
	size      int64
	block     int64
	pace      int64
	startTime int64
	done      *goutils.RangeLink_t
	todo      *goutils.RangeLink_t
}

type Range_t struct {
	Id    int64
	Start int64
	End   int64
	Err   error
}

/**
 * 计算分片计划
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

func NewBlockSlice(size int64, trd int, block int64, linker []goutils.Range_t) *BlockSlice_t {
	done := goutils.NewRangeLink(linker)
	todo := done.Converse(0, size)

	return &BlockSlice_t{
		size:      size,
		block:     block,
		sgmTrd:    trd,
		pace:      0,
		startTime: time.Now().Unix(),
		done:      done,
		todo:      todo,
	}
}

func (this *BlockSlice_t) Pice() *Range_t {
	front := this.todo.Front()
	if nil == front {
		return nil
	}
	start := front.Start
	end := start + this.block

	if front.End <= end {
		end = front.End
		this.todo.Pop()
	} else {
		front.Start = end
	}
	return &Range_t{
		Start: start,
		End:   end,
	}
}

/**
 * 挂载到完成链表
 */
func (this *BlockSlice_t) Fill(ranger *Range_t) {
	// 错误
	if nil != ranger.Err {
		fmt.Fprintf(os.Stderr, "\nError in worker(range: %d-%d): %s\n", ranger.Start, ranger.End, ranger.Err.Error())
		this.todo.Mount(ranger.Start, ranger.End)
		return
	}
	this.done.Mount(ranger.Start, ranger.End)
	this.pace += ranger.End - ranger.Start
}

/**
 * 将完成链表输出一份数组格式的快照（非线程安全）
 */
func (this *BlockSlice_t) Check() []goutils.Range_t {
	return this.done.ToArray()
}
