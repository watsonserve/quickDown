package main

/*/ #cgo CFLAGS: -O3 */

import (
	//	"C"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"github.com/watsonserve/quickDown/httpDownloader"
	"strconv"
	"syscall"
)

func help() {
	fmt.Fprintln(os.Stderr, "version 1.0 License GPL2.0")
	fmt.Fprintln(os.Stderr, "(C) watsonserve.com made by James Watson\n")
	fmt.Fprintln(os.Stderr, "use [-b blockSize|-t sumOfThread|-o outputFile|--stdout] url")
	fmt.Fprintln(os.Stderr, "     -b block Size, will be integer multiples of 64K(max: 16). default is 1 multiple")
	fmt.Fprintln(os.Stderr, "     -t sum Of Thread. default is 1, max: 128")
	fmt.Fprintln(os.Stderr, "     -o output file name. auto set")
	fmt.Fprintln(os.Stderr, "     -h show this help information\n")
}

//export Http_download
// func Http_download(urlStr *C.char, outFile *C.char, block int64, sgmTrd int) C.int {
// 	downloader := httpDownloader.New(C.GoString(urlStr), C.GoString(outFile), block, sgmTrd)
// 	ret := downloader.Download()
// 	if nil != ret {
// 		return -1
// 	}
// 	return 0
// }

func signalListener() (chan os.Signal) {
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
	return signalChannel
}

func main() {
	argv := os.Args
	argc := len(argv)

	// 没有给出任何参数
	if 2 > argc {
		help()
		return
	}

	var block int64  // 分片大小，单位：段
	var sgmTrd int64 // 线程数
	var err error
	var uri *url.URL

	block = 1
	sgmTrd = 1
	outFile := ""
	urlStr := ""

	// get option
	for i := 1; i < argc; i++ {
		argp := argv[i]
		nextArg := ""
		if i + 1 < argc {
			nextArg = argv[i + 1]
		}

		// 一个选项
		if '-' == argp[0] {
			switch argp[1] {
			case 'b':
				block, err = strconv.ParseInt(nextArg, 0, 0)
				if nil != err {
					fmt.Fprintln(os.Stderr, "ERROR block should be a intger")
					return
				}
			case 'o':
				outFile = nextArg
			case 't':
				sgmTrd, err = strconv.ParseInt(nextArg, 0, 0)
				if nil != err {
					fmt.Fprintln(os.Stderr, "ERROR number of thread should be a intger")
					return
				}
			default:
				fmt.Fprintln(os.Stderr, "ERROR unknow option "+nextArg)
				fallthrough
			case 'h':
				help()
				return
			}
		} else {

			// 预下载文件地址
			urlStr = argp
		}
	}

	// debug
	fmt.Fprintln(os.Stderr, "url: " + urlStr)

	uri, err = url.Parse(urlStr)
	if nil != err {
		fmt.Fprintln(os.Stderr, "ERROR url")
		return
	}

	sigChannel := signalListener()

	// filter the protocol
	switch uri.Scheme {
	case "thunder":
		fmt.Fprintln(os.Stderr, "parse base64")
		return
	case "http":
		fallthrough
	case "https":
		downloader, err := httpDownloader.New(urlStr, block, int(sgmTrd))
		if nil != err {
            fmt.Fprintln(os.Stderr, err)
			return
		}
		// 获取远端文件信息
		err, fileName := downloader.OriginInfo()
		if nil != err {
            fmt.Fprintln(os.Stderr, err)
			return
		}
		// 若没有指定文件名，自动设定文件名
		if len(outFile) < 1 && 0 < len(fileName) {
			outFile = fileName
		}

		// 创建本地文件
		outStream, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE, 0666)
		if nil != err {
                        fmt.Fprintln(os.Stderr, err)
			return
		}
		go downloader.On(sigChannel)
		err = downloader.Download(outStream)
		break
	default:
		fmt.Fprintf(os.Stderr, "ERROR unsuppored protocol %s\n", uri.Scheme)
		return
	}
	if nil != err {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}

	return
}
