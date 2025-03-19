# GitMonitor
一个监测 github 项目进行增量备份的脚本，适用于 wiki 备份。监测项目更新，新增文件后，自动同步到本地文件夹，项目若删除文件，本地文件不进行删减，可配置钉钉机器人，有新增文件进行提醒。


## 0x01  使用方法

### 1.1 钉钉机器人

从钉钉机器人设置中复制 webhook 和 加签秘钥，配置在脚本 GitMonitor_DD.go 中：

```
const(
 defaultInterval = 300 * time.Second

 // 钉钉 Webhook（如果为空，则不启用通知）
 dingTalkWebhook = "https://oapi.dingtalk.com/robot/send?access_token=xxxxx"

 // 钉钉密钥（为空则不启用加签）
 dingTalkSecret = "SECxxxxx"

) 
```


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

## 0x02 声明

本工具仅用于个人安全研究学习。由于传播、利用本工具而造成的任何直接或者间接的后果及损失，均由使用者本人负责，工具作者不为此承担任何责任。

转载请注明来源！！！！

