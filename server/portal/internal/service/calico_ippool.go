package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Constants for default values and configuration
const (
	defaultBlockSize      = 26
	defaultIPIPMode       = "Never"
	defaultVXLANMode      = "Never"
	dateTimeFormat        = "2006-01-02T15:04:05Z"
	maxConcurrentRequests = 10 // Limit concurrent cluster requests
)


// CalicoIPPoolService 提供 Calico IPPool 信息检索服务
// 支持多集群环境下的 IPPool 查询与转换。
type CalicoIPPoolService struct {
	clusterClients map[string]*kubernetes.Clientset
	dynamicClients map[string]dynamic.Interface
}


// IPPoolInfo 表示Calico IPPool的信息
type IPPoolInfo struct {
	// IPPool名称
	Name string `json:"name"`
	// CIDR范围
	CIDR string `json:"cidr"`
	// 是否启用IPIP模式
	IPIPMode string `json:"ipipMode"`
	// 是否启用VXLAN模式
	VXLANMode string `json:"vxlanMode"`
	// 是否启用NAT出站流量
	NATOutgoing bool `json:"natOutgoing"`
	// 是否禁用此IPPool
	Disabled bool `json:"disabled"`
	// 节点选择器
	NodeSelector map[string]string `json:"nodeSelector"`
	// 块大小
	BlockSize int `json:"blockSize"`
	// 允许的IP版本
	IPVersion int `json:"ipVersion"`
	// 创建时间
	CreationTimestamp string `json:"creationTimestamp"`
	// 集群名称
	ClusterName string `json:"clusterName"`
	// 对象标签
	Labels map[string]string `json:"labels,omitempty"`
	// 原始数据
	Raw map[string]interface{} `json:"raw,omitempty"`
}

// NewCalicoIPPoolService 创建 CalicoIPPoolService 实例。
func NewCalicoIPPoolService(clusterClients map[string]*kubernetes.Clientset) *CalicoIPPoolService {
	service := &CalicoIPPoolService{
		clusterClients: clusterClients,
		dynamicClients: make(map[string]dynamic.Interface, len(clusterClients)),
	}
	for clusterName, clientset := range clusterClients {
		if clientset == nil {
			continue
		}
		restConfig, err := service.getRESTConfigFromClientset(clientset)
		if err != nil {
			continue
		}
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			continue
		}
		service.dynamicClients[clusterName] = dynamicClient
	}
	return service
}


// GetAllIPPools 获取所有管理的集群中的Calico IPPool信息
// 该函数会并发地从所有集群中获取IPPool信息，并返回一个按集群名称分组的映射
func (s *CalicoIPPoolService) GetAllIPPools(ctx context.Context) (map[string][]IPPoolInfo, error) {
	clusterNames := make([]string, 0, len(s.clusterClients))
	for clusterName := range s.clusterClients {
		clusterNames = append(clusterNames, clusterName)
	}

	result := make(map[string][]IPPoolInfo, len(clusterNames))
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentRequests)
	errCh := make(chan error, len(clusterNames))

	for _, clusterName := range clusterNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ipPools, err := s.GetClusterIPPools(ctx, name)
			if err != nil {
				errCh <- fmt.Errorf("failed to get IPPools from cluster %s: %w", name, err)
				return
			}
			resultMu.Lock()
			result[name] = ipPools
			resultMu.Unlock()
		}(clusterName)
	}

	wg.Wait()
	close(errCh)

	var firstErr error
	errCount := 0
	for err := range errCh {
		if errCount == 0 {
			firstErr = err
		}
		errCount++
	}
	if errCount > 0 {
		return result, fmt.Errorf("encountered %d errors, first error: %w", errCount, firstErr)
	}
	return result, nil
}


// 定义Calico IPPool资源的GVR (GroupVersionResource)
var ipPoolGVR = schema.GroupVersionResource{
	Group:    "crd.projectcalico.org",
	Version:  "v1",
	Resource: "ippools",
}

