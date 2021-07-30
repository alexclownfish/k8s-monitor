# go环境部署及打包钉钉报警插件
## 其他两个平台实时同步
* 个人Blog：https://alexcld.com
* CSDN：https://blog.csdn.net/weixin_45509582
## linux安装go1.11.5

下载解压
```
mkdir ~/go && cd ~/go
wget https://dl.google.com/go/go1.11.5.linux-amd64.tar.gz
#解压至/usr/local
tar -C /usr/local -zxvf  go1.11.5.linux-amd64.tar.gz
```
添加/usr/loacl/go/bin目录到PATH变量中。添加到/etc/profile 或$HOME/.profile都可以
```
# 习惯用vim，没有的话可以用命令`sudo apt-get install vim`安装一个
vim /etc/profile
# 在最后一行添加
export GOROOT=/usr/local/go
export PATH=$PATH:$GOROOT/bin
# 保存退出后source一下
source /etc/profile
```
执行go version
![在这里插入图片描述](https://img-blog.csdnimg.cn/0ee550e1c0b14ea99b4f4433a0347855.png)
## 运行程序并打包
创建工作空间
```
mkdir $HOME/go
```
将工作空间路径声明到变量
```
# 编辑 ~/.bash_profile 文件
vim ~/.bash_profile
# 在最后一行添加下面这句。$HOME/go 为你工作空间的路径，你也可以换成你喜欢的路径
export GOPATH=$HOME/go
# 保存退出后source一下
source ~/.bash_profile
```
我这里已经爬过梯子，如果没有爬梯子的话 需要设置下GOPROXY
```
###go版本不一样,可能命令也有所不一样
export GOPROXY=https://goproxy.cn,direct
或者
go env -w GOPROXY=https://goproxy.cn,direct
```
安装gin依赖
```
go get github.com/gin-gonic/gin
```
创建go文件
```
mkdir -p $GOPATH/src/alertgo && cd $GOPATH/src/alertgo
touch alertGo.go
```
```
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//const (
//	webHook_Alert = "https://oapi.dingtalk.com/robot/send?access_token=724402cd85e7e80aa5bbbb7d7a89c74db6a3a8bd8fac4c91923ed3f906664ba4"
//)
type Message struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content               string `json:"content"`
		Mentioned_list        string `json:"mentioned_list"`
		Mentioned_mobile_list string `json:"mentioned_mobile_list"`
	} `json:"text"`
}
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:annotations`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

//通知消息结构体
type Notification struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:receiver`
	GroupLabels       map[string]string `json:groupLabels`
	CommonLabels      map[string]string `json:commonLabels`
	CommonAnnotations map[string]string `json:commonAnnotations`
	ExternalURL       string            `json:externalURL`
	Alerts            []Alert           `json:alerts`
}

//获取报警信息
func getAlertInfo(notification Notification) string {
	var m Message
	m.MsgType = "text"
	//告警消息
	//重新定义报警消息
	var newbuffer bytes.Buffer
	//定义恢复消息
	var recoverbuffer bytes.Buffer
	fmt.Println(notification)
	fmt.Println(notification.Status)
	if notification.Status == "resolved" {
		for _, alert := range notification.Alerts {
			recoverbuffer.WriteString(fmt.Sprintf("状态已经恢复!!!!\n"))
			recoverbuffer.WriteString(fmt.Sprintf(" **项目: 恢复时间:**%s\n\n", alert.StartsAt.Add(8*time.Hour).Format("2006-01-02 15:04:05")))
			recoverbuffer.WriteString(fmt.Sprintf("项目: 告警主题: %s \n", alert.Annotations["summary"]))

		}

	} else {
		for _, alert := range notification.Alerts {
			newbuffer.WriteString(fmt.Sprintf("==============Start============ \n"))
			newbuffer.WriteString(fmt.Sprintf("项目: 告警程序：prometheus_alert_email \n"))
			newbuffer.WriteString(fmt.Sprintf("项目: 告警级别: %s \n", alert.Labels["severity"]))
			newbuffer.WriteString(fmt.Sprintf("项目: 告警类型: %s \n", alert.Labels["alertname"]))
			newbuffer.WriteString(fmt.Sprintf("项目: 故障主机: %s \n", alert.Labels["instance"]))
			newbuffer.WriteString(fmt.Sprintf("项目: 告警主题: %s \n", alert.Annotations["summary"]))
			newbuffer.WriteString(fmt.Sprintf("项目: 告警详情: %s \n", alert.Annotations["description"]))
			newbuffer.WriteString(fmt.Sprintf(" **项目: 开始时间:**%s\n\n", alert.StartsAt.Add(8*time.Hour).Format("2006-01-02 15:04:05")))
			newbuffer.WriteString(fmt.Sprintf("==============End============ \n"))
		}
	}

	if notification.Status == "resolved" {
		m.Text.Content = recoverbuffer.String()
	} else {
		m.Text.Content = newbuffer.String()
	}
	jsons, err := json.Marshal(m)
	if err != nil {
		fmt.Println("解析发送钉钉的信息有问题!!!!")
	}
	resp := string(jsons)
	return resp
}

//钉钉报警
func SendAlertDingMsg(msg string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("err")
		}
	}()
	token := os.Getenv("token")
	webHook_Alert := "https://oapi.dingtalk.com/robot/send?access_token=" + token
	fmt.Println("开始发送报警消息!!!")
	fmt.Println(webHook_Alert)
	//content := `{"msgtype": "text",
	//	"text": {"content": "` + msg + `"}
	//}`

	//创建一个请求
	req, err := http.NewRequest("POST", webHook_Alert, strings.NewReader(msg))
	if err != nil {
		fmt.Println(err)
		fmt.Println("钉钉报警请求异常")
	}
	client := &http.Client{}
	//设置请求头
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	//发送请求
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		fmt.Println(err)
		fmt.Println("顶顶报发送异常!!!")
	}
	defer resp.Body.Close()
}
func AlertInfo(c *gin.Context) {
	var notification Notification
	fmt.Println("接收到的信息是....")
	fmt.Println(notification)
	err := c.BindJSON(&notification)
	fmt.Printf("%#v", notification)
	if err != nil {
		fmt.Println("绑定信息错误!!!")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	} else {
		fmt.Println("绑定信息成功")
	}
	fmt.Println("绑定信息成功!!!")
	msg := getAlertInfo(notification)
	fmt.Println("打印的信息是.....")
	fmt.Println(msg)
	SendAlertDingMsg(msg)

}
func main() {
	t := gin.Default()
	t.POST("/Alert", AlertInfo)
	t.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, `<h1>关于alertmanager实现钉钉报警的方法V6！！！</h1>`+"\n新增了报警恢复机制！！！")
	})
	t.Run(":8088")
}
```
最后将go文件打包为linux二进制可运行程序
```
go build alertGo.go
```
运行alertGo程序
```
###赋权
chmod 775 alertGo
###后台运行
nohup /root/prometheus/alertGo/alertGo > alertGo.log 2>&1 &
###查看端口进程
lsof -i:8088
###查看日志
tail -f alertGo.log 
```

## 修改alertmanager-configmap.yaml
![在这里插入图片描述](https://img-blog.csdnimg.cn/a516c430f9564610aae3e311fb999fd0.png?x-oss-process=image/watermark,type_ZmFuZ3poZW5naGVpdGk,shadow_10,text_aHR0cHM6Ly9ibG9nLmNzZG4ubmV0L3dlaXhpbl80NTUwOTU4Mg==,size_16,color_FFFFFF,t_70)

```
webhook_configs:
      - url: 'http://xxxxx:8088/Alert'
        send_resolved: true
```
热更新prometheus
```
kubectl apply -f alertmanager-configmap.yaml
###alertmanager clusterIP
curl -X POST http://10.1.229.17/-/reload  
```

然后触发报警，钉钉就可以正常收取了，
![在这里插入图片描述](https://img-blog.csdnimg.cn/dfbbf2efe4b446c1bc71a20a54855180.png?x-oss-process=image/watermark,type_ZmFuZ3poZW5naGVpdGk,shadow_10,text_aHR0cHM6Ly9ibG9nLmNzZG4ubmV0L3dlaXhpbl80NTUwOTU4Mg==,size_16,color_FFFFFF,t_70)


在此感谢[大佬](https://blog.51cto.com/luoguoling)的代码
