package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Constants for PodDescribeService
const (
	ErrPodNotFoundMsg = "pod %s in namespace %s not found"
)

// PodDescribeService 提供Pod描述服务，类似kubectl describe pod的功能
type PodDescribeService struct {
	// 缓存不同集群的客户端连接
	clientCache map[string]*kubernetes.Clientset
}

// PodDescribeRequest 请求参数
type PodDescribeRequest struct {
	ClusterName string `json:"clusterName" form:"clusterName" binding:"required"` // 集群名称
	Namespace   string `json:"namespace" form:"namespace" binding:"required"`     // 命名空间
	PodName     string `json:"podName" form:"podName" binding:"required"`         // Pod名称
	KubeConfig  string `json:"kubeConfig" form:"kubeConfig"`                      // 可选的kubeconfig内容
}

// PodDescribeResponse Pod描述响应
type PodDescribeResponse struct {
	PodName           string                  `json:"podName"`           // Pod名称
	Namespace         string                  `json:"namespace"`         // 命名空间
	Status            string                  `json:"status"`            // 状态
	CreationTimestamp string                  `json:"creationTimestamp"` // 创建时间
	Labels            map[string]string       `json:"labels"`            // 标签
	Annotations       map[string]string       `json:"annotations"`       // 注解
	NodeName          string                  `json:"nodeName"`          // 节点名称
	IP                string                  `json:"ip"`                // Pod IP
	QoS               string                  `json:"qos"`               // QoS类别
	Containers        []ContainerDescription  `json:"containers"`        // 容器列表
	Events            []EventDescription      `json:"events"`            // 事件列表
	Conditions        []PodConditionDescription `json:"conditions"`      // Pod状态条件
	Volumes           []VolumeDescription     `json:"volumes"`           // 存储卷
	Tolerations       []TolerationDescription `json:"tolerations"`       // 容忍
}

// ContainerDescription 容器描述
type ContainerDescription struct {
	Name            string                 `json:"name"`            // 容器名称
	Image           string                 `json:"image"`           // 镜像
	ImageID         string                 `json:"imageId"`         // 镜像ID
	ContainerID     string                 `json:"containerId"`     // 容器ID
	State           string                 `json:"state"`           // 状态
	StateDetails    ContainerStateDetails  `json:"stateDetails"`    // 详细状态信息
	Ready           bool                   `json:"ready"`           // 是否就绪
	RestartCount    int32                  `json:"restartCount"`    // 重启次数
	Ports           []PortDescription      `json:"ports"`           // 端口
	Resources       ResourceDescription     `json:"resources"`       // 资源
	Mounts          []MountDescription      `json:"mounts"`         // 挂载
	LivenessProbe   *ProbeDescription       `json:"livenessProbe,omitempty"`  // 存活探针
	ReadinessProbe  *ProbeDescription       `json:"readinessProbe,omitempty"` // 就绪探针
	StartupProbe    *ProbeDescription       `json:"startupProbe,omitempty"`   // 启动探针
	Env             []EnvDescription        `json:"env"`            // 环境变量
	SecurityContext *SecurityContextDescription `json:"securityContext,omitempty"` // 安全上下文
	Command         []string               `json:"command,omitempty"`         // 容器命令
	Args            []string               `json:"args,omitempty"`            // 容器参数
	WorkingDir      string                 `json:"workingDir,omitempty"`      // 工作目录
}

// PortDescription 端口描述
type PortDescription struct {
	Name          string `json:"name,omitempty"`          // 端口名称
	ContainerPort int32  `json:"containerPort"`           // 容器端口
	Protocol      string `json:"protocol"`                // 协议
	HostPort      int32  `json:"hostPort,omitempty"`      // 主机端口
}

// ResourceDescription 资源描述
type ResourceDescription struct {
	Requests ResourceQuantity `json:"requests"` // 请求
	Limits   ResourceQuantity `json:"limits"`   // 限制
}

// ResourceQuantity 资源数量
type ResourceQuantity struct {
	CPU    string `json:"cpu"`    // CPU
	Memory string `json:"memory"` // 内存
}

