package main

import (
    "errors"
    "strconv"
    "github.com/watsonserve/goutils"
)

type Options_t struct {
    SgmTrd  int    // 线程数
    Block   int64  // 分片大小，单位：段
    OutFile string
    RawUrl  string
}

func getOptions() (*Options_t, error) {
    var err error
    // 获取命令行参数
    allOptions := []goutils.Option{
        {
            Opt: 'b',
            Option: "block",
            HasParams: true,
        },
        {
            Opt: 't',
            Option: "thread",
            HasParams: true,
        },
        {
            Opt: 'o',
            Option: "output",
            HasParams: true,
        },
        {
            Opt: 'h',
            Option: "help",
            HasParams: false,
        },
    }
    optionMap, urls := goutils.GetOptions(allOptions)

    _, ok := optionMap["help"]
    if ok {
        return nil, errors.New("")
    }

    // block
    var block int64 = 1
    strBlock, ok := optionMap["block"]
    if ok {
        block, err = strconv.ParseInt(strBlock, 0, 0)
        if nil != err {
            return nil, errors.New("Error block should be a intger")
        }
    }

    // thread
    var thread int64 = 1
    strThread, ok := optionMap["thread"]
    if ok {
        thread, err = strconv.ParseInt(strThread, 0, 0)
        if nil != err {
            return nil, errors.New("Error thread should be a intger")
        }
    }

    // url
    if 1 != len(urls) {
        return nil, errors.New("Error download url")
    }

    return &Options_t{
        SgmTrd: int(thread),
        Block: block,
        OutFile: optionMap["output"],
        RawUrl: urls[0],
    }, nil
}
