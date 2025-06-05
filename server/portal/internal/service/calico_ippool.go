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

// IPPool相关常量
const (
	defaultBlockSize      = 26
	defaultIPIPMode       = "Never"
	defaultVXLANMode      = "Never"
	dateTimeFormat        = "2006-01-02T15:04:05Z"
	maxConcurrentRequests = 10 // 限制并发请求数
)

// CalicoIPPoolService 提供Calico IPPool信息检索服务
// 支持多集群环境下的 IPPool 查询与转换。
// 该服务设计用于集中管理多个K8s集群的场景，每个集群都有独立的客户端配置。
type CalicoIPPoolService struct {
	clusterClients map[string]*kubernetes.Clientset
	dynamicClients map[string]dynamic.Interface
	restConfigs    map[string]*rest.Config
	mu             sync.RWMutex // 保护动态客户端和配置的并发访问
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

// IPPoolGVR 定义Calico IPPool资源的GVR (GroupVersionResource)
var ipPoolGVR = schema.GroupVersionResource{
	Group:    "crd.projectcalico.org",
	Version:  "v1",
	Resource: "ippools",
}

// NewCalicoIPPoolService 创建CalicoIPPoolService实例
// 参数:
//   - clusterClients: 集群名称到Kubernetes客户端的映射
//   - restConfigs: (可选) 集群名称到REST配置的映射，可传入nil
//
// 返回:
//   - CalicoIPPoolService实例
func NewCalicoIPPoolService(clusterClients map[string]*kubernetes.Clientset, restConfigs map[string]*rest.Config) *CalicoIPPoolService {
	service := &CalicoIPPoolService{
		clusterClients: clusterClients,
		dynamicClients: make(map[string]dynamic.Interface, len(clusterClients)),
		restConfigs:    make(map[string]*rest.Config, len(clusterClients)),
	}

	// 如果提供了REST配置，直接使用
	for name, config := range restConfigs {
		if config != nil {
			// 保存配置副本
			configCopy := rest.CopyConfig(config)
			service.restConfigs[name] = configCopy

			// 创建动态客户端
			if dynamicClient, err := dynamic.NewForConfig(configCopy); err == nil {
				service.dynamicClients[name] = dynamicClient
	}
		}
	}

	return service
}

// NewCalicoIPPoolServiceLegacy 创建CalicoIPPoolService实例的兼容版本
// 这是为了保持向后兼容性提供的简化版构造函数
// 参数:
//   - clusterClients: 集群名称到Kubernetes客户端的映射
//
// 返回:
//   - CalicoIPPoolService实例
func NewCalicoIPPoolServiceLegacy(clusterClients map[string]*kubernetes.Clientset) *CalicoIPPoolService {
	return NewCalicoIPPoolService(clusterClients, nil)
}

// GetAllIPPools 获取所有管理的集群中的Calico IPPool信息
// 该函数会并发地从所有集群中获取IPPool信息，并返回一个按集群名称分组的映射
func (s *CalicoIPPoolService) GetAllIPPools(ctx context.Context) (map[string][]IPPoolInfo, error) {
	// TODO: 优化建议 - 并发控制逻辑可以简化，错误处理可以更优雅
	fmt.Printf("[DEBUG] GetAllIPPools: 开始获取所有集群的 IPPool 信息\n")
	
	clusterNames := make([]string, 0, len(s.clusterClients))
	for clusterName := range s.clusterClients {
		clusterNames = append(clusterNames, clusterName)
	}
	
	fmt.Printf("[DEBUG] GetAllIPPools: 发现 %d 个集群: %v\n", len(clusterNames), clusterNames)

	result := make(map[string][]IPPoolInfo, len(clusterNames))
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentRequests)
	errCh := make(chan error, len(clusterNames))

	fmt.Printf("[DEBUG] GetAllIPPools: 开始并发处理，最大并发数: %d\n", maxConcurrentRequests)

	for _, clusterName := range clusterNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			fmt.Printf("[DEBUG] GetAllIPPools: 开始处理集群 %s\n", name)
			
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ipPools, err := s.GetClusterIPPools(ctx, name)
			if err != nil {
				fmt.Printf("[ERROR] GetAllIPPools: 集群 %s 处理失败: %v\n", name, err)
				errCh <- fmt.Errorf("failed to get IPPools from cluster %s: %w", name, err)
				return
			}
			
			fmt.Printf("[DEBUG] GetAllIPPools: 集群 %s 成功获取 %d 个 IPPool\n", name, len(ipPools))
			resultMu.Lock()
			result[name] = ipPools
			resultMu.Unlock()
		}(clusterName)
	}

	wg.Wait()
	close(errCh)

	fmt.Printf("[DEBUG] GetAllIPPools: 所有 goroutine 完成，开始处理错误\n")

	var firstErr error
	errCount := 0
	for err := range errCh {
		if errCount == 0 {
			firstErr = err
		}
		errCount++
	}
	if errCount > 0 {
		fmt.Printf("[ERROR] GetAllIPPools: 遇到 %d 个错误，第一个错误: %v\n", errCount, firstErr)
		return result, fmt.Errorf("encountered %d errors, first error: %w", errCount, firstErr)
	}
	
	fmt.Printf("[DEBUG] GetAllIPPools: 成功完成，返回 %d 个集群的结果\n", len(result))
	return result, nil
}

