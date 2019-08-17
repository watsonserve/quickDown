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

type Poolable interface {
    Worker(taskPipe chan *Range_t, notifyPipe chan *Range_t)
}

func NewPool(poolable Poolable, size int) *Pool_t {
    taskPipe := make(chan *Range_t, size)
    notifyPipe := make(chan *Range_t, size << 1)

    this := &Pool_t {
        taskPipe:   taskPipe,
        notifyPipe: notifyPipe,
    }

    for i := 0; i < size; i++ {
        go poolable.Worker(taskPipe, notifyPipe)
    }

    return this
}

func (this *Pool_t) Push(foo *Range_t) {
    this.taskPipe <- foo
}

func (this *Pool_t) Wait() *Range_t {
    ret := <- this.notifyPipe
    return ret
}
