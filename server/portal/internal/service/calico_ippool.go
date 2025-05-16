package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// IPPoolGroupRule 定义IPPool分组规则
type IPPoolGroupRule struct {
	// 组名称
	GroupName string
	// 正则表达式模式列表，用于匹配IPPool名称
	// 只要匹配其中一个模式，就会被分到这个组
	Patterns []string
	// 编译后的正则表达式列表
	regexps []*regexp.Regexp
	// 组描述
	Description string
}

// IPPoolGroup 表示IPPool组
type IPPoolGroup struct {
	// 组名称
	Name string `json:"name"`
	// 组描述
	Description string `json:"description,omitempty"`
	// 组内IPPool列表
	IPPools []IPPoolInfo `json:"ipPools"`
	// 所属集群
	ClusterName string `json:"clusterName"`
}

// CalicoIPPoolService 提供Calico IPPool信息检索服务
type CalicoIPPoolService struct {
	// 集群名称到Kubernetes客户端的映射
	clusterClients map[string]*kubernetes.Clientset
	// 缓存集群名称到Dynamic客户端的映射
	dynamicClients map[string]dynamic.Interface
	// IPPool分组规则
	groupRules []IPPoolGroupRule
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
	// 原始数据
	Raw map[string]interface{} `json:"raw,omitempty"`
}

// NewCalicoIPPoolService 创建Calico IPPool服务
func NewCalicoIPPoolService(clusterClients map[string]*kubernetes.Clientset) *CalicoIPPoolService {
	// 初始化服务
	service := &CalicoIPPoolService{
		clusterClients: clusterClients,
		dynamicClients: make(map[string]dynamic.Interface),
		groupRules:     make([]IPPoolGroupRule, 0),
	}

	// 对每个集群创建dynamic客户端
	for clusterName, clientset := range clusterClients {
		// 获取REST配置
		restConfig, err := service.getRESTConfigFromClientset(clientset)
		if err == nil && restConfig != nil {
			dynamicClient, err := dynamic.NewForConfig(restConfig)
			if err == nil {
				service.dynamicClients[clusterName] = dynamicClient
			}
		}
	}

	return service
}

// GetAllIPPools 获取所有管理的集群中的Calico IPPool信息
func (s *CalicoIPPoolService) GetAllIPPools(ctx context.Context) (map[string][]IPPoolInfo, error) {
	// 获取所有集群名称
	clusterNames := make([]string, 0, len(s.clusterClients))
	for clusterName := range s.clusterClients {
		clusterNames = append(clusterNames, clusterName)
	}

	// 存储所有集群的IPPool信息
	result := make(map[string][]IPPoolInfo)
	var wg sync.WaitGroup
	var resultMu sync.Mutex
	var errors []error
	var errorsMu sync.Mutex

	// 并行获取所有集群的IPPool信息
	for _, clusterName := range clusterNames {
		wg.Add(1)
		go func(clusterName string) {
			defer wg.Done()

			// 获取单个集群的IPPool信息
			ipPools, err := s.GetClusterIPPools(ctx, clusterName)
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("cluster %s: %w", clusterName, err))
				errorsMu.Unlock()
				return
			}

			// 存储结果
			resultMu.Lock()
			result[clusterName] = ipPools
			resultMu.Unlock()
		}(clusterName)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 如果有错误，返回第一个错误
	if len(errors) > 0 {
		return result, errors[0]
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
		CreationTimestamp: obj.GetCreationTimestamp().Format("2006-01-02T15:04:05Z"),
		Raw:               obj.Object,
	}

	// 从spec中提取字段
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return info, fmt.Errorf("spec not found in IPPool: %v", err)
	}

	// 提取CIDR
	if cidr, found, _ := unstructured.NestedString(spec, "cidr"); found {
		info.CIDR = cidr
		// 根据CIDR推断IP版本
		if strings.Contains(cidr, ":") {
			info.IPVersion = 6
		} else {
			info.IPVersion = 4
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
		// 解析节点选择器字符串为map
		// 节点选择器通常是一个标签选择器表达式，如"key == value"或"key in (value1, value2)"
		// 这里我们做一个简化处理，假设它是简单的key=value形式
		info.NodeSelector = parseNodeSelector(nodeSelector)
	} else {
		// 初始化为空映射
		info.NodeSelector = make(map[string]string)
		
		// 如果有完整的节点选择器对象，则尝试提取
		if nodeSelectorMap, found, _ := unstructured.NestedMap(spec, "nodeSelector"); found {
			for k, v := range nodeSelectorMap {
				if strValue, ok := v.(string); ok {
					info.NodeSelector[k] = strValue
				}
			}
		}
	}

	// 提取IPIP模式
	if ipipMode, found, _ := unstructured.NestedString(spec, "ipipMode"); found {
		info.IPIPMode = ipipMode
	} else {
		info.IPIPMode = "Never"
	}

	// 提取VXLAN模式
	if vxlanMode, found, _ := unstructured.NestedString(spec, "vxlanMode"); found {
		info.VXLANMode = vxlanMode
	} else {
		info.VXLANMode = "Never"
	}

	// 提取块大小
	if blockSize, found, _ := unstructured.NestedInt64(spec, "blockSize"); found {
		info.BlockSize = int(blockSize)
	} else {
		// 默认块大小
		info.BlockSize = 26
	}

	return info, nil
}

