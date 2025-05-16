package service

import (
	"encoding/json"
	"time"
)

// GenerateMockPodDescription 生成模拟的Pod描述响应，用于测试和文档目的
func GenerateMockPodDescription() (*PodDescribeResponse, error) {
	// 创建当前时间和过去时间，用于模拟创建时间和事件时间
	now := time.Now()
	creationTime := now.Add(-24 * time.Hour) // Pod创建于24小时前
	eventTime1 := now.Add(-24 * time.Hour)   // 首个事件发生在Pod创建时
	eventTime2 := now.Add(-12 * time.Hour)   // 第二个事件发生在12小时前
	eventTime3 := now.Add(-30 * time.Minute) // 最近的事件发生在30分钟前

	// 创建模拟的Pod描述响应
	mockResponse := &PodDescribeResponse{
		PodName:           "nginx-deployment-66b6c48dd5-abcde",
		Namespace:         "default",
		Status:            "Running",
		CreationTimestamp: creationTime.Format(time.RFC3339),
		Labels: map[string]string{
			"app":               "nginx",
			"pod-template-hash": "66b6c48dd5",
			"tier":              "frontend",
			"environment":       "production",
		},
		Annotations: map[string]string{
			"kubernetes.io/config.seen":    creationTime.Format(time.RFC3339),
			"kubernetes.io/config.source":  "api",
			"prometheus.io/scrape":         "true",
			"prometheus.io/port":           "9113",
			"sidecar.istio.io/inject":      "true",
			"kubectl.kubernetes.io/restartedAt": creationTime.Format(time.RFC3339),
		},
		NodeName: "worker-node-01",
		IP:       "10.244.2.15",
		QoS:      "Burstable",
		Containers: []ContainerDescription{
			{
				Name:         "nginx",
				Image:        "nginx:1.21",
				ImageID:      "docker-pullable://nginx@sha256:2834dc507516af02784808c5f48b7cbe38b8ed5d0f4837f16e78d00deb7e7767",
				ContainerID:  "docker://3a45c7865d85395f5e22de902526c4f565c7f096c79dfe87bbf5a195c6006427",
				State:        "Running",
				StateDetails: ContainerStateDetails{
					State:   "Running",
					Running: &RunningState{
						StartedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
					},
					LastState: &LastState{
						State: "Terminated",
						Terminated: &TerminatedState{
							Reason:   "Error",
							ExitCode: 1,
						},
					},
				},
				Ready:        true,
				RestartCount: 2,
				Command:      []string{"/docker-entrypoint.sh"},
				Args:         []string{"nginx", "-g", "daemon off;"},
				WorkingDir:   "/usr/share/nginx/html",
				Ports: []PortDescription{
					{
						Name:          "http",
						ContainerPort: 80,
						Protocol:      "TCP",
					},
					{
						Name:          "https",
						ContainerPort: 443,
						Protocol:      "TCP",
					},
				},
				Resources: ResourceDescription{
					Requests: ResourceQuantity{
						CPU:    "100m",
						Memory: "128Mi",
					},
					Limits: ResourceQuantity{
						CPU:    "200m",
						Memory: "256Mi",
					},
				},
				Mounts: []MountDescription{
					{
						Name:      "nginx-config",
						MountPath: "/etc/nginx/conf.d",
						ReadOnly:  true,
					},
					{
						Name:      "nginx-data",
						MountPath: "/var/www/html",
						ReadOnly:  false,
					},
				},
				SecurityContext: &SecurityContextDescription{
					Privileged:             false,
					RunAsUser:              1000,
					RunAsGroup:             1000,
					RunAsNonRoot:           true,
					ReadOnlyRootFilesystem: false,
					AllowPrivilegeEscalation: false,
					CapabilitiesAdd:        []string{"NET_ADMIN"},
					CapabilitiesDrop:       []string{"ALL"},
					SeccompProfile:         "RuntimeDefault",
					AppArmorProfile:        "runtime/default",
				},
				LivenessProbe: &ProbeDescription{
					Type:                "HTTP",
					Path:                "/healthz",
					Port:                80,
					Scheme:              "HTTP",
					InitialDelaySeconds: 30,
					TimeoutSeconds:      5,
					PeriodSeconds:       10,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				ReadinessProbe: &ProbeDescription{
					Type:                "HTTP",
					Path:                "/readiness",
					Port:                80,
					Scheme:              "HTTP",
					InitialDelaySeconds: 10,
					TimeoutSeconds:      5,
					PeriodSeconds:       10,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				Env: []EnvDescription{
					{
						Name:  "NGINX_HOST",
						Value: "example.com",
					},
					{
						Name:      "POD_NAMESPACE",
						ValueFrom: "Field metadata.namespace",
					},
					{
						Name:      "DATABASE_PASSWORD",
						ValueFrom: "Secret db-credentials, key password",
					},
				},
			},
			{
				Name:         "nginx-exporter",
				Image:        "nginx/nginx-prometheus-exporter:0.9.0",
				ImageID:      "docker-pullable://nginx/nginx-prometheus-exporter@sha256:7d3117a0f4c95f3b0a41d7e02c936a83e55e51a8a7985d0e3399c9b5786d6c27",
				ContainerID:  "docker://9b7f5d8b2a7c5d8e2a7f5d8b2a7c5d8e2a7f5d8b2a7c5d8e2a7f5d8b2a7c5d8e",
				State:        "Running",
				StateDetails: ContainerStateDetails{
					State:   "Running",
					Running: &RunningState{
						StartedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
					},
				},
				Ready:        true,
				RestartCount: 0,
				Command:      []string{"/bin/nginx_exporter"},
				Args:         []string{"--nginx.scrape-uri=http://localhost/nginx_status"},
				Ports: []PortDescription{
					{
						Name:          "metrics",
						ContainerPort: 9113,
						Protocol:      "TCP",
					},
				},
				Resources: ResourceDescription{
					Requests: ResourceQuantity{
						CPU:    "50m",
						Memory: "64Mi",
					},
					Limits: ResourceQuantity{
						CPU:    "100m",
						Memory: "128Mi",
					},
				},
				SecurityContext: &SecurityContextDescription{
					RunAsNonRoot:           true,
					RunAsUser:              10001,
					ReadOnlyRootFilesystem: true,
					AllowPrivilegeEscalation: false,
					CapabilitiesDrop:       []string{"ALL"},
				},
				Env: []EnvDescription{
					{
						Name:  "SCRAPE_URI",
						Value: "http://localhost/nginx_status",
					},
				},
			},
		},
		Events: []EventDescription{
			{
				Type:           "Normal",
				Reason:         "Scheduled",
				Age:            "24h",
				From:           "default-scheduler",
				Message:        "Successfully assigned default/nginx-deployment-66b6c48dd5-abcde to worker-node-01",
				Count:          1,
				FirstTimestamp: eventTime1.Format(time.RFC3339),
				LastTimestamp:  eventTime1.Format(time.RFC3339),
			},
			{
				Type:           "Normal",
				Reason:         "Pulled",
				Age:            "24h",
				From:           "kubelet",
				Message:        "Container image \"nginx:1.21\" already present on machine",
				Count:          1,
				FirstTimestamp: eventTime1.Format(time.RFC3339),
				LastTimestamp:  eventTime1.Format(time.RFC3339),
			},
			{
				Type:           "Normal",
				Reason:         "Created",
				Age:            "24h",
				From:           "kubelet",
				Message:        "Created container nginx",
				Count:          1,
				FirstTimestamp: eventTime1.Format(time.RFC3339),
				LastTimestamp:  eventTime1.Format(time.RFC3339),
			},
			{
				Type:           "Normal",
				Reason:         "Started",
				Age:            "24h",
				From:           "kubelet",
				Message:        "Started container nginx",
				Count:          1,
				FirstTimestamp: eventTime1.Format(time.RFC3339),
				LastTimestamp:  eventTime1.Format(time.RFC3339),
			},
			{
				Type:           "Warning",
				Reason:         "Unhealthy",
				Age:            "12h",
				From:           "kubelet",
				Message:        "Liveness probe failed: HTTP probe failed with statuscode: 500",
				Count:          3,
				FirstTimestamp: eventTime2.Format(time.RFC3339),
				LastTimestamp:  eventTime2.Format(time.RFC3339),
			},
			{
				Type:           "Normal",
				Reason:         "Killing",
				Age:            "12h",
				From:           "kubelet",
				Message:        "Container nginx failed liveness probe, will be restarted",
				Count:          1,
				FirstTimestamp: eventTime2.Format(time.RFC3339),
				LastTimestamp:  eventTime2.Format(time.RFC3339),
			},
			{
				Type:           "Normal",
				Reason:         "Pulled",
				Age:            "30m",
				From:           "kubelet",
				Message:        "Container image \"nginx:1.21\" already present on machine",
				Count:          2,
				FirstTimestamp: eventTime3.Format(time.RFC3339),
				LastTimestamp:  eventTime3.Format(time.RFC3339),
			},
		},
		Conditions: []PodConditionDescription{
			{
				Type:               "Initialized",
				Status:             "True",
				LastProbeTime:      creationTime.Format(time.RFC3339),
				LastTransitionTime: creationTime.Format(time.RFC3339),
			},
			{
				Type:               "Ready",
				Status:             "True",
				LastProbeTime:      creationTime.Format(time.RFC3339),
				LastTransitionTime: creationTime.Add(30 * time.Second).Format(time.RFC3339),
			},
			{
				Type:               "ContainersReady",
				Status:             "True",
				LastProbeTime:      creationTime.Format(time.RFC3339),
				LastTransitionTime: creationTime.Add(30 * time.Second).Format(time.RFC3339),
			},
			{
				Type:               "PodScheduled",
				Status:             "True",
				LastProbeTime:      creationTime.Format(time.RFC3339),
				LastTransitionTime: creationTime.Format(time.RFC3339),
			},
		},
		Volumes: []VolumeDescription{
			{
				Name:        "nginx-config",
				Type:        "ConfigMap",
				Source:      "nginx-config",
				Description: "ConfigMap nginx-config",
			},
			{
				Name:        "nginx-data",
				Type:        "PersistentVolumeClaim",
				Source:      "nginx-data-pvc",
				Description: "PVC nginx-data-pvc",
			},
			{
				Name:        "default-token-xyz123",
				Type:        "Secret",
				Source:      "default-token-xyz123",
				Description: "Secret default-token-xyz123",
			},
		},
		Tolerations: []TolerationDescription{
			{
				Key:      "node.kubernetes.io/not-ready",
				Operator: "Exists",
				Effect:   "NoExecute",
				TolerationSeconds: 300,
			},
			{
				Key:      "node.kubernetes.io/unreachable",
				Operator: "Exists",
				Effect:   "NoExecute",
				TolerationSeconds: 300,
			},
		},
	}

	return mockResponse, nil
}

// PrintMockPodDescriptionWithComments 打印带有字段说明的模拟Pod描述响应
func PrintMockPodDescriptionWithComments() string {
	mockPod, _ := GenerateMockPodDescription()
	
	// 将Pod描述转换为JSON
	podJSON, _ := json.MarshalIndent(mockPod, "", "  ")
	
	// 添加字段说明
	explanation := `
/*
Pod描述响应字段说明:

PodName: Pod的名称，通常包含部署名称和随机生成的标识符
Namespace: Pod所在的命名空间
Status: Pod的当前状态，如Running、Pending、Failed等
CreationTimestamp: Pod的创建时间，ISO8601格式
Labels: Pod的标签，用于选择和分组
  - app: 应用名称
  - pod-template-hash: 部署模板的哈希值，用于版本控制
  - tier: 应用层级，如frontend、backend
  - environment: 环境类型，如production、staging
Annotations: Pod的注解，提供额外的元数据
  - kubernetes.io/config.seen: Kubernetes首次看到此配置的时间
  - kubernetes.io/config.source: 配置来源
  - prometheus.io/scrape: 是否允许Prometheus抓取指标
  - prometheus.io/port: Prometheus抓取指标的端口
  - sidecar.istio.io/inject: 是否注入Istio sidecar
NodeName: 运行Pod的节点名称
IP: Pod的IP地址
QoS: 服务质量类别(Quality of Service)，如Guaranteed、Burstable、BestEffort

Containers: Pod中的容器列表
  - Name: 容器名称
  - Image: 容器镜像
  - ImageID: 容器镜像的唯一标识符
  - ContainerID: 容器的唯一标识符
  - State: 容器的当前状态，如Running、Waiting、Terminated
  - StateDetails: 容器状态的详细信息
    - State: 当前状态类型，如Running、Waiting、Terminated
    - Running: 运行状态详情（当State为Running时有值）
      - StartedAt: 容器开始运行的时间
    - Waiting: 等待状态详情（当State为Waiting时有值）
      - Reason: 等待原因
      - Message: 等待消息
    - Terminated: 终止状态详情（当State为Terminated时有值）
      - Reason: 终止原因
      - ExitCode: 终止退出码
      - Signal: 终止信号
      - StartedAt: 终止开始时间
      - FinishedAt: 终止结束时间
      - Message: 终止消息
    - LastState: 上一个状态详情
      - State: 上一个状态类型
      - Waiting: 上一个等待状态详情（如果有）
      - Terminated: 上一个终止状态详情（如果有）
  - Ready: 容器是否就绪
  - RestartCount: 容器重启次数
  - Command: 容器启动命令
  - Args: 容器启动参数
  - WorkingDir: 容器工作目录
  
  Ports: 容器暴露的端口
    - Name: 端口名称
    - ContainerPort: 容器端口号
    - Protocol: 协议，如TCP、UDP
    - HostPort: 主机端口号(如果设置)
  
  Resources: 容器资源配置
    Requests: 容器请求的最小资源
      - CPU: CPU请求量，如100m表示0.1核
      - Memory: 内存请求量，如128Mi表示128兆字节
    Limits: 容器可使用的最大资源
      - CPU: CPU限制，如200m表示0.2核
      - Memory: 内存限制，如256Mi表示256兆字节
  
  Mounts: 容器的挂载点
    - Name: 挂载的卷名称
    - MountPath: 挂载路径
    - ReadOnly: 是否只读
  - SecurityContext: 容器安全上下文设置
    - Privileged: 是否以特权模式运行容器
    - RunAsUser: 容器运行的用户ID
    - RunAsGroup: 容器运行的组ID
    - RunAsNonRoot: 是否必须以非root用户运行
    - ReadOnlyRootFilesystem: 是否使用只读根文件系统
    - AllowPrivilegeEscalation: 是否允许特权提升
    - CapabilitiesAdd: 添加的Linux能力列表
    - CapabilitiesDrop: 删除的Linux能力列表
    - SeccompProfile: 安全计算配置
    - AppArmorProfile: AppArmor配置

  - LivenessProbe: 存活探针，检测容器是否运行
    - Type: 探针类型，如HTTP、TCP、Exec
    - Path: HTTP路径(HTTP探针)
    - Port: 端口
    - Scheme: 协议方案，如HTTP、HTTPS
    - InitialDelaySeconds: 容器启动后首次执行探测的等待时间
    - TimeoutSeconds: 探测超时时间
    - PeriodSeconds: 执行探测的频率
    - SuccessThreshold: 探测成功的阈值
    - FailureThreshold: 探测失败的阈值
  
  ReadinessProbe: 就绪探针，检测容器是否准备好接收流量
    (字段同LivenessProbe)
  
  Env: 环境变量
    - Name: 环境变量名称
    - Value: 环境变量值
    - ValueFrom: 环境变量值的来源，如Secret、ConfigMap等

Events: Pod相关事件
  - Type: 事件类型，如Normal、Warning
  - Reason: 事件原因
  - Age: 事件年龄
  - From: 事件来源
  - Message: 事件消息
  - Count: 事件发生次数
  - FirstTimestamp: 首次发生时间
  - LastTimestamp: 最后发生时间

Conditions: Pod状态条件
  - Type: 条件类型，如Initialized、Ready、ContainersReady、PodScheduled
  - Status: 条件状态，如True、False、Unknown
  - LastProbeTime: 最后一次探测时间
  - LastTransitionTime: 最后一次状态转换时间
  - Reason: 状态原因(可选)
  - Message: 状态消息(可选)

Volumes: Pod使用的存储卷
  - Name: 卷名称
  - Type: 卷类型，如ConfigMap、Secret、PersistentVolumeClaim、EmptyDir
  - Source: 卷来源
  - Description: 卷描述

Tolerations: Pod的容忍设置，允许Pod调度到有特定污点的节点
  - Key: 污点键
  - Operator: 操作符，如Exists、Equal
  - Value: 污点值
  - Effect: 效果，如NoSchedule、PreferNoSchedule、NoExecute
  - TolerationSeconds: 容忍时间(秒)
*/
`
	
	return string(podJSON) + explanation
}