// MountDescription 挂载描述
type MountDescription struct {
	Name      string `json:"name"`      // 名称
	MountPath string `json:"mountPath"` // 挂载路径
	ReadOnly  bool   `json:"readOnly"`  // 是否只读
}

// ProbeDescription 探针描述
type ProbeDescription struct {
	Type                string `json:"type"`                // 探针类型
	Path                string `json:"path,omitempty"`      // HTTP路径
	Port                int32  `json:"port,omitempty"`      // 端口
	Host                string `json:"host,omitempty"`      // 主机
	Scheme              string `json:"scheme,omitempty"`    // 协议
	InitialDelaySeconds int32  `json:"initialDelaySeconds"` // 初始延迟
	TimeoutSeconds      int32  `json:"timeoutSeconds"`      // 超时
	PeriodSeconds       int32  `json:"periodSeconds"`       // 周期
	SuccessThreshold    int32  `json:"successThreshold"`    // 成功阈值
	FailureThreshold    int32  `json:"failureThreshold"`    // 失败阈值
}

// EnvDescription 环境变量描述
type EnvDescription struct {
	Name      string `json:"name"`                // 名称
	Value     string `json:"value,omitempty"`     // 值
	ValueFrom string `json:"valueFrom,omitempty"` // 值来源
}

// EventDescription 事件描述
type EventDescription struct {
	Type           string `json:"type"`           // 类型
	Reason         string `json:"reason"`         // 原因
	Age            string `json:"age"`            // 时间
	From           string `json:"from"`           // 来源
	Message        string `json:"message"`        // 消息
	Count          int32  `json:"count"`          // 计数
	FirstTimestamp string `json:"firstTimestamp"` // 首次时间
	LastTimestamp  string `json:"lastTimestamp"`  // 最后时间
}

// PodConditionDescription Pod状态条件描述
type PodConditionDescription struct {
	Type               string `json:"type"`               // 类型
	Status             string `json:"status"`             // 状态
	LastProbeTime      string `json:"lastProbeTime"`      // 最后探测时间
	LastTransitionTime string `json:"lastTransitionTime"` // 最后转换时间
	Reason             string `json:"reason,omitempty"`   // 原因
	Message            string `json:"message,omitempty"`  // 消息
}

// VolumeDescription 存储卷描述
type VolumeDescription struct {
	Name        string `json:"name"`        // 名称
	Type        string `json:"type"`        // 类型
	Source      string `json:"source"`      // 来源
	Description string `json:"description"` // 描述
}

// TolerationDescription 容忍描述
type TolerationDescription struct {
	Key               string `json:"key,omitempty"`               // 键
	Operator          string `json:"operator"`                    // 操作符
	Value             string `json:"value,omitempty"`             // 值
	Effect            string `json:"effect,omitempty"`            // 效果
	TolerationSeconds int64  `json:"tolerationSeconds,omitempty"` // 容忍秒数
}

// ContainerStateDetails 容器状态详细信息
type ContainerStateDetails struct {
	State       string        `json:"state"`                  // 当前状态: Running, Waiting, Terminated
	Running     *RunningState `json:"running,omitempty"`      // 运行状态详情，当State为Running时有值
	Waiting     *WaitingState `json:"waiting,omitempty"`      // 等待状态详情，当State为Waiting时有值
	Terminated  *TerminatedState `json:"terminated,omitempty"` // 终止状态详情，当State为Terminated时有值
	LastState   *LastState   `json:"lastState,omitempty"`    // 上一个状态详情
}

// RunningState 运行状态详情
type RunningState struct {
	StartedAt string `json:"startedAt"` // 运行开始时间
}

// WaitingState 等待状态详情
type WaitingState struct {
	Reason  string `json:"reason"`            // 等待原因
	Message string `json:"message,omitempty"` // 等待消息
}

// TerminatedState 终止状态详情
type TerminatedState struct {
	Reason    string `json:"reason"`              // 终止原因
	ExitCode  int32  `json:"exitCode"`            // 终止退出码
	Signal    string `json:"signal,omitempty"`     // 终止信号
	StartedAt string `json:"startedAt,omitempty"`  // 终止开始时间
	FinishedAt string `json:"finishedAt,omitempty"` // 终止结束时间
	Message   string `json:"message,omitempty"`    // 终止消息
}

