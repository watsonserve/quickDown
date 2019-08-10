package http_downloader

type Range_t struct {
    Id    int64
    Start int64
    End   int64
    Err   error
}

type Pool_t struct {
    taskPipe   chan *Range_t
    notifyPipe chan *Range_t
}

func (this *Pool_t) Push(foo *Range_t) {
    this.taskPipe <- foo
}

func (this *Pool_t) Wait() *Range_t {
    ret := <- this.notifyPipe
    return ret
}