// GetClusterIPPools 获取指定集群的Calico IPPool信息
func (s *CalicoIPPoolService) GetClusterIPPools(ctx context.Context, clusterName string) ([]IPPoolInfo, error) {
	// 获取动态客户端
		dynamicClient, err := s.getOrCreateDynamicClient(clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to get dynamic client for cluster %s: %w", clusterName, err)
	}

	// 获取IPPool列表
	ipPoolList, err := dynamicClient.Resource(ipPoolGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list IPPools in cluster %s: %w", clusterName, err)
	}

	// 转换结果
	ipPools := make([]IPPoolInfo, 0, len(ipPoolList.Items))
	for _, ipPool := range ipPoolList.Items {
		poolInfo, err := convertToIPPoolInfo(ipPool, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to convert IPPool in cluster %s: %w", clusterName, err)
		}
		ipPools = append(ipPools, poolInfo)
	}

	return ipPools, nil
}

// GetIPPoolByName 根据名称获取指定集群中的IPPool信息
func (s *CalicoIPPoolService) GetIPPoolByName(ctx context.Context, clusterName, ipPoolName string) (*IPPoolInfo, error) {
	// 获取动态客户端
	dynamicClient, err := s.getOrCreateDynamicClient(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get dynamic client for cluster %s: %w", clusterName, err)
	}

	// 获取IPPool
	ipPool, err := dynamicClient.Resource(ipPoolGVR).Get(ctx, ipPoolName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get IPPool %s in cluster %s: %w", ipPoolName, clusterName, err)
	}

	// 转换为IPPoolInfo
	info, err := convertToIPPoolInfo(*ipPool, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to convert IPPool %s in cluster %s: %w", ipPoolName, clusterName, err)
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

// getOrCreateDynamicClient 获取或创建指定集群的动态客户端
// 该方法尝试以下步骤来获取动态客户端:
// 1. 首先尝试从缓存获取已存在的客户端
// 2. 如果没有缓存的客户端，则尝试使用InClusterConfig创建新客户端
// 3. 新创建的客户端会被缓存以供后续使用
// 参数:
//   - clusterName: 集群名称
//
// 返回:
//   - dynamic.Interface: 动态客户端
//   - error: 错误信息，如果无法创建客户端
func (s *CalicoIPPoolService) getOrCreateDynamicClient(clusterName string) (dynamic.Interface, error) {
	// 尝试从缓存获取客户端
	s.mu.RLock()
	client, exists := s.dynamicClients[clusterName]
		s.mu.RUnlock()

	if exists {
		return client, nil
	}

	// 使用 InClusterConfig 创建一个默认配置
	config, err := rest.InClusterConfig()
		if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config for %s: %w", clusterName, err)
	}

	// 创建动态客户端
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// 缓存动态客户端
	s.mu.Lock()
	s.dynamicClients[clusterName] = dynamicClient
	s.restConfigs[clusterName] = config
	s.mu.Unlock()

	return dynamicClient, nil
}

// convertToIPPoolInfo 将Unstructured对象转换为IPPoolInfo
func convertToIPPoolInfo(obj unstructured.Unstructured, clusterName string) (IPPoolInfo, error) {
	// 初始化基本信息
	info := IPPoolInfo{
		Name:              obj.GetName(),
		ClusterName:       clusterName,
		CreationTimestamp: obj.GetCreationTimestamp().Format(dateTimeFormat),
		Labels:            obj.GetLabels(),
		NodeSelector:      make(map[string]string),
		Raw:               obj.Object,
		IPIPMode:          defaultIPIPMode,  // 默认值
		VXLANMode:         defaultVXLANMode, // 默认值
		BlockSize:         defaultBlockSize, // 默认值
	}

	// 获取spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return info, fmt.Errorf("failed to get spec: %w", err)
	}
	if !found {
		return info, fmt.Errorf("spec not found")
	}

	// 提取CIDR并确定IP版本
	if err := extractCIDRInfo(&info, spec); err != nil {
		return info, err
	}

	// 提取NAT出站设置
	extractBoolField(&info.NATOutgoing, spec, "natOutgoing")

	// 提取禁用状态
	extractBoolField(&info.Disabled, spec, "disabled")

	// 提取节点选择器
	extractNodeSelector(&info, spec)

	// 提取IPIP和VXLAN模式
	extractStringField(&info.IPIPMode, spec, "ipipMode")
	extractStringField(&info.VXLANMode, spec, "vxlanMode")

	// 提取块大小
	extractBlockSize(&info, spec)

	return info, nil
}