// LastState 上一个状态详情
type LastState struct {
	State     string          `json:"state"`               // 状态类型: Running, Waiting, Terminated
	Waiting   *WaitingState   `json:"waiting,omitempty"`   // 等待状态详情
	Terminated *TerminatedState `json:"terminated,omitempty"` // 终止状态详情
}

// SecurityContextDescription 安全上下文描述
type SecurityContextDescription struct {
	Privileged             bool    `json:"privileged,omitempty"`             // 是否特权容器
	RunAsUser              int64   `json:"runAsUser,omitempty"`              // 运行用户ID
	RunAsGroup             int64   `json:"runAsGroup,omitempty"`             // 运行组ID
	RunAsNonRoot           bool    `json:"runAsNonRoot,omitempty"`           // 是否以非root用户运行
	ReadOnlyRootFilesystem bool    `json:"readOnlyRootFilesystem,omitempty"` // 只读根文件系统
	AllowPrivilegeEscalation bool  `json:"allowPrivilegeEscalation,omitempty"` // 允许特权提升
	CapabilitiesAdd        []string `json:"capabilitiesAdd,omitempty"`        // 添加的Linux能力
	CapabilitiesDrop       []string `json:"capabilitiesDrop,omitempty"`       // 删除的Linux能力
	SeccompProfile         string   `json:"seccompProfile,omitempty"`         // Seccomp配置文件
	AppArmorProfile        string   `json:"appArmorProfile,omitempty"`        // AppArmor配置文件
}

// NewPodDescribeService 创建Pod描述服务
func NewPodDescribeService() *PodDescribeService {
	return &PodDescribeService{
		clientCache: make(map[string]*kubernetes.Clientset),
	}
}

// DescribePod 获取Pod详细描述，类似kubectl describe pod
func (s *PodDescribeService) DescribePod(ctx context.Context, request *PodDescribeRequest) (*PodDescribeResponse, error) {
	// 获取k8s客户端
	client, err := s.getClient(request.ClusterName, request.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// 获取Pod信息
	pod, err := client.CoreV1().Pods(request.Namespace).Get(ctx, request.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(ErrPodNotFoundMsg, request.PodName, request.Namespace)
	}

	// 获取Pod相关事件
	events, err := s.getPodEvents(ctx, client, pod)
	if err != nil {
		// 记录错误但继续处理，因为Pod信息仍然可用
		fmt.Printf("Warning: failed to get pod events: %v\n", err)
	}

	// 构建响应
	response := s.buildPodDescribeResponse(pod, events)
	return response, nil
}

// getClient 获取Kubernetes客户端
func (s *PodDescribeService) getClient(clusterName, kubeConfig string) (*kubernetes.Clientset, error) {
	// 如果已有缓存的客户端，直接返回
	if client, ok := s.clientCache[clusterName]; ok && kubeConfig == "" {
		return client, nil
	}

	var config *rest.Config
	var err error

	// 如果提供了kubeConfig内容，使用它创建客户端
	if kubeConfig != "" {
		config, err = clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
		if err != nil {
			return nil, fmt.Errorf("failed to create config from kubeconfig: %w", err)
		}
	} else {
		// 否则尝试从默认位置加载配置
		// 在实际应用中，这里可能需要根据clusterName从数据库或配置中心获取对应集群的kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		if clusterName != "" {
			configOverrides.CurrentContext = clusterName
		}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	// 创建客户端
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 缓存客户端
	if kubeConfig == "" {
		s.clientCache[clusterName] = client
	}

	return client, nil
}

// getPodEvents 获取Pod相关事件
func (s *PodDescribeService) getPodEvents(ctx context.Context, client *kubernetes.Clientset, pod *corev1.Pod) ([]corev1.Event, error) {
	// 获取与Pod相关的事件
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=Pod", 
		pod.Name, pod.Namespace)
	events, err := client.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}

