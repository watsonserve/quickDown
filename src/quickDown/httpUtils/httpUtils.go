package httpUtils

import (
    "crypto/tls"
    "net/http"
)

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
