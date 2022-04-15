package http_downloader

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/watsonserve/goutils"
	"github.com/watsonserve/quickDown/downloader"
	"github.com/watsonserve/quickDown/http_remote"
)

type HttpTask_t struct {
	BlockSlice_t
	downloader.Outer
	url   string
	store *downloader.Store_t
}

type Translater struct {
	http_remote.HttpReader
	store *downloader.Store_t
}

/**
 * 消费者
 * 传入nil使线程结束
 */
func (translater *Translater) Work(params goutils.Any_t) goutils.Any_t {
	ranger := params.(*Range_t)
	rsp, err := translater.HttpReader.Read(ranger.Start, ranger.End, 3)
	if nil == err {
		err = translater.store.SendFileAt(rsp.Body, ranger.Start)
		rsp.Body.Close()
	}
	ranger.Err = err
	return ranger
}

func (translater *Translater) Destroy() {}

func general(url string, outFile string) (*downloader.Store_t, bool, int64, error) {
	var size int64 = 0
	var store *downloader.Store_t = nil
	parallelable := false
	// 一个远端资源对象
	httpResource, err := http_remote.NewHttpResource(url)
	if nil == err {
		// 没有配置则拉取元信息
		err = httpResource.GetMeta()
	}

	if nil == err {
		size = httpResource.Size()
		fileName := httpResource.Filename()
		url = httpResource.Url()

		// 若没有指定文件名，自动设定文件名
		if 0 < len(outFile) || len(fileName) < 1 {
			fileName = outFile
		}
		store, err = downloader.NewStore(url, size, fileName, "")
		parallelable = httpResource.Parallelable()
	}

	return store, parallelable, size, err
}

func resume(cfgFileName string) (*downloader.Store_t, []goutils.Range_t, int64, error) {
	lines, err := goutils.ReadLineN(cfgFileName, 4)
	if nil != err {
		return nil, nil, 0, errors.New("Read Config file: " + err.Error())
	}
	rawUrl := lines[0]
	outFile := lines[1]
	size, err := strconv.ParseInt(lines[2], 10, 64)
	if nil != err {
		return nil, nil, 0, err
	}
	arr := downloader.Reduction(lines[3])
	store, err := downloader.NewStore(rawUrl, size, outFile, cfgFileName)
	return store, arr, size, err
}

/**
 * 构造函数
 */
func New(options *downloader.Options_t) (downloader.Task_t, error) {
	var store *downloader.Store_t = nil
	var linker []goutils.Range_t = nil
	var size int64 = 0
	var err error = nil

	parallelable := true
	url := options.RawUrl

	// 如果存在配置文件，读取之
	if "" != options.ConfigFile {
		store, linker, size, err = resume(options.ConfigFile)
	} else {
		store, parallelable, size, err = general(url, options.OutFile)
	}
	if nil != err {
		return nil, err
	}

	// 计算分片
	trd := 1
	block := size
	if parallelable {
		block, trd = GetBlockSlice(size, options.SgmTrd, options.Block)
	}
	// debug
	fmt.Printf("%s\nblock: %d\nthread: %d\r\n", store.FileInfo, block, trd)

	// 一个下载器实例
	return &HttpTask_t{
		BlockSlice_t: *NewBlockSlice(size, trd, block, linker),
		Outer:        *downloader.NewOuter(0),
		url:          url,
		store:        store,
	}, nil
}

/**
 * 生产者
 */
func (that *HttpTask_t) Download() error {
	id := int64(0)
	if that.size < 1 {
		return errors.New("unknow origin file size")
	}
	threadPool := goutils.NewPool(that.WorkerInit, that.sgmTrd)

	fmt.Fprintf(os.Stderr, ".")
	for ; id < int64(that.sgmTrd); id++ {
		// 入队
		foo := that.Pice()
		if nil == foo {
			break
		}
		foo.Id = id
		threadPool.Push(foo)
	}

	fmt.Fprintf(os.Stderr, ".\n")
	// 任务发放完成 并且 全部线程均已关闭
	for 0 < threadPool.Count() {
		ranger := threadPool.Wait()
		// 线程退出
		if nil == ranger {
			continue
		}
		that.Fill(ranger.(*Range_t))
		that.Output(that.startTime, that.pace, that.size, that.sgmTrd)
		foo := that.Pice()
		if nil != foo {
			id++
			foo.Id = id
		}
		threadPool.Push(foo)
	}
	that.store.Close()
	fmt.Fprintf(os.Stderr, "----\n{\"cost\": \"%ds\"}\n", time.Now().Unix()-that.startTime)

	return nil
}

func (that *HttpTask_t) WorkerInit() (goutils.Worker, error) {
	httpRequester, err := http_remote.NewHttpReader(that.url)
	if nil != err {
		return nil, err
	}
	return &Translater{
		HttpReader: *httpRequester,
		store:      that.store,
	}, err
}

func (that *HttpTask_t) Emit(cmd string) {
	switch cmd {
	case "check":
		arr := that.done.ToArray()
		fmt.Printf("\n")
		for i := 0; i < len(arr); i++ {
			fmt.Printf("{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
		}
	case "quit":
		arr := that.done.ToArray()
		that.store.Sync(arr)
		that.store.Close()
		os.Exit(0)
	}
}

func (that *HttpTask_t) EmitError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	os.Exit(0)
}