// buildPodDescribeResponse 构建Pod描述响应
func (s *PodDescribeService) buildPodDescribeResponse(pod *corev1.Pod, events []corev1.Event) *PodDescribeResponse {
	response := &PodDescribeResponse{
		PodName:           pod.Name,
		Namespace:         pod.Namespace,
		Status:            string(pod.Status.Phase),
		CreationTimestamp: pod.CreationTimestamp.Format(time.RFC3339),
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		NodeName:          pod.Spec.NodeName,
		IP:                pod.Status.PodIP,
		QoS:               string(pod.Status.QOSClass),
		Containers:        s.buildContainersDescription(pod),
		Events:            s.buildEventsDescription(events),
		Conditions:        s.buildPodConditionsDescription(pod.Status.Conditions),
		Volumes:           s.buildVolumesDescription(pod.Spec.Volumes),
		Tolerations:       s.buildTolerationsDescription(pod.Spec.Tolerations),
	}
	return response
}

// buildContainersDescription 构建容器描述
func (s *PodDescribeService) buildContainersDescription(pod *corev1.Pod) []ContainerDescription {
	containers := make([]ContainerDescription, 0, len(pod.Spec.Containers))
	
	// 创建容器状态映射，用于快速查找
	containerStatuses := make(map[string]corev1.ContainerStatus)
	for _, status := range pod.Status.ContainerStatuses {
		containerStatuses[status.Name] = status
	}
	
	for _, container := range pod.Spec.Containers {
		containerDesc := ContainerDescription{
			Name:       container.Name,
			Image:      container.Image,
			Command:    container.Command,
			Args:       container.Args,
			WorkingDir: container.WorkingDir,
			Ports:      s.buildPortsDescription(container.Ports),
			Resources: ResourceDescription{
				Requests: ResourceQuantity{
					CPU:    container.Resources.Requests.Cpu().String(),
					Memory: container.Resources.Requests.Memory().String(),
				},
				Limits: ResourceQuantity{
					CPU:    container.Resources.Limits.Cpu().String(),
					Memory: container.Resources.Limits.Memory().String(),
				},
			},
			Mounts: s.buildMountsDescription(container.VolumeMounts),
			Env:    s.buildEnvDescription(container.Env),
		}
		
		// 添加安全上下文信息
		if container.SecurityContext != nil {
			secContext := &SecurityContextDescription{}
			
			// 填充安全上下文信息
			if container.SecurityContext.Privileged != nil {
				secContext.Privileged = *container.SecurityContext.Privileged
			}
			if container.SecurityContext.RunAsUser != nil {
				secContext.RunAsUser = *container.SecurityContext.RunAsUser
			}
			if container.SecurityContext.RunAsGroup != nil {
				secContext.RunAsGroup = *container.SecurityContext.RunAsGroup
			}
			if container.SecurityContext.RunAsNonRoot != nil {
				secContext.RunAsNonRoot = *container.SecurityContext.RunAsNonRoot
			}
			if container.SecurityContext.ReadOnlyRootFilesystem != nil {
				secContext.ReadOnlyRootFilesystem = *container.SecurityContext.ReadOnlyRootFilesystem
			}
			if container.SecurityContext.AllowPrivilegeEscalation != nil {
				secContext.AllowPrivilegeEscalation = *container.SecurityContext.AllowPrivilegeEscalation
			}
			
			// 处理Linux能力
			if container.SecurityContext.Capabilities != nil {
				if len(container.SecurityContext.Capabilities.Add) > 0 {
					secContext.CapabilitiesAdd = make([]string, len(container.SecurityContext.Capabilities.Add))
					for i, cap := range container.SecurityContext.Capabilities.Add {
						secContext.CapabilitiesAdd[i] = string(cap)
					}
				}
				if len(container.SecurityContext.Capabilities.Drop) > 0 {
					secContext.CapabilitiesDrop = make([]string, len(container.SecurityContext.Capabilities.Drop))
					for i, cap := range container.SecurityContext.Capabilities.Drop {
						secContext.CapabilitiesDrop[i] = string(cap)
					}
				}
			}
			
			// 处理SeccompProfile
			if container.SecurityContext.SeccompProfile != nil {
				secContext.SeccompProfile = string(container.SecurityContext.SeccompProfile.Type)
				if container.SecurityContext.SeccompProfile.LocalhostProfile != nil {
					secContext.SeccompProfile += ": " + *container.SecurityContext.SeccompProfile.LocalhostProfile
				}
			}
			
			// 获取AppArmor配置（从注解中提取）
			if pod.Annotations != nil {
				appArmorKey := fmt.Sprintf("container.apparmor.security.beta.kubernetes.io/%s", container.Name)
				if profile, ok := pod.Annotations[appArmorKey]; ok {
					secContext.AppArmorProfile = profile
				}
			}
			
			containerDesc.SecurityContext = secContext
		}
		
		// 添加探针信息
		if container.LivenessProbe != nil {
			containerDesc.LivenessProbe = s.buildProbeDescription(container.LivenessProbe, "Liveness")
		}
		if container.ReadinessProbe != nil {
			containerDesc.ReadinessProbe = s.buildProbeDescription(container.ReadinessProbe, "Readiness")
		}
		if container.StartupProbe != nil {
			containerDesc.StartupProbe = s.buildProbeDescription(container.StartupProbe, "Startup")
		}
		
		// 添加状态信息
		if status, ok := containerStatuses[container.Name]; ok {
			containerDesc.ImageID = status.ImageID
			containerDesc.ContainerID = status.ContainerID
			containerDesc.Ready = status.Ready
			containerDesc.RestartCount = status.RestartCount
			
			// 创建详细状态信息
			stateDetails := ContainerStateDetails{}
			
			// 处理当前状态
			if status.State.Running != nil {
				containerDesc.State = "Running"
				stateDetails.State = "Running"
				stateDetails.Running = &RunningState{
					StartedAt: status.State.Running.StartedAt.Format(time.RFC3339),
				}
			} else if status.State.Waiting != nil {
				containerDesc.State = fmt.Sprintf("Waiting: %s", status.State.Waiting.Reason)
				stateDetails.State = "Waiting"
				stateDetails.Waiting = &WaitingState{
					Reason:  status.State.Waiting.Reason,
					Message: status.State.Waiting.Message,
				}
			} else if status.State.Terminated != nil {
				containerDesc.State = fmt.Sprintf("Terminated: %s (exit code: %d)", 
					status.State.Terminated.Reason, status.State.Terminated.ExitCode)
				stateDetails.State = "Terminated"
				stateDetails.Terminated = &TerminatedState{
					Reason:    status.State.Terminated.Reason,
					ExitCode:  status.State.Terminated.ExitCode,
					Signal:    string(status.State.Terminated.Signal),
					StartedAt: status.State.Terminated.StartedAt.Format(time.RFC3339),
					FinishedAt: status.State.Terminated.FinishedAt.Format(time.RFC3339),
					Message:   status.State.Terminated.Message,
				}
			}
			
			// 处理上一个状态
			hasLastState := false
			lastState := &LastState{}
			
			if status.LastTerminationState.Running != nil {
				lastState.State = "Running"
				hasLastState = true
			} else if status.LastTerminationState.Waiting != nil {
				lastState.State = "Waiting"
				lastState.Waiting = &WaitingState{
					Reason:  status.LastTerminationState.Waiting.Reason,
					Message: status.LastTerminationState.Waiting.Message,
				}
				hasLastState = true
			} else if status.LastTerminationState.Terminated != nil {
				lastState.State = "Terminated"
				lastState.Terminated = &TerminatedState{
					Reason:    status.LastTerminationState.Terminated.Reason,
					ExitCode:  status.LastTerminationState.Terminated.ExitCode,
					Signal:    string(status.LastTerminationState.Terminated.Signal),
					StartedAt: status.LastTerminationState.Terminated.StartedAt.Format(time.RFC3339),
					FinishedAt: status.LastTerminationState.Terminated.FinishedAt.Format(time.RFC3339),
					Message:   status.LastTerminationState.Terminated.Message,
				}
				hasLastState = true
			}
			
			if hasLastState {
				stateDetails.LastState = lastState
			}
			
			containerDesc.StateDetails = stateDetails
		}
		
		containers = append(containers, containerDesc)
	}
	
	return containers
}

