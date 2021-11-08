# 钉钉报警插件已打包在镜像，不想麻烦的可以直接pull
docker pull alexcld/alertgo:v5
## 其他两个平台实时同步
* 个人Blog：https://blog.alexcld.com https://vue.alexcld.com
* CSDN：https://blog.csdn.net/weixin_45509582

加密token,token变量指的是顶顶机器人webhook所携带的token
```
echo -n 'token' | base64
```
alertGo-deployment.yaml
```
apiVersion: v1
kind: Secret
metadata:
  name: dd-token
  namespace: ops
type: Opaque
data:
  token: '加密后的token'
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertgo
  namespace: ops
spec:
  selector:
    matchLabels:
      app: alertgo
  replicas: 1
  template:
    metadata:
      labels:
        app: alertgo
    spec:
      containers:
        - name: alertgo
          image: alexcld/alertgo:v5
          env:
          - name: token
            valueFrom:
              secretKeyRef:
                name: dd-token
                key: token
          ports:
            - containerPort: 8088
          livenessProbe:
            httpGet:
              path: /
              port: 8088
            initialDelaySeconds: 30
            periodSeconds: 10
            successThreshold: 1
            failureThreshold: 3
            timeoutSeconds: 1
          readinessProbe:
            httpGet:
              path: /
              port: 8088
            initialDelaySeconds: 30
            periodSeconds: 10
            successThreshold: 1
            failureThreshold: 3
            timeoutSeconds: 1
          lifecycle:
            preStop:
              exec:
                command: ["/bin/bash","-c","sleep 20"]
---
apiVersion: v1
kind: Service
metadata:
  name: alertgo
  namespace: ops
spec:
  selector:
    app: alertgo
  ports:
    - port: 80
      targetPort: 8088

```
kubectl apply -f alertGo-deployment.yaml
## 修改alertmanager-configmap.yaml
![在这里插入图片描述](https://img-blog.csdnimg.cn/a516c430f9564610aae3e311fb999fd0.png?x-oss-process=image/watermark,type_ZmFuZ3poZW5naGVpdGk,shadow_10,text_aHR0cHM6Ly9ibG9nLmNzZG4ubmV0L3dlaXhpbl80NTUwOTU4Mg==,size_16,color_FFFFFF,t_70)

```
webhook_configs:
      - url: 'http://clusterIP/Alert'
        send_resolved: true
```
至此完成

# go环境部署及打包钉钉报警插件
## 其他两个平台实时同步
* 个人Blog：https://alexcld.com
* CSDN：https://blog.csdn.net/weixin_45509582
## linux安装go1.13.10

下载解压
```
cd /opt && wget https://golang.org/dl/go1.13.10.linux-amd64.tar.gz
#解压至/usr/local
tar -zxvf go1.13.10.linux-amd64.tar.gz
```
创建/opt/gocode/{src,bin,pkg}，用于设置GOPATH为/opt/gocode
```
mkdir -p /opt/gocode/{src,bin,pkg}

/opt/gocode/
├── bin
├── pkg
└── src
```
修改/etc/profile系统环境变量文件，写入GOPATH信息以及go sdk路径
```
export GOROOT=/opt/go           #Golang源代码目录，安装目录
export GOPATH=/opt/gocode        #Golang项目代码目录
export PATH=$GOROOT/bin:$PATH    #Linux环境变量
export GOBIN=$GOPATH/bin        #go install后生成的可执行命令存放路径
# 保存退出后source一下
source /etc/profile
```
执行go version
```
[root@localhost gocode]# go version
go version go1.13.10 linux/amd64
```
## 运行程序并打包
### 安装gin web框架
爬过梯子的可以直接安装，不再赘述如何爬梯子，如果没有爬梯子的话 需要设置下GOPROXY
```
golang 1.13 可以直接执行：

七牛云
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

阿里云
go env -w GO111MODULE=on
go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
```
安装gin依赖
```
go get -u github.com/gin-gonic/gin
```
创建go文件
```
mkdir -p $GOPATH/alertgo && cd $GOPATH/alertgo
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

![2fadc571a9f677760116ecb8ab051fd](https://user-images.githubusercontent.com/63449830/129005591-f95c6575-5df7-4ec4-8372-afca711f8b4b.png)


在此感谢[大佬](https://blog.51cto.com/luoguoling)的代码
