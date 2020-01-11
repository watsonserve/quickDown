package http_downloader

import (
    "github.com/watsonserve/quickDown/downloader"
    "github.com/watsonserve/quickDown/http_downloader/remote"
)

type BlockStorer struct {
    httpResource *remote.HttpResource
    store        *downloader.Store_t
}

/**
 * 构造函数
 */
func NewTransportor(store *downloader.Store_t, reader *remote.HttpResource) *BlockStorer {
    return &BlockStorer {
        httpResource: reader,
        store: store,
    }
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
            err = this.store.SendFileAt(rsp.Body, ranger.Start)
            rsp.Body.Close()
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