// buildPortsDescription 构建端口描述
func (s *PodDescribeService) buildPortsDescription(ports []corev1.ContainerPort) []PortDescription {
	portDescs := make([]PortDescription, 0, len(ports))
	for _, port := range ports {
		portDesc := PortDescription{
			Name:          port.Name,
			ContainerPort: port.ContainerPort,
			Protocol:      string(port.Protocol),
			HostPort:      port.HostPort,
		}
		portDescs = append(portDescs, portDesc)
	}
	return portDescs
}

// buildMountsDescription 构建挂载描述
func (s *PodDescribeService) buildMountsDescription(mounts []corev1.VolumeMount) []MountDescription {
	mountDescs := make([]MountDescription, 0, len(mounts))
	for _, mount := range mounts {
		mountDesc := MountDescription{
			Name:      mount.Name,
			MountPath: mount.MountPath,
			ReadOnly:  mount.ReadOnly,
		}
		mountDescs = append(mountDescs, mountDesc)
	}
	return mountDescs
}

// buildProbeDescription 构建探针描述
func (s *PodDescribeService) buildProbeDescription(probe *corev1.Probe, probeType string) *ProbeDescription {
	probeDesc := &ProbeDescription{
		Type:                probeType,
		InitialDelaySeconds: probe.InitialDelaySeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}
	
	// 根据探针类型设置特定字段
	if probe.HTTPGet != nil {
		probeDesc.Path = probe.HTTPGet.Path
		probeDesc.Port = probe.HTTPGet.Port.IntVal
		probeDesc.Host = probe.HTTPGet.Host
		probeDesc.Scheme = string(probe.HTTPGet.Scheme)
	} else if probe.TCPSocket != nil {
		probeDesc.Port = probe.TCPSocket.Port.IntVal
		probeDesc.Host = probe.TCPSocket.Host
	} else if probe.Exec != nil {
		// 对于Exec探针，我们可以将命令作为Path字段
		probeDesc.Path = strings.Join(probe.Exec.Command, " ")
	}
	
	return probeDesc
}