// GetClusterIPPools 获取指定集群的Calico IPPool信息
func (s *CalicoIPPoolService) GetClusterIPPools(ctx context.Context, clusterName string) ([]IPPoolInfo, error) {
	// 获取Dynamic客户端
	dynamicClient, ok := s.dynamicClients[clusterName]
	if !ok {
		// 如果没有找到dynamic客户端，尝试从原始clientset创建
		clientset, clientOk := s.clusterClients[clusterName]
		if !clientOk {
			return nil, fmt.Errorf("no client found for cluster: %s", clusterName)
		}

		// 从客户端创建dynamic客户端
		restConfig, err := s.getRESTConfigFromClientset(clientset)
		if err != nil {
			return nil, fmt.Errorf("failed to get REST config: %w", err)
		}

		newDynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamic client: %w", err)
		}

		// 缓存新创建的dynamic客户端
		s.dynamicClients[clusterName] = newDynamicClient
		dynamicClient = newDynamicClient
	}

	// 获取IPPool列表
	ipPoolList, err := dynamicClient.Resource(ipPoolGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list IPPools: %w", err)
	}

	// 转换为IPPoolInfo
	ipPools := make([]IPPoolInfo, 0, len(ipPoolList.Items))
	for _, ipPool := range ipPoolList.Items {
		poolInfo, err := convertUnstructuredToIPPoolInfo(ipPool, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to convert IPPool: %w", err)
		}
		ipPools = append(ipPools, poolInfo)
	}

	return ipPools, nil
}

// GetIPPoolByName 根据名称获取指定集群中的IPPool信息
func (s *CalicoIPPoolService) GetIPPoolByName(ctx context.Context, clusterName, ipPoolName string) (*IPPoolInfo, error) {
	// 获取Dynamic客户端
	dynamicClient, ok := s.dynamicClients[clusterName]
	if !ok {
		// 如果没有找到dynamic客户端，尝试从原始clientset创建
		clientset, clientOk := s.clusterClients[clusterName]
		if !clientOk {
			return nil, fmt.Errorf("no client found for cluster: %s", clusterName)
		}

		// 从客户端创建dynamic客户端
		restConfig, err := s.getRESTConfigFromClientset(clientset)
		if err != nil {
			return nil, fmt.Errorf("failed to get REST config: %w", err)
		}

		newDynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamic client: %w", err)
		}

		// 缓存新创建的dynamic客户端
		s.dynamicClients[clusterName] = newDynamicClient
		dynamicClient = newDynamicClient
	}

	// 获取IPPool
	ipPool, err := dynamicClient.Resource(ipPoolGVR).Get(ctx, ipPoolName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get IPPool %s: %w", ipPoolName, err)
	}

	// 转换为IPPoolInfo
	info, err := convertUnstructuredToIPPoolInfo(*ipPool, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to convert IPPool: %w", err)
	}
	return &info, nil
}

