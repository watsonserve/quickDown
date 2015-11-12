package downloaders

import(
    "fmt"
    "strings"
    "net"
    "net/url"
    "bufio"
    "strconv"
)
type Downloader interface {
    Load(conn *net.Conn, req *map[string]string) (*map[string]string, error)
}

type HttpDownloader struct {
    request map[string]string
    host string
}

func NewHttpDownloader(uri *url.URL) *HttpDownloader {
    path := uri.Path
    this := &HttpDownloader{}
    this.request = make(map[string]string)
    this.request["method"] = "HEAD"
    this.request["protocol"] = "HTTP/1.1"
    this.request["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.80 Safari/537.36"
    this.request["Accept"] = "*/*"
    this.request["Accept-Encoding"] = "gzip"
    this.request["Connection"] = "keep-alive"
    this.request["Cache-Control"] = "max-age=0"
    if "" != uri.RawQuery {
        path += "?" + uri.RawQuery
    }
    if "" != uri.Fragment {
        path += "#" + uri.Fragment
    }
    this.request["path"] = path
    this.host = uri.Host
    if 2 != len(strings.Split(this.host, ":")) {
        this.host += ":80"
    }
    return this
}

func (this *HttpDownloader) Load(conn *net.Conn, req *map[string]string) (*map[string]string, error) {
    oneReq := this.request
    if nil != req {
        for v := range *req {
            oneReq[v] = (*req)[v]
        }
    }
    requestText := oneReq["method"] + " " + oneReq["path"] + " " + oneReq["protocol"] + "\r\n"
    for v := range oneReq {
        if "method" == v || "path" == v || "protocol" == v {
            continue
        }
        requestText += v + ": " + oneReq[v] + "\r\n"
    }
    requestText += "\r\n"
    if nil == conn {
        connLocal, err := net.Dial("tcp", this.host)
        if nil != err {
            return nil, err
        }
        defer connLocal.Close()
        conn = &connLocal
    }
    fmt.Fprintf((*conn), requestText)    // send http request
    fmt.Println(requestText)
    reader := bufio.NewReader(*conn)
    line, err := reader.ReadString('\n')
    if nil != err {
        return nil, err
    }
    response := make(map[string]string)
    kv := strings.SplitN(line, " ", 3)
    response["protocol"] = kv[0]
    response["state"] = kv[1]
    response["message"] = kv[2]
    for {
        line, err = reader.ReadString('\n')
        if nil != err {
            return nil, err
        }
        if "" == strings.TrimSpace(line) {
            break
        }
        kv = strings.Split(line, ": ")
        if 2 != len(kv) {
            return nil, err
        }
        response[kv[0]] = strings.TrimSpace(kv[1])
    }
    if "HEAD" != oneReq["method"] {
        recvsize, err := strconv.ParseInt(response["Content-Length"], 0, 0)
        fmt.Printf("recvsize: %d\n", recvsize)
        response["content"] = ""
        for {
            line, err = reader.ReadString('\x00')
            response["content"] += line
            if 0 == len(line) || int(recvsize) == len(response["content"]) {
                break
            }
            if nil != err {
                return nil, err
            }
        }
        response["Content-Length"] = strconv.FormatInt(int64(len(response["content"])), 10)
    }
    return &response, nil
}