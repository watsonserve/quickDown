package http_downloader

import (
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "github.com/watsonserve/quickDown/remote"
)

type BlockStorer struct {
    httpResource *remote.HttpResource
    store        *os.File
}

/**
 * 构造函数
 */
func NewTransportor(file *os.File, reader *remote.HttpResource) *BlockStorer {
    return &BlockStorer {
        httpResource: reader,
        store: file,
    }
}

/**
 * 线程安全
 */
func (this *BlockStorer) SendFileAt(rs io.ReadCloser, w_off int64) error {
	buf, err := ioutil.ReadAll(rs)
	if nil != err {
		return err
    }
    rs.Close()
    bugLen := len(buf)
	length, err := this.store.WriteAt(buf, w_off)
	if nil != err {
		return err
    }
    
	if bugLen != length {
		return errors.New(fmt.Sprintf("write faild, len: %d", length))
	}
	return nil
}

/**
 * 消费者
 * 传入nil使线程结束
 */
func (this *BlockStorer) Worker(taskPipe chan *Range_t, notifyPipe chan *Range_t) {
    httpRequester, err := this.httpResource.NewHttpReader()
    if nil != err {
        notifyPipe <- nil
        return
    }
    for ranger := <- taskPipe; nil != ranger; ranger = <- taskPipe {
        rsp, err := httpRequester.Read(ranger.Start, ranger.End, 3)
        if nil == err {
            err = this.SendFileAt(rsp.Body, ranger.Start)
        }
        ranger.Err = err
        notifyPipe <- ranger
    }
    // 得到的任务为nil则传出nil
    notifyPipe <- nil
}

func (this *BlockStorer) Close() {
    this.store.Close()
}