// buildEnvDescription 构建环境变量描述
func (s *PodDescribeService) buildEnvDescription(envVars []corev1.EnvVar) []EnvDescription {
	envDescs := make([]EnvDescription, 0, len(envVars))
	for _, env := range envVars {
		envDesc := EnvDescription{
			Name:  env.Name,
			Value: env.Value,
		}
		
		// 处理环境变量引用
		if env.ValueFrom != nil {
			if env.ValueFrom.ConfigMapKeyRef != nil {
				envDesc.ValueFrom = fmt.Sprintf("ConfigMap %s, key %s", 
					env.ValueFrom.ConfigMapKeyRef.Name, env.ValueFrom.ConfigMapKeyRef.Key)
			} else if env.ValueFrom.SecretKeyRef != nil {
				envDesc.ValueFrom = fmt.Sprintf("Secret %s, key %s", 
					env.ValueFrom.SecretKeyRef.Name, env.ValueFrom.SecretKeyRef.Key)
			} else if env.ValueFrom.FieldRef != nil {
				envDesc.ValueFrom = fmt.Sprintf("Field %s", env.ValueFrom.FieldRef.FieldPath)
			} else if env.ValueFrom.ResourceFieldRef != nil {
				envDesc.ValueFrom = fmt.Sprintf("Resource %s", env.ValueFrom.ResourceFieldRef.Resource)
			}
		}
		
		envDescs = append(envDescs, envDesc)
	}
	return envDescs
}

