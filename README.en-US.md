

# Prometheus + Grafana + Alertmanager Kubernetes Monitoring (Trouble-Free Version)
## Real-time Sync to Other Platforms
* Personal Blog: https://blog.alexcld.com https://vue.alexcld.com
* CSDN: https://blog.csdn.net/weixin_45509582
## Overview
A dynamic storage (PersistentVolume) is required on top of a properly set up and functioning Kubernetes cluster.
My Environment:
K8s Version    | Kubeadm Deployment v1.18.0
-------- | -----
k8s-master  | 172.22.254.57
k8s-node1  | 172.22.254.62
k8s-node2  | 172.22.254.63 (NFS Server)
StorageClass | nfs-storage

The k8s-master node has a taint. If you need to monitor the master, simply remove the taint (optional).
```
kubectl taint nodes node1 key1=value1:NoSchedule-
```

The rule fields in `prometheus-rules` may change with version updates. If you notice any changes, please notify me so I can update the documentation in real-time. The fields in the current rules have already been updated for this version. Feel free to use them.

One minor detail: The ConfigMaps for Prometheus and Alertmanager support hot reloading. You can use the following command to hot-reload. The refresh process might take a moment, so just wait a little.
```
curl -X POST http://ClusterIP:PORT/-/reload
```
Resource Download: [https://github.com/alexclownfish/k8s-monitor](https://github.com/alexclownfish/k8s-monitor)

The trigger time received in the alert email is in UTC. You can modify the trigger event in the custom template `alertmanager-template.yaml`.
```
Solution:
To change it to Beijing Time (UTC+8), you can modify it as follows:
The original trigger time in the alert template looks like this:
Trigger Time: {{ .StartsAt.Format "2020-01-02 15:04:05" }}
We can change it to:
{{ (.StartsAt.Add 28800e9).Format "2020-01-02 15:04:05" }}
Here, `Add 28800e9` means adding 8 hours.
```

I have updated the Grafana templates (Ops Resource Overview, Node Resource Overview). They can still be loaded from the original directory. If you encounter any issues, feel free to contact me.


## Deployment Guide
### Create the `ops` Namespace
```
kubectl create ns ops
```
### Prometheus YAML Files
#### Prometheus Configuration File: `prometheus-configmap.yaml`
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: ops 
data:
  prometheus.yml: |
    rule_files:
    - /etc/config/rules/*.rules

    scrape_configs:
    - job_name: prometheus
      static_configs:
      - targets:
        - localhost:9090

    - job_name: kubernetes-apiservers
      kubernetes_sd_configs:
      - role: endpoints
      relabel_configs:
      - action: keep
        regex: default;kubernetes;https
        source_labels:
        - __meta_kubernetes_namespace
        - __meta_kubernetes_service_name
        - __meta_kubernetes_endpoint_port_name
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
 
    - job_name: kubernetes-nodes-kubelet
      kubernetes_sd_configs:
      - role: node  # Discover nodes in the cluster
      relabel_configs:
      # Use labels (.*) as new label names, keeping original values
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

    - job_name: kubernetes-nodes-cadvisor
      kubernetes_sd_configs:
      - role: node
      relabel_configs:
      # Use labels (.*) as new label names, keeping original values
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      # Actually accesses the metrics endpoint https://NodeIP:10250/metrics/cadvisor, replacing the default metrics URL path here
      - target_label: __metrics_path__
        replacement: /metrics/cadvisor
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

    - job_name: kubernetes-service-endpoints
      kubernetes_sd_configs:
      - role: endpoints  # Discover Pods as targets from Endpoints in the Service list
      relabel_configs:
      # Do not scrape Services without the prometheus.io/scrape annotation
      - action: keep
        regex: true
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_scrape
      # Rename scrape target scheme
      - action: replace
        regex: (https?)
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_scheme
        target_label: __scheme__
      # Rename scrape target metrics URL path
      - action: replace
        regex: (.+)
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_path
        target_label: __metrics_path__
      # Rename scrape target address
      - action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        source_labels:
        - __address__
        - __meta_kubernetes_service_annotation_prometheus_io_port
        target_label: __address__
      # Use K8s labels (.*) as new label names, keeping original values
      - action: labelmap
        regex: __meta_kubernetes_service_label_(.+)
      # Generate namespace label
      - action: replace
        source_labels:
        - __meta_kubernetes_namespace
        target_label: kubernetes_namespace
      # Generate Service name label
      - action: replace
        source_labels:
        - __meta_kubernetes_service_name
        target_label: kubernetes_name

    - job_name: kubernetes-pods
      kubernetes_sd_configs:
      - role: pod   # Discover all Pods as targets
      # Rename scrape target scheme
      relabel_configs:
      - action: keep
        regex: true
        source_labels:
        - __meta_kubernetes_pod_annotation_prometheus_io_scrape
      # Rename scrape target metrics URL path
      - action: replace
        regex: (.+)
        source_labels:
        - __meta_kubernetes_pod_annotation_prometheus_io_path
        target_label: __metrics_path__
      # Rename scrape target address
      - action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        source_labels:
        - __address__
        - __meta_kubernetes_pod_annotation_prometheus_io_port
        target_label: __address__
      # Use K8s labels (.*) as new label names, keeping original values
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      # Generate namespace label
      - action: replace
        source_labels:
        - __meta_kubernetes_namespace
        target_label: kubernetes_namespace
      # Generate Pod name label
      - action: replace
        source_labels:
        - __meta_kubernetes_pod_name
        target_label: kubernetes_pod_name

    alerting:
      alertmanagers:
      - static_configs:
          - targets: ["alertmanager:80"]
```
#### kube-state-metrics collects status information of various resource objects in K8s: `kube-state-metrics.yaml`
```
apiVersion: apps/v1 
kind: Deployment
metadata:
  name: kube-state-metrics
  namespace: ops
  labels:
    k8s-app: kube-state-metrics
spec:
  selector:
    matchLabels:
      k8s-app: kube-state-metrics
      version: v1.3.0
  replicas: 1
  template:
    metadata:
      labels:
        k8s-app: kube-state-metrics
        version: v1.3.0
    spec:
      serviceAccountName: kube-state-metrics
      containers:
      - name: kube-state-metrics
        image: lizhenliang/kube-state-metrics:v1.8.0 
        ports:
        - name: http-metrics
          containerPort: 8080
        - name: telemetry
          containerPort: 8081
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          timeoutSeconds: 5
      - name: addon-resizer
        image: lizhenliang/addon-resizer:1.8.6
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 30Mi
        env:
          - name: MY_POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: MY_POD_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
        volumeMounts:
          - name: config-volume
            mountPath: /etc/config
        command:
          - /pod_nanny
          - --config-dir=/etc/config
          - --container=kube-state-metrics
          - --cpu=100m
          - --extra-cpu=1m
          - --memory=100Mi
          - --extra-memory=2Mi
          - --threshold=5
          - --deployment=kube-state-metrics
      volumes:
        - name: config-volume
          configMap:
            name: kube-state-metrics-config
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-state-metrics-config
  namespace: ops
data:
  NannyConfiguration: |-
    apiVersion: nannyconfig/v1alpha1
    kind: NannyConfiguration
---
apiVersion: v1
kind: Service
metadata:
  name: kube-state-metrics
  namespace: ops
  annotations:
    prometheus.io/scrape: 'true'
spec:
  ports:
  - name: http-metrics
    port: 8080
    targetPort: http-metrics
    protocol: TCP
  - name: telemetry
    port: 8081
    targetPort: telemetry
    protocol: TCP
  selector:
    k8s-app: kube-state-metrics
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-state-metrics
  namespace: ops
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kube-state-metrics
rules:
- apiGroups: [""]
  resources:
  - configmaps
  - secrets
  - nodes
  - pods
  - services
  - resourcequotas
  - replicationcontrollers
  - limitranges
  - persistentvolumeclaims
  - persistentvolumes
  - namespaces
  - endpoints
  verbs: ["list", "watch"]
- apiGroups: ["apps"]
  resources:
  - statefulsets
  - daemonsets
  - deployments
  - replicasets
  verbs: ["list", "watch"]
- apiGroups: ["batch"]
  resources:
  - cronjobs
  - jobs
  verbs: ["list", "watch"]
- apiGroups: ["autoscaling"]
  resources:
  - horizontalpodautoscalers
  verbs: ["list", "watch"]
- apiGroups: ["networking.k8s.io", "extensions"]
  resources:
  - ingresses 
  verbs: ["list", "watch"]
- apiGroups: ["storage.k8s.io"]
  resources:
  - storageclasses 
  verbs: ["list", "watch"]
- apiGroups: ["certificates.k8s.io"]
  resources:
  - certificatesigningrequests
  verbs: ["list", "watch"]
- apiGroups: ["policy"]
  resources:
  - poddisruptionbudgets 
  verbs: ["list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kube-state-metrics-resizer
  namespace: ops
rules:
- apiGroups: [""]
  resources:
  - pods
  verbs: ["get"]
- apiGroups: ["extensions","apps"]
  resources:
  - deployments
  resourceNames: ["kube-state-metrics"]
  verbs: ["get", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1 
kind: ClusterRoleBinding
metadata:
  name: kube-state-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kube-state-metrics
subjects:
- kind: ServiceAccount
  name: kube-state-metrics
  namespace: ops
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kube-state-metrics
  namespace: ops
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kube-state-metrics-resizer
subjects:
- kind: ServiceAccount
  name: kube-state-metrics
  namespace: ops
```
#### Prometheus Deployment File: `prometheus-deploy.yaml` (Note: Version must be 2.20)
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus 
  namespace: ops
  labels:
    k8s-app: prometheus
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: prometheus
  template:
    metadata:
      labels:
        k8s-app: prometheus
    spec:
      serviceAccountName: prometheus
      initContainers:
      - name: "init-chown-data"
        image: "busybox:latest"
        imagePullPolicy: "IfNotPresent"
        command: ["chown", "-R", "65534:65534", "/data"]
        volumeMounts:
        - name: prometheus-data
          mountPath: /data
          subPath: ""
      containers:
        - name: prometheus-server-configmap-reload
          image: "jimmidyson/configmap-reload:v0.1"
          imagePullPolicy: "IfNotPresent"
          args:
            - --volume-dir=/etc/config
            - --webhook-url=http://localhost:9090/-/reload
          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
              readOnly: true
            - mountPath: /etc/localtime
              name: timezone
          resources:
            limits:
              cpu: 10m
              memory: 100Mi
            requests:
              cpu: 10m
              memory: 100Mi

        - name: prometheus-server
          image: "prom/prometheus:v2.20.0"
          imagePullPolicy: "IfNotPresent"
          args:
            - --config.file=/etc/config/prometheus.yml
            - --storage.tsdb.path=/data
            - --web.console.libraries=/etc/prometheus/console_libraries
            - --web.console.templates=/etc/prometheus/consoles
            - --web.enable-lifecycle
          ports:
            - containerPort: 9090
          readinessProbe:
            httpGet:
              path: /-/ready
              port: 9090
            initialDelaySeconds: 30
            timeoutSeconds: 30
          livenessProbe:
            httpGet:
              path: /-/healthy
              port: 9090
            initialDelaySeconds: 30
            timeoutSeconds: 30
          resources:
            limits:
              cpu: 500m
              memory: 800Mi
            requests:
              cpu: 200m
              memory: 400Mi
            
          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
            - name: prometheus-data
              mountPath: /data
              subPath: ""
            - name: prometheus-rules
              mountPath: /etc/config/rules
            - mountPath: /etc/localtime
              name: timezone  
      volumes:
        - name: config-volume
          configMap:
            name: prometheus-config
        - name: prometheus-rules
          configMap:
            name: prometheus-rules
        - name: prometheus-data
          persistentVolumeClaim:
            claimName: prometheus
        - name: timezone
          hostPath:
            path: /usr/share/zoneinfo/Asia/Shanghai
                                                  

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: prometheus
  namespace: ops
spec:
  storageClassName: "nfs-storage"
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Gi
---
apiVersion: v1
kind: Service
metadata: 
  name: prometheus
  namespace: ops
spec: 
  type: NodePort
  ports: 
    - name: http 
      port: 9090
      protocol: TCP
      targetPort: 9090
      nodePort: 30089
  selector: 
    k8s-app: prometheus
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: ops
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus
rules:
  - apiGroups:
      - ""
    resources:
      - nodes
      - nodes/metrics
      - services
      - endpoints
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs:
      - get
  - nonResourceURLs:
      - "/metrics"
    verbs:
      - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
- kind: ServiceAccount
  name: prometheus
  namespace: ops  
```
#### Prometheus Alert Rules Configuration: `prometheus-rules.yaml`
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-rules
  namespace: ops
data:
  general.rules: |
    groups:
    - name: general.rules
      rules:
      - alert: InstanceDown
        expr: up == 0
        for: 1m
        labels:
          severity: error 
        annotations:
          summary: "Instance {{ $labels.instance }} is down"
          description: "{{ $labels.instance }} job {{ $labels.job }} has been down for over 5 minutes."
               
  node.rules: |
    groups:
    - name: node.rules
      rules:
      - alert: NodeFilesystemUsage
        expr: |
          100 - (node_filesystem_free_bytes / node_filesystem_size_bytes) * 100 > 60
        for: 1m
        labels:
          severity: warning 
        annotations:
          summary: "Instance {{ $labels.instance }}: {{ $labels.mountpoint }} partition usage is too high"
          description: "{{ $labels.instance }}: {{ $labels.mountpoint }} partition usage exceeds 60% (current value: {{ $value }})"

      - alert: NodeMemoryUsage
        expr: |
          100 - (node_memory_MemFree_bytes+node_memory_Cached_bytes+node_memory_Buffers_bytes) / node_memory_MemTotal_bytes * 100 > 60
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Instance {{ $labels.instance }} memory usage is too high"
          description: "{{ $labels.instance }} memory usage exceeds 60% (current value: {{ $value }})"

      - alert: NodeCPUUsage    
        expr: |
          100 - (avg(irate(node_cpu_seconds_total{mode="idle"}[5m])) by (instance) * 100) > 60 
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Instance {{ $labels.instance }} CPU usage is too high"       
          description: "{{ $labels.instance }} CPU usage exceeds 60% (current value: {{ $value }})"

      - alert: KubeNodeNotReady
        expr: |
          kube_node_status_condition{condition="Ready",status="true"} == 0
        for: 1m
        labels:
          severity: error
        annotations:
          message: '{{ $labels.node }} has not been ready for over 10 minutes.'

  pod.rules: |
    groups:
    - name: pod.rules
      rules:
      - alert: PodCPUUsage
        expr: |
           sum by(pod, namespace) (rate(container_cpu_usage_seconds_total{image!=""}[5m]) * 100) > 5
        for: 5m
        labels:
          severity: warning 
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} CPU usage exceeds 80% (current value: {{ $value }})"

      - alert: PodMemoryUsage
        expr: |
           sum(container_memory_rss{image!=""}) by(pod, namespace) / sum(container_spec_memory_limit_bytes{image!=""}) by(pod, namespace) * 100 != +inf > 80
        for: 5m
        labels:
          severity: error 
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} memory usage exceeds 80% (current value: {{ $value }})"

      - alert: PodNetworkReceive
        expr: |
           sum(rate(container_network_receive_bytes_total{image!="",name=~"^k8s_.*"}[5m]) /1000) by (pod,namespace) > 30000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} inbound traffic exceeds 30MB/s (current value: {{ $value }}K/s)"           

      - alert: PodNetworkTransmit
        expr: | 
           sum(rate(container_network_transmit_bytes_total{image!="",name=~"^k8s_.*"}[5m]) /1000) by (pod,namespace) > 30000
        for: 5m
        labels:
          severity: warning 
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} outbound traffic exceeds 30MB/s (current value: {{ $value }}K/s)"

      - alert: PodRestart
        expr: |
           sum(changes(kube_pod_container_status_restarts_total[1m])) by (pod,namespace) > 0
        for: 1m
        labels:
          severity: warning 
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod restart (current value: {{ $value }})"

      - alert: PodFailed
        expr: |
           sum(kube_pod_status_phase{phase="Failed"}) by (pod,namespace) > 0
        for: 5s
        labels:
          severity: error 
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod status is Failed (current value: {{ $value }})"

      - alert: PodPending
        expr: | 
           sum(kube_pod_status_phase{phase="Pending"}) by (pod,namespace) > 0
        for: 1m
        labels:
          severity: error
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod status is Pending (current value: {{ $value }})"


      - alert: PodErrImagePull
        expr: |
           sum by(namespace,pod) (kube_pod_container_status_waiting_reason{reason="ErrImagePull"}) == 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod status is ErrImagePull (current value: {{ $value }})"
      - alert: PodImagePullBackOff
        expr: |
           sum by(namespace,pod) (kube_pod_container_status_waiting_reason{reason="ImagePullBackOff"}) == 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod status is ImagePullBackOff (current value: {{ $value }})"
      - alert: PodCrashLoopBackOff
        expr: |
           sum by(namespace,pod) (kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"}) == 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod status is CrashLoopBackOff (current value: {{ $value }})"
      - alert: PodInvalidImageName
        expr: |
           sum by(namespace,pod) (kube_pod_container_status_waiting_reason{reason="InvalidImageName"}) == 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod status is InvalidImageName (current value: {{ $value }})"
      - alert: PodCreateContainerConfigError
        expr: |
           sum by(namespace,pod) (kube_pod_container_status_waiting_reason{reason="CreateContainerConfigError"}) == 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Namespace: {{ $labels.namespace }} | Pod: {{ $labels.pod }} Pod status is CreateContainerConfigError (current value: {{ $value }})"

  volume.rules: |
    groups:
    - name: volume.rules
      rules:
      - alert: PersistentVolumeClaimLost
        expr: |
           sum by(namespace, persistentvolumeclaim) (kube_persistentvolumeclaim_status_phase{phase="Lost"}) == 1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "PersistentVolumeClaim {{ $labels.namespace }}/{{ $labels.persistentvolumeclaim }} is lost\n  VALUE = {{ $value }}\n  LABELS = {{ $labels }}"
      - alert: PersistentVolumeClaimPendig
        expr: |
           sum by(namespace, persistentvolumeclaim) (kube_persistentvolumeclaim_status_phase{phase="Pendig"}) == 1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "PersistentVolumeClaim {{ $labels.namespace }}/{{ $labels.persistentvolumeclaim }} is pending\n  VALUE = {{ $value }}\n  LABELS = {{ $labels }}"
      - alert: PersistentVolume Failed
        expr: |
           sum(kube_persistentvolume_status_phase{phase="Failed",job="kubernetes-service-endpoints"}) by (persistentvolume) == 1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Persistent volume is in failed state\n  VALUE = {{ $value }}\n  LABELS = {{ $labels }}"
      - alert: PersistentVolume Pending
        expr: |
           sum(kube_persistentvolume_status_phase{phase="Pending",job="kubernetes-service-endpoints"}) by (persistentvolume) == 1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Persistent volume is in pending state\n  VALUE = {{ $value }}\n  LABELS = {{ $labels }}"

```
#### node-exporter Configuration: `node-exporter.yaml` (Note: Version must be 1.0.1)
```
apiVersion: apps/v1 
kind: DaemonSet
metadata:
  name: node-exporter 
  namespace: ops 
  labels:
    k8s-app: node-exporter 
spec:
  selector:
    matchLabels:
      k8s-app: node-exporter
      version: v1.0.1
  template:
    metadata:
      labels:
        k8s-app: node-exporter 
        version: v1.0.1
    spec:
      containers:
        - name: prometheus-node-exporter
          image: "prom/node-exporter:v1.0.1"
          #imagePullPolicy: "Always"
          args:
            - --path.procfs=/host/proc
            - --path.sysfs=/host/sys
          ports:
            - name: metrics
              containerPort: 9100
              hostPort: 9100
          volumeMounts:
            - name: proc
              mountPath: /host/proc
              readOnly:  true
            - name: sys
              mountPath: /host/sys
              readOnly: true
          resources:
            limits:
              cpu: 10m
              memory: 50Mi
            requests:
              cpu: 10m
              memory: 50Mi
      hostNetwork: true
      hostPID: true
      hostIPC: true
      volumes:
        - name: proc
          hostPath:
            path: /proc
        - name: sys
          hostPath:
            path: /sys
        - name: rootfs
          hostPath:
            path: /
        - name: dev
          hostPath:
            path: /dev
---
apiVersion: v1
kind: Service
metadata:
  name: node-exporter
  namespace: ops
  annotations:
    prometheus.io/scrape: "true"
spec:
  clusterIP: None
  ports:
    - name: metrics
      port: 9100
      protocol: TCP
      targetPort: 9100
  selector:
    k8s-app: node-exporter

```
### Alertmanager YAML Files
#### Alertmanager Configuration File: `alertmanger-configmap.yaml`
Note: You need to apply for an email account on NetEase (163.com) yourself and obtain the authorization management password.
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: alertmanager-config
  namespace: ops
data:
  alertmanager.yml: |-
    global:
      # Time to declare resolved if no alerts
      resolve_timeout: 5m
      # Configure email sending information
      smtp_smarthost: 'smtp.163.com:465'
      smtp_from: 'xxx@163.com'
      smtp_auth_username: 'xxx@163.com'
      smtp_auth_password: 'xxxxxx'
      smtp_hello: '163.com'
      smtp_require_tls: false
    # Root route for all incoming alerts, used to set the dispatching strategy
    route:
      # The label list here is for regrouping after receiving alert information. For example, alerts with cluster=A and alertname=LatncyHigh will be grouped together.
      group_by: ['alertname', 'cluster']
      # When a new alert group is created, wait at least group_wait time to initialize notifications. This ensures you have enough time to collect multiple alerts for the same group and trigger them together.
      group_wait: 30s
 
      # After the first alert is sent, wait 'group_interval' time before sending a new group of alerts.
      group_interval: 5m
 
      # If an alert has been successfully sent, wait 'repeat_interval' time before re-sending it.
      repeat_interval: 5m
 
      # Default receiver: If an alert is not matched by a route, send it to the default receiver.
      receiver: default
 
      # All properties above are inherited by all child routes and can be overridden in each child route.
      routes:
      - receiver: email
        group_wait: 10s
        match:
          team: node
    templates:
      - '/etc/config/template/email.tmpl'
    receivers:
    - name: 'default'
      email_configs:
      - to: 'xxxx@qq.com'
        html: '{{ template "email.html" . }}'
        headers: { Subject: "[WARN] Prometheus Alert Email" }
        #send_resolved: true
    - name: 'email'
      email_configs:
      - to: 'xxxx@gmail.com'
        send_resolved: true

```
#### Alertmanager Template File: `alertmanager-template.yaml`
```
# Custom alert template
apiVersion: v1
kind: ConfigMap
metadata:
  name: alertmanager-template-volume
  namespace: ops
data:
  email.tmpl: |
    {{ define "email.html" }}
        {{ range .Alerts }}
    <pre>
        ========start==========
       Alert Program: prometheus_alert_email 
       Alert Level: {{ .Labels.severity }} 
       Alert Type: {{ .Labels.alertname }} 
       Faulty Host: {{ .Labels.instance }} 
       Alert Subject: {{ .Annotations.summary }}
       Alert Details: {{ .Annotations.description }}
       Resolution Method: {{ .Annotations.console }}
       Trigger Time: {{ .StartsAt.Format "2006-01-02 15:04:05" }}
       ========end==========
    </pre>
        {{ end }}
    {{ end }}

```
#### Alertmanager Deployment File: `alertmanager-deployment.yaml`
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertmanager
  namespace: ops
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: alertmanager
      version: v0.14.0
  template:
    metadata:
      labels:
        k8s-app: alertmanager
        version: v0.14.0
    spec:
      containers:
        - name: prometheus-alertmanager
          image: "prom/alertmanager:v0.14.0"
          imagePullPolicy: "IfNotPresent"
          args:
            - --config.file=/etc/config/alertmanager.yml
            - --storage.path=/data
            - --web.external-url=/
          ports:
            - containerPort: 9093
          readinessProbe:
            httpGet:
              path: /#/status
              port: 9093
            initialDelaySeconds: 30
            timeoutSeconds: 30
          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
# Custom alert template
            - name: config-template-volume
              mountPath: /etc/config/template
            - name: storage-volume
              mountPath: "/data"
              subPath: ""
            - mountPath: /etc/localtime
              name: timezone
          resources:
            limits:
              cpu: 10m
              memory: 200Mi
            requests:
              cpu: 10m
              memory: 100Mi
        - name: prometheus-alertmanager-configmap-reload
          image: "jimmidyson/configmap-reload:v0.1"
          imagePullPolicy: "IfNotPresent"
          args:
            - --volume-dir=/etc/config
            - --webhook-url=http://localhost:9093/-/reload
          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
              readOnly: true
          resources:
            limits:
              cpu: 10m
              memory: 200Mi
            requests:
              cpu: 10m
              memory: 100Mi
      volumes:
        - name: config-volume
          configMap:
            name: alertmanager-config
        - name: config-template-volume
          configMap:
            name: alertmanager-template-volume
        - name: storage-volume
          persistentVolumeClaim:
            claimName: alertmanager
        - name: timezone
          hostPath:
            path: /usr/share/zoneinfo/Asia/Shanghai
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: alertmanager
  namespace: ops
spec:
  storageClassName: nfs-storage
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: "2Gi"
---
apiVersion: v1
kind: Service
metadata:
  name: alertmanager
  namespace: ops
  labels:
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
    kubernetes.io/name: "Alertmanager"
spec:
  type: "NodePort"
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 9093
      nodePort: 30093
  selector:
    k8s-app: alertmanager


```
### Grafana YAML File
```
apiVersion: apps/v1
kind: Deployment 
metadata:
  name: grafana
  namespace: ops
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:7.1.0
        ports:
          - containerPort: 3000
            protocol: TCP
        resources:
          limits:
            cpu: 100m            
            memory: 256Mi          
          requests:
            cpu: 100m            
            memory: 256Mi
        volumeMounts:
          - name: grafana-data
            mountPath: /var/lib/grafana
            subPath: grafana
          - mountPath: /etc/localtime
            name: timezone
      securityContext:
        fsGroup: 472
        runAsUser: 472
      volumes:
      - name: grafana-data
        persistentVolumeClaim:
          claimName: grafana
      - name: timezone
        hostPath:
          path: /usr/share/zoneinfo/Asia/Shanghai 
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: grafana 
  namespace: ops
spec:
  storageClassName: "nfs-storage"
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 5Gi
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: ops
spec:
  type: NodePort
  ports:
  - port : 80
    targetPort: 3000
    nodePort: 30030
  selector:
    app: grafana

```
### Deploy to Kubernetes
```
kubectl apply -f .
```

## Grafana Datasource and Monitoring
### Add Datasource in Grafana
![Insert image description here](https://img-blog.csdnimg.cn/img_convert/2e2b52257bf58eb3de3800b73bf1b33a.png)
Click Datasource - Add Datasource 
![Insert image description here](https://img-blog.csdnimg.cn/img_convert/f4a6d0482662d940b54f5345ff5719b6.png)
Then click Save & Test to finish adding the datasource.

### Import Templates
Template Download: [https://github.com/alexclownfish/k8s-monitor/tree/main/grafana_template](https://github.com/alexclownfish/k8s-monitor/tree/main/grafana_template)
![Insert image description here](https://img-blog.csdnimg.cn/img_convert/0da7161b13ec62e8a64bb700885473fc.png)
![Insert image description here](https://img-blog.csdnimg.cn/img_convert/6dc324132e15af124c96a6cb10632f20.png)
### Modify Prometheus Rules to Verify Monitoring Triggers Alerts and Sends Emails
Modify `prometheus-rules.yaml`
![Insert image description here](https://img-blog.csdnimg.cn/img_convert/68786c7f9abf4518c5cf910056074369.png)
```
# Hot reload ConfigMap
kubectl apply -f prometheus-rules.yaml
curl -X POST http://10.1.230.219:9090/-/reload
```
![Insert image description here](https://img-blog.csdnimg.cn/img_convert/9bed866e7556554490febdfb84bf3769.png)
![Insert image description here](https://img-blog.csdnimg.cn/img_convert/7a5776cc75bbd830ee989699467317c4.png)

![image](https://user-images.githubusercontent.com/63449830/140845934-b34278e9-3a89-4ef7-a245-bd426e9a9ea7.png)
![image](https://user-images.githubusercontent.com/63449830/140845943-5bf320bc-2727-4973-b175-481f0eea5792.png)

You can see that the alert has been triggered and the email has been sent.
This concludes the guide.

## Credits / Thanks
[https://blog.51cto.com/luoguoling](https://blog.51cto.com/luoguoling)
[https://alexcld.com](https://alexcld.com)
