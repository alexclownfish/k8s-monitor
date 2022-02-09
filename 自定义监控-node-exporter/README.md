# k8s集群+传统服务器集群自定义监控node-exporter
### 由于我的项目是在两套环境上运行：k8s集群+传统服务器集群所以记录下在传统服务器集群自定义监控node-exporter
## Prometheus监控平台配置node_exporter
源码包上边有直接下载解压
```
tar -xvf node_exporter-*.linux-amd64.tar.gz -C /usr/local/

mv node_exporter-0.18.1.linux-amd64/ node_exporter
```
可以修改默认端口
```
vim node_exporter    #查找9100，然后重启node_exporter
```
![image](https://user-images.githubusercontent.com/63449830/131477090-1af86328-42db-4fe1-8046-19b88fea680f.png)

将node_exporter设置为系统服务开机自启
```
cat > /etc/systemd/system/node_exporter.service << "EOF"
[Unit]
Description=node_export
Documentation=https://github.com/prometheus/node_exporter

[Service]
ExecStart=/usr/local/node_exporter/node_exporter
ExecStart=                                                                                                   #新加参数的前一行需要添加占位
ExecStart=/usr/local/node_exporter/node_exporter --collector.textfile.directory=/usr/local/node_exporter/key #如果不做自定义监控不是node_exporter添加系统服务可以不加此行
Restart=on-failure
[Install]
WantedBy=multi-user.target
EOF
```
```
systemctl daemon-reload

systemctl enable node_exporter

systemctl start node_exporter

systemctl status node_exporter

[root@pro-zab-test3 key]# lsof -i:9100
COMMAND     PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
node_expo 11798 root    3u  IPv6  87556      0t0  TCP *:jetdirect (LISTEN)
```
http://ip:9100/metrics访问
![image](https://user-images.githubusercontent.com/63449830/131452612-7e2c0f9f-12f8-40b3-bb4b-0a025b4bf912.png)

### prometheus配置
传统方式 安装的prometheus打开prometheus.yml
如果是按照我之前的在k8s中部署的prometheus 打开prometheus-configmap.yaml

进行添加 | 修改

也可使用consul，这里不再赘述
```
    - job_name: linux-node
      static_configs:
      - targets:
        - 172.22.254.87:9100   #node_exporter主机
        - 172.22.254.86:9100   #node_exporter主机
```
![image](https://user-images.githubusercontent.com/63449830/131453250-1c2f92f8-676b-4b9c-8517-cbe16d078f63.png)

传统方式直接重启prometheus即可
k8s方式 kubectl apply -f prometheus-configmap.yaml 更新configmap配置文件到prometheus ，再热更新使配置文件生效 curl -X POST http://10.1.230.219:9090/-/reload 

## 在传统服务器上自定义node_exporter监控
首先创建目录key目录
```
cd /usr/local/node_exporter/ && mkdir key
```
创建监控程序或服务脚本key.sh，我这里做了案例，其他程序或者服务思路一致
```
#!/bin/bash
#node_exporter_status_scripts
status=`systemctl status node_exporter | grep "Active" | awk '{print $2}'`

if [ $status=="active" ]
then
  echo "node_exporter_status 0"
else
  echo "node_exporter_status 1"
fi
#alertgo_status_scripts

alertgostatus=`lsof -i:8088`

if [ "$?" = 0 ]
then
  echo "alertgo_status 0"
else
  echo "alertgo_status 1"
fi
```
```
chmod +x key.sh
```
配置计划任务
```
vim /etc/crontab

* * * * * root bash /usr/local/node_exporter/key/key.sh > /usr/local/node_exporter/key/key.prom
```
由于新加了自定义监控配置项，所以需要把自定义配置项的保存目录告诉node_exporter，我们的node_exporter使用以系统服务来启动的，所以需要在node_exporter中加入以下内容,在部署上边node_exporter中有提到
```
ExecStart=
ExecStart=/usr/local/node_exporter/node_exporter --collector.textfile.directory=/usr/local/node_exporter/key
```
到此就结束了，如果配置正确，重启一下node_exporter再次刷新页面可以看到
根据服务的启停可以看到
```
[root@pro-zab-test3 key]# cat key.prom 
node_exporter_status 0
alertgo_status 0
```

![image](https://user-images.githubusercontent.com/63449830/131454926-7867fc50-39dc-4400-8ff2-8ba50f53676c.png)
![image](https://user-images.githubusercontent.com/63449830/131454973-58e207b7-ed3e-4656-b0b4-10a8461fdec9.png)

在prometheus 中也可以用promSql进行查询制表

![image](https://user-images.githubusercontent.com/63449830/131455274-c1b0f06e-7e08-44d5-a704-d6ea5e1e63d0.png)
![image](https://user-images.githubusercontent.com/63449830/131455393-02746554-1623-4859-9431-1ec41eec78f1.png)

## 模拟故障
在prometheus-rules.yaml中添加rules规则，传统部署正常添加即可，我这里用k8s方式示例
```
linux-node.rules: |
    groups:
    - name: linux-node.rules
      rules:
      - alert: alertgoDone
        expr: |
           alertgo_status==1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "{{ $labels.instance }}: alertgo is lost\n  VALUE = {{ $value }}\n  LABELS = {{ $labels }}"
```
```
 kubectl apply -f prometheus-rules.yaml 更新configmap配置文件到prometheus ，再热更新使配置文件生效 curl -X POST http://10.1.230.219:9090/-/reload 
```
我这里alertgo是go开发的二进制，我直接杀掉进程即可模拟
查询进程号
```
[root@pro-zab-test3 key]# lsof -i:8088
COMMAND     PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
alertGoV6 11984 root    3u  IPv6  89542      0t0  TCP *:radan-http (LISTEN)

[root@pro-zab-test3 key]# cat key.prom 
node_exporter_status 0
alertgo_status 0

```
杀死进程
```
kill 11984
```
再次查看key.prom，发现value为1
```
[root@pro-zab-test3 key]# cat key.prom 
node_exporter_status 0
alertgo_status 1
```
查看prometheus alerts
![1630393080(1)](https://user-images.githubusercontent.com/63449830/131457282-95826395-4ba1-4099-af1f-1ab3b950765d.jpg)
钉钉已报警

![1630393114(1)](https://user-images.githubusercontent.com/63449830/131457342-d5bedbf5-f806-4c37-a142-976e5548ab30.jpg)

# grafana部分后续更新 
![1630466573](https://user-images.githubusercontent.com/63449830/131606618-3ce8813b-b25c-43f2-8cff-f5b13cfda4ce.jpg)

![1630466575(1)](https://user-images.githubusercontent.com/63449830/131606624-71149d88-455d-466d-9f66-174461770359.jpg)