// buildEventsDescription 构建事件描述
func (s *PodDescribeService) buildEventsDescription(events []corev1.Event) []EventDescription {
	eventDescs := make([]EventDescription, 0, len(events))
	for _, event := range events {
		// 计算事件年龄
		age := "unknown"
		if !event.FirstTimestamp.IsZero() {
			age = formatDuration(time.Since(event.FirstTimestamp.Time))
		}
		
		eventDesc := EventDescription{
			Type:           event.Type,
			Reason:         event.Reason,
			Age:            age,
			From:           event.Source.Component,
			Message:        event.Message,
			Count:          event.Count,
			FirstTimestamp: event.FirstTimestamp.Format(time.RFC3339),
			LastTimestamp:  event.LastTimestamp.Format(time.RFC3339),
		}
		eventDescs = append(eventDescs, eventDesc)
	}
	return eventDescs
}

// buildPodConditionsDescription 构建Pod状态条件描述
func (s *PodDescribeService) buildPodConditionsDescription(conditions []corev1.PodCondition) []PodConditionDescription {
	conditionDescs := make([]PodConditionDescription, 0, len(conditions))
	for _, condition := range conditions {
		conditionDesc := PodConditionDescription{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastProbeTime:      condition.LastProbeTime.Format(time.RFC3339),
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		conditionDescs = append(conditionDescs, conditionDesc)
	}
	return conditionDescs
}

// buildVolumesDescription 构建存储卷描述
func (s *PodDescribeService) buildVolumesDescription(volumes []corev1.Volume) []VolumeDescription {
	volumeDescs := make([]VolumeDescription, 0, len(volumes))
	for _, volume := range volumes {
		volumeDesc := VolumeDescription{
			Name: volume.Name,
		}
		
		// 根据卷类型设置特定字段
		if volume.ConfigMap != nil {
			volumeDesc.Type = "ConfigMap"
			volumeDesc.Source = volume.ConfigMap.Name
			volumeDesc.Description = fmt.Sprintf("ConfigMap %s", volume.ConfigMap.Name)
		} else if volume.Secret != nil {
			volumeDesc.Type = "Secret"
			volumeDesc.Source = volume.Secret.SecretName
			volumeDesc.Description = fmt.Sprintf("Secret %s", volume.Secret.SecretName)
		} else if volume.PersistentVolumeClaim != nil {
			volumeDesc.Type = "PersistentVolumeClaim"
			volumeDesc.Source = volume.PersistentVolumeClaim.ClaimName
			volumeDesc.Description = fmt.Sprintf("PVC %s", volume.PersistentVolumeClaim.ClaimName)
		} else if volume.HostPath != nil {
			volumeDesc.Type = "HostPath"
			volumeDesc.Source = volume.HostPath.Path
			volumeDesc.Description = fmt.Sprintf("HostPath %s", volume.HostPath.Path)
		} else if volume.EmptyDir != nil {
			volumeDesc.Type = "EmptyDir"
			medium := "none"
			if volume.EmptyDir.Medium != "" {
				medium = string(volume.EmptyDir.Medium)
			}
			volumeDesc.Description = fmt.Sprintf("EmptyDir (medium: %s)", medium)
		} else {
			volumeDesc.Type = "Other"
			volumeDesc.Description = "Other volume type"
		}
		
		volumeDescs = append(volumeDescs, volumeDesc)
	}
	return volumeDescs
}

// buildTolerationsDescription 构建容忍描述
func (s *PodDescribeService) buildTolerationsDescription(tolerations []corev1.Toleration) []TolerationDescription {
	tolerationDescs := make([]TolerationDescription, 0, len(tolerations))
	for _, toleration := range tolerations {
		tolerationDesc := TolerationDescription{
			Key:      toleration.Key,
			Operator: string(toleration.Operator),
			Value:    toleration.Value,
			Effect:   string(toleration.Effect),
		}
		if toleration.TolerationSeconds != nil {
			tolerationDesc.TolerationSeconds = *toleration.TolerationSeconds
		}
		tolerationDescs = append(tolerationDescs, tolerationDesc)
	}
	return tolerationDescs
}

// formatDuration 格式化持续时间，类似kubectl的格式
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	
	hours := d / time.Hour
	d -= hours * time.Hour
	
	minutes := d / time.Minute
	d -= minutes * time.Minute
	
	seconds := d / time.Second
	
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
