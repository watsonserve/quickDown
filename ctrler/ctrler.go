package ctrler

import (
    "github.com/watsonserve/quickDown/myio"
    "io"
    "os"
    "os/signal"
    "syscall"
)

type EventCommiter interface {
    Emit(cmd string)
    EmitError(err error)
}

func ListenSign(this EventCommiter) {
    signalChannel := make(chan os.Signal)
    //监听所有信号
    signal.Notify(
        signalChannel,
        syscall.SIGHUP,
        syscall.SIGINT,
        syscall.SIGQUIT,
        syscall.SIGTERM,
        // syscall.SIGTSTP,
        // syscall.SIGUSR1,
        // syscall.SIGUSR2,
    )

    for {
        sign := <- signalChannel
        cmd := ""
        switch sign {
        case syscall.SIGHUP:
            cmd = "hup"
        case syscall.SIGINT:
            cmd = "int"
        case syscall.SIGQUIT:
            cmd = "quit"
        case syscall.SIGTERM:
            cmd = "term"
        default:
            cmd = ""
        }
        this.Emit(cmd)
    }
}

func ListenCmd(this EventCommiter, inStream io.Reader) {
    // 监听标准输入流
    readStream := myio.InitReadStream(inStream)
    for {
        command, err := readStream.ReadLine()
        if nil != err {
            this.EmitError(err)
            return
        }
        this.Emit(command)
    }
}

/*********************/

type EventListener interface {
    On(cmd string)
    OnError(err error)
}

type Ctrler struct {

}

func New() *Ctrler {
    return &Ctrler {}
}
func (this *Ctrler) Emit(cmd string) {}

func (this *Ctrler) EmitError(err error) {}