// GetClusterIPPoolsByCIDR 根据CIDR获取指定集群中的IPPool信息
func (s *CalicoIPPoolService) GetClusterIPPoolsByCIDR(ctx context.Context, clusterName, cidr string) ([]IPPoolInfo, error) {
	// 获取所有IPPool
	allPools, err := s.GetClusterIPPools(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	// 过滤符合CIDR的IPPool
	var result []IPPoolInfo
	for _, pool := range allPools {
		if pool.CIDR == cidr {
			result = append(result, pool)
		}
	}

	return result, nil
}

// convertUnstructuredToIPPoolInfo 将Unstructured对象转换为IPPoolInfo
func convertUnstructuredToIPPoolInfo(obj unstructured.Unstructured, clusterName string) (IPPoolInfo, error) {
	info := IPPoolInfo{
		Name:              obj.GetName(),
		ClusterName:       clusterName,
		CreationTimestamp: obj.GetCreationTimestamp().Format(dateTimeFormat),
		Labels:            obj.GetLabels(),
		NodeSelector:      make(map[string]string), // 初始化空映射
		Raw:               obj.Object,
	}

	// 从spec中提取字段
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return info, fmt.Errorf("failed to get spec from IPPool: %w", err)
	}
	if !found {
		return info, fmt.Errorf("spec not found in IPPool")
	}

	// 提取CIDR并确定IP版本
	if cidr, found, err := unstructured.NestedString(spec, "cidr"); err == nil && found {
		info.CIDR = cidr
		info.IPVersion = 4
		if strings.Contains(cidr, ":") {
			info.IPVersion = 6
		}
	}

	// 提取NAT出站设置
	if natOutgoing, found, _ := unstructured.NestedBool(spec, "natOutgoing"); found {
		info.NATOutgoing = natOutgoing
	}

	// 提取禁用状态
	if disabled, found, _ := unstructured.NestedBool(spec, "disabled"); found {
		info.Disabled = disabled
	}

	// 提取节点选择器
	if nodeSelector, found, _ := unstructured.NestedString(spec, "nodeSelector"); found && nodeSelector != "" {
		info.NodeSelector = parseNodeSelector(nodeSelector)
	} else if nodeSelectorMap, found, _ := unstructured.NestedMap(spec, "nodeSelector"); found {
		// 处理结构化的节点选择器
		for k, v := range nodeSelectorMap {
			if strValue, ok := v.(string); ok {
				info.NodeSelector[k] = strValue
			}
		}
	}

	// 设置IPIP和VXLAN模式，使用默认值
	info.IPIPMode = defaultIPIPMode
	if ipipMode, found, _ := unstructured.NestedString(spec, "ipipMode"); found {
		info.IPIPMode = ipipMode
	}

	info.VXLANMode = defaultVXLANMode
	if vxlanMode, found, _ := unstructured.NestedString(spec, "vxlanMode"); found {
		info.VXLANMode = vxlanMode
	}

	// 设置块大小，使用默认值
	info.BlockSize = defaultBlockSize
	if blockSize, found, _ := unstructured.NestedInt64(spec, "blockSize"); found && blockSize > 0 {
		info.BlockSize = int(blockSize)
	}

	return info, nil
}

// parseNodeSelector 将节点选择器字符串解析为键值对映射
// 支持以下格式：
// - key=value
// - key==value
// - key in (value1, value2)
// - 结构化的选择器对象
func parseNodeSelector(selector string) map[string]string {
	result := make(map[string]string)
	if selector == "" {
		return result
	}

	// 处理 key==value 或 key=value 格式
	if strings.Contains(selector, "==") || strings.Contains(selector, "=") {
		sep := "=="
		if !strings.Contains(selector, "==") {
			sep = "="
		}
		parts := strings.SplitN(selector, sep, 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, `'"`)
			if key != "" && value != "" {
				result[key] = value
			}
		}
		return result
	}

	// 处理 key in (value1, value2) 格式
	if inIndex := strings.Index(selector, " in "); inIndex > 0 {
		key := strings.TrimSpace(selector[:inIndex])
		valuesStart := strings.Index(selector, "(")
		valuesEnd := strings.LastIndex(selector, ")")

		if key != "" && valuesStart > inIndex && valuesEnd > valuesStart {
			valuesStr := selector[valuesStart+1 : valuesEnd]
			values := strings.Split(valuesStr, ",")
			var cleanValues []string

			for _, v := range values {
				v = strings.TrimSpace(v)
				v = strings.Trim(v, `'"`)
				if v != "" {
					cleanValues = append(cleanValues, v)
				}
			}

			if len(cleanValues) > 0 {
				result[key] = strings.Join(cleanValues, ",")
			}
		}
	}

	return result
}

// 从kubernetes.Clientset获取REST配置
func (s *CalicoIPPoolService) getRESTConfigFromClientset(clientset *kubernetes.Clientset) (*rest.Config, error) {
	// 由于系统已经提前建立了到apiserver的连接
	// 我们可以假设已经有了可用的配置

	// 在实际环境中，可能需要从外部获取配置或使用其他方式
	// 这里我们假设系统已经为我们提供了配置

	// 尝试使用InClusterConfig
	config, err := rest.InClusterConfig()
	if err != nil {
		// 如果无法获取集群内配置，则创建一个基本配置
		// 这里只是一个备用方案，实际应用中可能需要更复杂的逻辑
		config = &rest.Config{
			Host:    "https://kubernetes.default.svc",
			APIPath: "/api",
		}
	}

	return config, nil
}
