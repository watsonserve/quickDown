package httpDownloader

import (
    "fmt"
	"io"
    "os"
    "github.com/watsonserve/quickDown/link"
    "github.com/watsonserve/quickDown/myio"
    "time"
)

var units = []string{
    "B", "KB", "MB", "GB",
}

type IOProcess_t struct {
    BlockSlice_t
    completedLink     *link.TaskLink
    startTime         int64
    store             *os.File
}

/**
 * 构造函数
 */
func NewIOProcess(blockSlice *BlockSlice_t, outStream *os.File) *IOProcess_t {
    this := &IOProcess_t{
        completedLink: link.New(nil),
        startTime: time.Now().Unix(),
        store: outStream,
    }
    this.BlockSlice_t = *blockSlice
    return this
}

func (this *IOProcess_t) Write(rs io.ReadCloser, offset int64) error {
    err := myio.SendFileAt(rs, this.store, offset)
    rs.Close()
    return err
}

func (this *IOProcess_t) Record(ranger *Range_t) {
    this.completedLink.Mount(ranger.Start, ranger.End)
    // 统计
    progress, velocity, unit, planTime := this.statistic()
    fmt.Fprintf(
        os.Stderr,
        "{\"finish\": \"%0.2f%%\", \"speed\": \"%0.2f%s/s\", \"planTime\": \"%ds\"}\n",
        progress, velocity, unit, planTime,
    )
}

func (this *IOProcess_t) Check() []link.Line_t {
    return this.completedLink.ToArray()
}

/**
 * 统计
 */
func (this *IOProcess_t) statistic() (float32, float32, string, int) {
    startTime := this.startTime
    doneSeek := this.doneSeek
    size := this.size
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
