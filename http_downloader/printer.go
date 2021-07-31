package http_downloader

import (
    "fmt"
    "time"
    "strings"
)

/**
 * 统计
 * @param startTime 开始时间
 * @param doneSeek 已完成数据量
 * @param size 总体大小
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

type Outer struct {
    preLen int
}

func (this *Outer) Output(startTime int64, pace int64, size int64, sgmTrd int) {
    progress, velocity, unit, planTime := statistic(startTime, pace, size)
    put := fmt.Sprintf(
        "\r{\"finish\": \"%0.2f%%\", \"speed\": \"%0.2f%s/s\", \"planTime\": \"%ds, trd: %d\"}",
        progress, velocity, unit, planTime, sgmTrd,
    )
    putLen := len(put)
    delta := this.preLen - putLen
    this.preLen = putLen
    if 0 < delta {
        put += strings.Repeat(" ", delta)
    }
    fmt.Printf("%s", put)
}
