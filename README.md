# GitMonitor
一个监测github项目进行增量备份的脚本，项目更新后，自动同步到本地文件夹，项目里删除的文件，本地文件不进行删减，可配置钉钉机器人，有新增文件进行提醒。


## 0x01  使用方法

### 1.1 编译

* Linux

```
GOOS=linux GOARCH=amd64 go build -o xxx xxx.go
```

* windows

```
GOOS=windows GOARCH=amd64 go build -o xxx.exe xxx.go
```

* macos

```
intel芯片 ： GOOS=darwin GOARCH=amd64 go build -o xxx xxx.go
m芯片 ： GOOS=darwin GOARCH=arm64 go build -o xxx xxx.go
```


