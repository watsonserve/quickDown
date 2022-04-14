package downloader

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/watsonserve/goutils"
	"github.com/watsonserve/quickDown/myio"
)

type Options_t struct {
	SgmTrd     int   // 线程数
	Block      int64 // 分片大小，单位：段
	OutPath    string
	OutFile    string
	RawUrl     string
	ConfigFile string
}

type Meta_t struct {
	SgmTrd  int   // 线程数
	Block   int64 // 分片大小，单位：段
	OutFile string
	RawUrl  string
	Size    int64
}

type Subject_t interface {
	GetMeta() *Meta_t
	CreateTask(store *Store_t, linker []goutils.Range_t) (Task_t, error)
}

type Task_t interface {
	Download() error
	// control
	Emit(cmd string)
	EmitError(err error)
}

func ListenSign(this Task_t) {
	signalChannel := make(chan os.Signal, 1)
	//监听所有信号
	signal.Notify(
		signalChannel,
		syscall.SIGHUP,
		syscall.SIGINT, // ctrl + C
		syscall.SIGQUIT,
		syscall.SIGTERM,
		// syscall.SIGTSTP,
		// syscall.SIGUSR1,
		// syscall.SIGUSR2,
	)

	for {
		sign := <-signalChannel
		cmd := ""
		switch sign {
		case syscall.SIGHUP:
			cmd = "hup"
		case syscall.SIGINT:
			cmd = "quit"
		case syscall.SIGQUIT:
			cmd = "quit"
		case syscall.SIGTERM:
			cmd = "quit"
		default:
			cmd = ""
		}
		this.Emit(cmd)
	}
}

func ListenCmd(this Task_t, inStream io.Reader) {
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
