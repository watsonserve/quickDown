package httpUtils

import (
	"errors"
    "crypto/tls"
	"io"
    "net/http"
	"net/url"
	"path"
    "strconv"
    "strings"
)

type Resp_t struct {
	Length int64
	Body   io.ReadCloser
}

type HTTPRequest struct {
	uri       *url.URL
	url       string
	useTls    bool
}

func Dail(urlStr string, method string, headers *http.Header, useTls bool) (*http.Response, error) {
    client := &http.Client{}
    if useTls {
        client.Transport = &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }
    }
    req, err := http.NewRequest(method, urlStr, nil)
    if nil != err {
        return nil, err
    }
    if nil != headers {
        req.Header = *headers
    }
    return client.Do(req)
}

/**
 * 构造函数
 */
 func New(url_raw string) *HTTPRequest {
	uri, err := url.Parse(url_raw)
	if nil != err {
		return nil
	}
	this := &HTTPRequest{
        url:       url_raw,
		uri:       uri,
		useTls:    "https" == uri.Scheme,
	}
	return this
}

/**
 * 请求一个分片
 * @params {int64}   start
 * @params {int64}   end
 * @return {*Resp_t}
 * @return {error}
 */
 func (this *HTTPRequest) RequestRange(start int64, end int64, repeat int) (*Resp_t, error) {
	headers := &http.Header{}
    headers.Add("Range", "bytes="+strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10))

    resp, err := Dail(this.url, "GET", headers, this.useTls)
    
    for ; nil != err && 0 < repeat; {
        repeat--
        resp, err = Dail(this.url, "GET", headers, this.useTls)
    }
	if nil != err {
		return nil, err
	}
	// 应答错误
	if 2 != resp.StatusCode/100 {
		return nil, errors.New(resp.Status)
	}
	// 直接返回流
	return &Resp_t{
		Length: 0,
		Body:   resp.Body,
	}, nil
}


/**
 * 试着获取远端信息，文件名和内容长度
 * @return {error}
 */
 func (this *HTTPRequest) OriginInfo() (error, bool, int64, string) {
	resp, err := Dail(this.url, "HEAD", nil, this.useTls)
	if nil != err {
		return err, false, 0, ""
	}
	// 应答错误
	if 200 != resp.StatusCode {
		return errors.New(resp.Status), false, 0, ""
	}
	resp.Body.Close()

	acceptRanges := resp.Header.Get("Accept-Ranges")
	contentLength := resp.Header.Get("Content-Length")

	retCanRange := "" != acceptRanges && "none" != acceptRanges
	i, err := strconv.ParseInt(contentLength, 10, 64)
	if nil != err {
		return err, false, 0, ""
	}
	retContentLength := i

    // 优先使用应答头里的文件名
    retFileName := resp.Header.Get("Content-Disposition")
    if 0 < len(retFileName) {
        foo := strings.Split(retFileName, "filename=")
        if 0 < len(foo) {
            retFileName = foo[1]
        }
        if '"' == retFileName[0] {
            retFileName = retFileName[1 : len(retFileName) - 1]
        }
    }
    // 使用url的文件名
    if len(retFileName) < 1 {
        retFileName = path.Base(this.uri.Path)
    }

	return nil, retCanRange, retContentLength, retFileName
}