// extractCIDRInfo 提取CIDR信息并确定IP版本
func extractCIDRInfo(info *IPPoolInfo, spec map[string]interface{}) error {
	cidr, found, err := unstructured.NestedString(spec, "cidr")
	if err != nil {
		return fmt.Errorf("failed to extract CIDR: %w", err)
	}
	if !found {
		return fmt.Errorf("CIDR not found")
	}

		info.CIDR = cidr
		info.IPVersion = 4
		if strings.Contains(cidr, ":") {
			info.IPVersion = 6
		}

	return nil
}

// extractBoolField 从spec中提取布尔字段
func extractBoolField(target *bool, spec map[string]interface{}, fieldName string) {
	if value, found, _ := unstructured.NestedBool(spec, fieldName); found {
		*target = value
	}
}

// extractStringField 从spec中提取字符串字段
func extractStringField(target *string, spec map[string]interface{}, fieldName string) {
	if value, found, _ := unstructured.NestedString(spec, fieldName); found {
		*target = value
	}
}

// extractNodeSelector 提取节点选择器
func extractNodeSelector(info *IPPoolInfo, spec map[string]interface{}) {
	// 尝试获取字符串形式的选择器
	nodeSelector, found, _ := unstructured.NestedString(spec, "nodeSelector")
	if found && nodeSelector != "" {
		info.NodeSelector = parseNodeSelector(nodeSelector)
		return
	}

	// 尝试获取映射形式的选择器
	nodeSelectorMap, found, _ := unstructured.NestedMap(spec, "nodeSelector")
	if found {
		for k, v := range nodeSelectorMap {
			if strValue, ok := v.(string); ok {
				info.NodeSelector[k] = strValue
			}
		}
	}
	}

// extractBlockSize 提取块大小
func extractBlockSize(info *IPPoolInfo, spec map[string]interface{}) {
	blockSize, found, _ := unstructured.NestedInt64(spec, "blockSize")
	if found && blockSize > 0 {
		info.BlockSize = int(blockSize)
	}
}

// parseNodeSelector 将节点选择器字符串解析为键值对映射
func parseNodeSelector(selector string) map[string]string {
	result := make(map[string]string)
	if selector == "" {
		return result
	}

	// 处理 key==value 或 key=value 格式
	if strings.Contains(selector, "==") || strings.Contains(selector, "=") {
		return parseEqualitySelector(selector)
	}

	// 处理 key in (value1, value2) 格式
	if inIndex := strings.Index(selector, " in "); inIndex > 0 {
		return parseInSelector(selector, inIndex)
	}

	return result
}

// parseEqualitySelector 解析等式选择器 (key=value 或 key==value)
func parseEqualitySelector(selector string) map[string]string {
	result := make(map[string]string)

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

// parseInSelector 解析in选择器 (key in (value1, value2))
func parseInSelector(selector string, inIndex int) map[string]string {
	result := make(map[string]string)

		key := strings.TrimSpace(selector[:inIndex])
		valuesStart := strings.Index(selector, "(")
		valuesEnd := strings.LastIndex(selector, ")")

	if key == "" || valuesStart <= inIndex || valuesEnd <= valuesStart {
		return result
	}

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

	return result
}
