# GitMonitor
一个监测github项目进行增量备份的脚本，项目更新后，自动同步到本地文件夹，项目里删除的文件，本地文件不进行删减，可配置钉钉机器人，有新增文件进行提醒。


## 0x01  使用方法

### 1.1 钉钉机器人

<img width="637" alt="image" src="https://github.com/user-attachments/assets/2f204702-d5c8-4227-9435-fed75784d016" />


### 1.2 编译

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
intel 芯片 ： GOOS=darwin GOARCH=amd64 go build -o xxx xxx.go
m 芯片 ： GOOS=darwin GOARCH=arm64 go build -o xxx xxx.go
```

### 1.3 用法

```
gitmonitor -u https://github.com/xxx/xxx.git -p /xxx/xxx [-t 10s/10m/10h]

    -u：指定目标地址。
    -p：指定本地文件夹路径。
    -t：自定义检测频率，10s(10秒)/10m(10分钟)/10h(10小时)，可自定义，默认5分钟。

```


