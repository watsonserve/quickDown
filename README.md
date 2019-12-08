# quickDown
quick http downloader

## 机理
  http协议的分片下载
  go语言的多协程并发

## 使用方法
 quickDown -b 8 -t 128 https://xxx.xxx.com/xxxx.zip
 * -o 指定文件名，如果不指定文件名，优先使用应答头中指定的文件名，其次使用url中的文件名
 * -b 指定分片大小，以段为单位，每段64K
 * -t 指定协程数，最大128个协程

## 编译
  go install quickDown

## 任务描述格式
#### header
  | 行号 | 内容 |
  |---|---|
  |0|远端URL|
  |1|文件名|
#### body
  | 64bit | 64bit |
  | --- | --- |
  | 数组长度 | 0 |
  | start_offset | length |
