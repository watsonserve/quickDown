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
	httpResource *http_remote.HttpResource
	store        *downloader.Store_t
}

type Translater struct {
	store         *downloader.Store_t
	httpRequester *http_remote.HttpReader
}

func resume(cfgFileName string) (*downloader.Store_t, []goutils.Range_t, int64, string, error) {
	lines, err := goutils.ReadLineN(cfgFileName, 4)
	if nil != err {
		return nil, nil, 0, "", errors.New("Read Config file: " + err.Error())
	}
	rawUrl := lines[0]
	outFile := lines[1]
	size, err := strconv.ParseInt(lines[2], 10, 64)
	if nil != err {
		return nil, nil, 0, "", err
	}
	arr := downloader.Reduction(lines[3])
	store, err := downloader.NewStore(rawUrl, size, outFile, cfgFileName)
	return store, arr, size, outFile, err
}

/**
 * 构造函数
 */
func New(options *downloader.Options_t) (downloader.Task_t, error) {
	var store *downloader.Store_t = nil
	var linker []goutils.Range_t = nil
	var size int64 = 0
	fileName := ""
	parallelable := true

	// 一个远端资源对象
	httpResource, err := http_remote.NewHttpResource(options.RawUrl)
	if nil != err {
		return nil, err
	}

	// 如果存在配置文件，读取之
	if "" != options.ConfigFile {
		store, linker, size, fileName, err = resume(options.ConfigFile)
	} else {
		// 没有配置则拉取元信息
		err = httpResource.GetMeta()
		size = httpResource.Size()
		fileName = httpResource.Filename()

		// 若没有指定文件名，自动设定文件名
		if 0 < len(options.OutFile) || len(fileName) < 1 {
			fileName = options.OutFile
		}
		store, err = downloader.NewStore(options.RawUrl, size, fileName, "")
		parallelable = httpResource.Parallelable()
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
		Outer: Outer{
			preLen: 0,
		},
		httpResource: httpResource,
		store:        store,
	}, nil
}

/**
 * 生产者
 */
func (this *HttpTask_t) Download() error {
	id := int64(0)
	if this.size < 1 {
		return errors.New("unknow origin file size")
	}
	threadPool := goutils.NewPool(this.WorkerInit, this.sgmTrd)

	fmt.Fprintf(os.Stderr, ".")
	for ; id < int64(this.sgmTrd); id++ {
		// 入队
		foo := this.Pice()
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
		this.Fill(ranger.(*Range_t))
		this.Output(this.startTime, this.pace, this.size, this.sgmTrd)
		foo := this.Pice()
		if nil != foo {
			id++
			foo.Id = id
		}
		threadPool.Push(foo)
	}
	this.store.Close()
	fmt.Fprintf(os.Stderr, "----\n{\"cost\": \"%ds\"}\n", time.Now().Unix()-this.startTime)

	return nil
}

func (this *HttpTask_t) WorkerInit() (*Translater, error) {
	httpRequester, err := this.httpResource.NewHttpReader()
	if nil != err {
		return nil, err
	}
	return &Translater{
		store:         this.store,
		httpRequester: httpRequester,
	}, err
}

func (this *HttpTask_t) Emit(cmd string) {
	switch cmd {
	case "check":
		arr := this.done.ToArray()
		fmt.Printf("\n")
		for i := 0; i < len(arr); i++ {
			fmt.Printf("{start: %d, end: %d}\n", arr[i].Start, arr[i].End)
		}
	case "quit":
		arr := this.done.ToArray()
		this.store.Sync(arr)
		this.store.Close()
		os.Exit(0)
	}
}

func (this *HttpTask_t) EmitError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	os.Exit(0)
}

/**
 * 消费者
 * 传入nil使线程结束
 */
func (translater *Translater) Worker(params goutils.Any_t) goutils.Any_t {
	ranger := params.(*Range_t)
	rsp, err := translater.httpRequester.Read(ranger.Start, ranger.End, 3)
	if nil == err {
		err = translater.store.SendFileAt(rsp.Body, ranger.Start)
		rsp.Body.Close()
	}
	ranger.Err = err
	return ranger
}

func (translater *Translater) Destroy() {}
