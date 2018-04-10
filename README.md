# quickDown
quick http downloader

## 机理
  http协议的分片下载
  go语言的多协程并发

## 使用方法
  quickDown -b 8 -t 128 https://xxx.xxx.com/xxxx.zip
  -b 指定分片大小，以段为单位，每段64K
  -t 指定协程数，最大128个协程

## 编译
  #先设定GOPATH
  go install quickDown