// 从现有的客户端映射创建CalicoIPPoolService
func NewCalicoIPPoolServiceFromClients(clientsets map[string]*kubernetes.Clientset) *CalicoIPPoolService {
	return NewCalicoIPPoolService(clientsets)
}

// RegisterIPPoolGroupRule 注册一个IPPool分组规则
func (s *CalicoIPPoolService) RegisterIPPoolGroupRule(groupName string, patterns ...string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("at least one pattern must be provided")
	}
	
	// 编译所有正则表达式
	regexps := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		regexp, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}
		regexps = append(regexps, regexp)
	}
	
	// 添加规则
	s.groupRules = append(s.groupRules, IPPoolGroupRule{
		GroupName:   groupName,
		Patterns:    patterns,
		regexps:     regexps,
		Description: fmt.Sprintf("%s group (matches %d patterns)", groupName, len(patterns)),
	})
	
	return nil
}

// RegisterIPPoolGroupRuleWithDescription 注册一个带描述的IPPool分组规则
func (s *CalicoIPPoolService) RegisterIPPoolGroupRuleWithDescription(groupName, description string, patterns ...string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("at least one pattern must be provided")
	}
	
	// 编译所有正则表达式
	regexps := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		regexp, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}
		regexps = append(regexps, regexp)
	}
	
	// 添加规则
	s.groupRules = append(s.groupRules, IPPoolGroupRule{
		GroupName:   groupName,
		Patterns:    patterns,
		regexps:     regexps,
		Description: description,
	})
	
	return nil
}

// RegisterDefaultGroupRules 注册默认的分组规则
func (s *CalicoIPPoolService) RegisterDefaultGroupRules() {
	// 注册一些常见的分组规则
	_ = s.RegisterIPPoolGroupRuleWithDescription(
		"app-general",
		"General application pools",
		"^appgeneral-.*$", "^app-general-.*$",
	)
	
	_ = s.RegisterIPPoolGroupRuleWithDescription(
		"app-specific",
		"Application-specific pools",
		"^app-.*$", "^application-.*$",
	)
	
	_ = s.RegisterIPPoolGroupRuleWithDescription(
		"system",
		"System and infrastructure pools",
		"^system-.*$", "^kube-system-.*$", "^infra-.*$",
	)
	
	_ = s.RegisterIPPoolGroupRule("default", "^default$")
}

// GetIPPoolGroups 获取指定集群的所有IPPool分组
func (s *CalicoIPPoolService) GetIPPoolGroups(ctx context.Context, clusterName string) ([]IPPoolGroup, error) {
	// 获取集群的所有IPPool
	ipPools, err := s.GetClusterIPPools(ctx, clusterName)
	if err != nil {
		return nil, err
	}
	
	// 按规则分组
	return s.groupIPPools(ipPools, clusterName), nil
}

// GetIPPoolGroupByName 根据组名称获取指定集群的IPPool分组
func (s *CalicoIPPoolService) GetIPPoolGroupByName(ctx context.Context, clusterName, groupName string) (*IPPoolGroup, error) {
	// 获取集群的所有IPPool
	ipPools, err := s.GetClusterIPPools(ctx, clusterName)
	if err != nil {
		return nil, err
	}
	
	// 按规则分组
	groups := s.groupIPPools(ipPools, clusterName)
	
	// 查找指定的组
	for _, group := range groups {
		if group.Name == groupName {
			return &group, nil
		}
	}
	
	return nil, fmt.Errorf("group %s not found in cluster %s", groupName, clusterName)
}

