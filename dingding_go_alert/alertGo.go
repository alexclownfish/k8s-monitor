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
