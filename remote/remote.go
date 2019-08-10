package remote

import (
    "errors"
    "net/http"
	"net/url"
    "path"
    "strconv"
    "strings"
)

type HttpResource struct {
    parallelable  bool
    size          int64
    filename      string
    rawUrl        string
    connector     *HttpConn
}

/**
 * 构造函数
 */
func NewHttpResource(rawUrl string) (*HttpResource, error) {
	var err error
	for {
        var uri *url.URL
		uri, err = url.Parse(rawUrl)
		if nil != err {
			break
		}

        use_tls := 0
        if "https" == uri.Scheme {
            use_tls = 1
        }

        connector, err := NewHttpConn(uri.Host, use_tls)
        if nil != err {
            return nil, err
        }

        // 默认使用url的文件名
		this := &HttpResource{
            filename:  path.Base(uri.Path),
            rawUrl:    rawUrl,
            connector: connector,
		}
		return this, nil
	}
	return nil, err
}


func (this *HttpResource) loadMeta() (*http.Header, error) {
    cli, err := this.NewHttpReader()
    if nil != err {
		return nil, err
    }
    err = cli.Conn(true)
	if nil != err {
		return nil, err
    }
    req := cli.req
    req.Method = "HEAD"
    req.Header.Set("Connection", "Close")
    resp, err := cli.Send(req)
    if nil != err {
        return nil, err
    }
    resp.Body.Close()
    cli.Close()
	// 应答错误
	if 200 != resp.StatusCode {
		return nil, errors.New(resp.Status)
	}

	return &resp.Header, nil
}

/**
 * 试着获取远端信息，文件名和内容长度
 * @return {error}
 */
func (this *HttpResource) GetMeta() error {
    header, err := this.loadMeta()
    if nil != err {
        return err
    }

	contentLength := header.Get("Content-Length")
	this.size, err = strconv.ParseInt(contentLength, 10, 64)
	if nil != err {
		return err
    }

	acceptRanges := header.Get("Accept-Ranges")
	this.parallelable = "" != acceptRanges && "none" != acceptRanges

    // 优先使用应答头里的文件名
    filename := header.Get("Content-Disposition")
    if len(filename) < 1 {
        return nil
    }
    foo := strings.Split(filename, "filename=")
    if len(foo) < 1 {
        return nil
    }
    filename = foo[1]
    if '"' == filename[0] {
        filename = filename[1 : len(filename) - 1]
    }
    if 0 < len(filename) {
        this.filename = filename
    }

	return nil
}

func (this *HttpResource) Filename() string {
    return this.filename
}

func (this *HttpResource) Size() int64 {
    return this.size
}

func (this *HttpResource) Parallelable() bool {
    return this.parallelable
}
