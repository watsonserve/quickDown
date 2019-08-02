package main

import (
    "errors"
    "os"
    "strconv"
)

type Options_t struct {
    SgmTrd  int    // 线程数
    Block   int64  // 分片大小，单位：段
    OutFile string
    RawUrl  string
}

func getOptions() (*Options_t, error) {
    var err error
    argv := os.Args
    argc := len(argv)

    // 没有给出任何参数
    if 2 > argc {
        help()
        return nil, errors.New("")
    }
    this := &Options_t {
        SgmTrd: 1,
        Block: 1,
        OutFile: "",
        RawUrl: "",
    }

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
                this.Block, err = strconv.ParseInt(nextArg, 0, 0)
                if nil != err {
                    return nil, errors.New("ERROR block should be a intger")
                }
            case 'o':
                this.OutFile = nextArg
            case 't':
                trd, err := strconv.ParseInt(nextArg, 0, 0)
                if nil != err {
                    return nil, errors.New("ERROR number of thread should be a intger")
                }
                this.SgmTrd = int(trd)
            default:
                return nil, errors.New("ERROR unknow option " + nextArg)
            case 'h':
                return nil, errors.New("")
            }
        } else {
            // 预下载文件地址
            this.RawUrl = argp
        }
    }
    return this, nil
}