// groupIPPools 根据规则将IPPool分组
func (s *CalicoIPPoolService) groupIPPools(ipPools []IPPoolInfo, clusterName string) []IPPoolGroup {
	// 初始化组映射
	groupMap := make(map[string][]IPPoolInfo)
	
	// 将每个IPPool分配到相应的组
	for _, ipPool := range ipPools {
		assigned := false
		
		// 尝试匹配每个规则
		for _, rule := range s.groupRules {
			// 尝试该规则的所有正则表达式
			for _, re := range rule.regexps {
				if re != nil && re.MatchString(ipPool.Name) {
					groupMap[rule.GroupName] = append(groupMap[rule.GroupName], ipPool)
					assigned = true
					break
				}
			}
			
			// 如果已经分配到组，则不再尝试其他规则
			if assigned {
				break
			}
		}
		
		// 如果没有匹配的规则，则放入“其他”组
		if !assigned {
			groupMap["other"] = append(groupMap["other"], ipPool)
		}
	}
	
	// 转换为结果列表
	result := make([]IPPoolGroup, 0, len(groupMap))
	for groupName, pools := range groupMap {
		// 查找规则以获取描述
		description := fmt.Sprintf("%s group containing %d IPPools", groupName, len(pools))
		for _, rule := range s.groupRules {
			if rule.GroupName == groupName && rule.Description != "" {
				description = rule.Description
				break
			}
		}
		
		result = append(result, IPPoolGroup{
			Name:        groupName,
			Description: description,
			IPPools:     pools,
			ClusterName: clusterName,
		})
	}
	
	return result
}

// IPVersion 表示IP版本的枚举类型
type IPVersion int

// IP版本常量
const (
	IPVersionAll IPVersion = 0 // 所有IP版本
	IPVersion4   IPVersion = 4 // IPv4
	IPVersion6   IPVersion = 6 // IPv6
)

// FindIPPoolsByGroupPattern 根据组模式查找IPPools
func (s *CalicoIPPoolService) FindIPPoolsByGroupPattern(ctx context.Context, clusterName, pattern string, ipVersion IPVersion) ([]IPPoolInfo, error) {
	// 编译模式
	regexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %s: %w", pattern, err)
	}
	
	// 获取集群的所有IPPool
	ipPools, err := s.GetClusterIPPools(ctx, clusterName)
	if err != nil {
		return nil, err
	}
	
	// 过滤匹配的IPPool
	var result []IPPoolInfo
	for _, ipPool := range ipPools {
		// 先检查IP版本
		if ipVersion != IPVersionAll && ipPool.IPVersion != int(ipVersion) {
			continue
		}
		
		// 再检查名称模式
		if regexp.MatchString(ipPool.Name) {
			result = append(result, ipPool)
		}
	}
	
	return result, nil
}

// FindIPPoolsByGroupPatternAll 根据组模式查找所有版本的IPPools
// 为了兼容性保留的方法
func (s *CalicoIPPoolService) FindIPPoolsByGroupPatternAll(ctx context.Context, clusterName, pattern string) ([]IPPoolInfo, error) {
	return s.FindIPPoolsByGroupPattern(ctx, clusterName, pattern, IPVersionAll)
}

// 解析节点选择器字符串为map[string]string
func parseNodeSelector(selector string) map[string]string {
	result := make(map[string]string)
	
	// 如果是空的，直接返回空映射
	if selector == "" {
		return result
	}
	
	// 处理简单的key==value或key=value形式
	// 注意：这是一个简化的处理，实际的节点选择器可能更复杂
	if strings.Contains(selector, "==") {
		parts := strings.Split(selector, "==")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	} else if strings.Contains(selector, "=") {
		parts := strings.Split(selector, "=")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	} else if strings.Contains(selector, " in ") {
		// 处理key in (value1, value2)形式
		parts := strings.Split(selector, " in ")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			// 提取括号内的值
			valueStr := strings.TrimSpace(parts[1])
			valueStr = strings.Trim(valueStr, "()")
			valueStr = strings.Trim(valueStr, "[]")
			// 将多个值合并为逗号分隔的字符串
			values := strings.Split(valueStr, ",")
			for i, v := range values {
				values[i] = strings.TrimSpace(v)
			}
			result[key] = strings.Join(values, ",")
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
