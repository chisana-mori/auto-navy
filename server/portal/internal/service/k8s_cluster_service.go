package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// K8sClusterService K8s集群服务
type K8sClusterService struct {
	db *gorm.DB
}

// NewK8sClusterService 创建K8s集群服务实例
func NewK8sClusterService(db *gorm.DB) *K8sClusterService {
	return &K8sClusterService{db: db}
}

// GetK8sClusters 获取K8s集群列表
func (s *K8sClusterService) GetK8sClusters(ctx context.Context, query *K8sClusterQuery) (*K8sClusterListResponse, error) {
	// 创建一个带有超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var clusters []portal.K8sCluster
	var total int64

	// 构建查询
	db := s.db.WithContext(timeoutCtx).Model(&portal.K8sCluster{})

	// 应用过滤条件
	if query.ClusterName != "" {
		db = db.Where("clustername LIKE ?", "%"+query.ClusterName+"%")
	}
	if query.ClusterNameCn != "" {
		db = db.Where("clusternamecn LIKE ?", "%"+query.ClusterNameCn+"%")
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.ClusterType != "" {
		db = db.Where("clustertype = ?", query.ClusterType)
	}
	if query.Idc != "" {
		db = db.Where("idc = ?", query.Idc)
	}
	if query.Zone != "" {
		db = db.Where("zone = ?", query.Zone)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, NewServerError("获取集群总数失败", err)
	}

	// 应用分页
	offset := (query.Page - 1) * query.Size
	if err := db.Offset(offset).Limit(query.Size).Order("created_at DESC").Find(&clusters).Error; err != nil {
		return nil, NewServerError("获取集群列表失败", err)
	}

	// 转换为响应格式
	responses := make([]*K8sClusterResponse, len(clusters))
	for i, cluster := range clusters {
		responses[i] = convertToK8sClusterResponse(&cluster)
	}

	return &K8sClusterListResponse{
		List:  responses,
		Total: total,
		Page:  query.Page,
		Size:  query.Size,
	}, nil
}

// GetK8sClusterByID 根据ID获取K8s集群
func (s *K8sClusterService) GetK8sClusterByID(ctx context.Context, id int) (*K8sClusterResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var cluster portal.K8sCluster
	if err := s.db.WithContext(timeoutCtx).Preload("Nodes").First(&cluster, id).Error; err != nil {
		if IsNotFound(err) {
			return nil, NewNotFoundError("K8s集群", id)
		}
		return nil, NewServerError("获取集群失败", err)
	}

	return convertToK8sClusterResponse(&cluster), nil
}

// CreateK8sCluster 创建K8s集群
func (s *K8sClusterService) CreateK8sCluster(ctx context.Context, req *CreateK8sClusterRequest, username string) (*K8sClusterResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 检查集群ID是否已存在
	var existingCluster portal.K8sCluster
	if err := s.db.WithContext(timeoutCtx).Where("cluster_id = ?", req.ClusterID).First(&existingCluster).Error; err == nil {
		return nil, NewBadRequestError("集群ID已存在")
	} else if !IsNotFound(err) {
		return nil, NewServerError("检查集群ID失败", err)
	}

	// 转换为数据模型
	cluster := req.ToModel(username)

	// 创建集群
	if err := s.db.WithContext(timeoutCtx).Create(&cluster).Error; err != nil {
		return nil, NewServerError("创建集群失败", err)
	}

	return convertToK8sClusterResponse(&cluster), nil
}

// UpdateK8sCluster 更新K8s集群
func (s *K8sClusterService) UpdateK8sCluster(ctx context.Context, id int, req *UpdateK8sClusterRequest, username string) (*K8sClusterResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 检查集群是否存在
	var cluster portal.K8sCluster
	if err := s.db.WithContext(timeoutCtx).First(&cluster, id).Error; err != nil {
		if IsNotFound(err) {
			return nil, NewNotFoundError("K8s集群", id)
		}
		return nil, NewServerError("获取集群失败", err)
	}

	// 如果更新了集群ID，检查新的集群ID是否已存在
	if req.ClusterID != "" && req.ClusterID != cluster.ClusterID {
		var existingCluster portal.K8sCluster
		if err := s.db.WithContext(timeoutCtx).Where("cluster_id = ? AND id != ?", req.ClusterID, id).First(&existingCluster).Error; err == nil {
			return nil, NewBadRequestError("集群ID已存在")
		} else if !IsNotFound(err) {
			return nil, NewServerError("检查集群ID失败", err)
		}
	}

	// 更新字段
	updateData := req.ToUpdateMap(username)
	if err := s.db.WithContext(timeoutCtx).Model(&cluster).Updates(updateData).Error; err != nil {
		return nil, NewServerError("更新集群失败", err)
	}

	// 重新获取更新后的数据
	if err := s.db.WithContext(timeoutCtx).First(&cluster, id).Error; err != nil {
		return nil, NewServerError("获取更新后的集群失败", err)
	}

	return convertToK8sClusterResponse(&cluster), nil
}

// DeleteK8sCluster 删除K8s集群
func (s *K8sClusterService) DeleteK8sCluster(ctx context.Context, id int) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 检查集群是否存在
	var cluster portal.K8sCluster
	if err := s.db.WithContext(timeoutCtx).First(&cluster, id).Error; err != nil {
		if IsNotFound(err) {
			return NewNotFoundError("K8s集群", id)
		}
		return NewServerError("获取集群失败", err)
	}

	// 检查是否有关联的节点
	var nodeCount int64
	if err := s.db.WithContext(timeoutCtx).Model(&portal.K8sNode{}).Where("k8s_cluster_id = ?", id).Count(&nodeCount).Error; err != nil {
		return NewServerError("检查关联节点失败", err)
	}

	if nodeCount > 0 {
		return NewBadRequestError(fmt.Sprintf("无法删除集群，存在 %d 个关联节点", nodeCount))
	}

	// 删除集群
	if err := s.db.WithContext(timeoutCtx).Delete(&cluster).Error; err != nil {
		return NewServerError("删除集群失败", err)
	}

	return nil
}

// GetK8sClusterNodes 获取集群的节点列表
func (s *K8sClusterService) GetK8sClusterNodes(ctx context.Context, clusterID int) ([]*K8sNodeResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 检查集群是否存在
	var cluster portal.K8sCluster
	if err := s.db.WithContext(timeoutCtx).First(&cluster, clusterID).Error; err != nil {
		if IsNotFound(err) {
			return nil, NewNotFoundError("K8s集群", clusterID)
		}
		return nil, NewServerError("获取集群失败", err)
	}

	// 获取节点列表
	var nodes []portal.K8sNode
	if err := s.db.WithContext(timeoutCtx).Where("k8s_cluster_id = ?", clusterID).Order("created_at DESC").Find(&nodes).Error; err != nil {
		return nil, NewServerError("获取节点列表失败", err)
	}

	// 转换为响应格式
	responses := make([]*K8sNodeResponse, len(nodes))
	for i, node := range nodes {
		responses[i] = convertToK8sNodeResponse(&node)
	}

	return responses, nil
}

// convertToK8sClusterResponse 转换为集群响应格式
func convertToK8sClusterResponse(cluster *portal.K8sCluster) *K8sClusterResponse {
	response := &K8sClusterResponse{
		ID:                cluster.ID,
		ClusterID:         cluster.ClusterID,
		ClusterName:       cluster.ClusterName,
		ClusterNameCn:     cluster.ClusterNameCn,
		Alias:             cluster.Alias,
		ApiServer:         cluster.ApiServer,
		ApiServerVip:      cluster.ApiServerVip,
		EtcdServer:        cluster.EtcdServer,
		EtcdServerVip:     cluster.EtcdServerVip,
		IngressServername: cluster.IngressServername,
		IngressServerVip:  cluster.IngressServerVip,
		KubePromVersion:   cluster.KubePromVersion,
		PromServer:        cluster.PromServer,
		ThanosServer:      cluster.ThanosServer,
		Idc:               cluster.Idc,
		Zone:              cluster.Zone,
		Status:            cluster.Status,
		ClusterType:       cluster.ClusterType,
		KubeConfig:        cluster.KubeConfig,
		Desc:              cluster.Desc,
		Creator:           cluster.Creator,
		Group:             cluster.Group,
		EsServer:          cluster.EsServer,
		NetType:           cluster.NetType,
		Architecture:      cluster.Architecture,
		FlowType:          cluster.FlowType,
		NovaName:          cluster.NovaName,
		Priority:          cluster.Priority,
		ClusterGroup:      cluster.ClusterGroup,
		PodCidr:           cluster.PodCidr,
		ServiceCidr:       cluster.ServiceCidr,
		RrCicode:          cluster.RrCicode,
		RrGroup:           cluster.RrGroup,
		CreatedAt:         cluster.CreatedAt.String(),
		UpdatedAt:         cluster.UpdatedAt.String(),
	}

	// 如果包含节点信息，转换节点数据
	if len(cluster.Nodes) > 0 {
		response.Nodes = make([]*K8sNodeResponse, len(cluster.Nodes))
		for i, node := range cluster.Nodes {
			response.Nodes[i] = convertToK8sNodeResponse(&node)
		}
	}

	return response
}

// convertToK8sNodeResponse 转换为节点响应格式
func convertToK8sNodeResponse(node *portal.K8sNode) *K8sNodeResponse {
	return &K8sNodeResponse{
		ID:                      node.ID,
		NodeName:                node.NodeName,
		HostIP:                  node.HostIP,
		Role:                    node.Role,
		OSImage:                 node.OSImage,
		KernelVersion:           node.KernelVersion,
		KubeletVersion:          node.KubeletVersion,
		ContainerRuntimeVersion: node.ContainerRuntimeVersion,
		KubeProxyVersion:        node.KubeProxyVersion,
		CPULogic:                node.CPULogic, // 保持字符串类型
		MemLogic:                node.MemLogic, // 保持字符串类型
		CPUCapacity:             node.CPUCapacity,
		MemCapacity:             node.MemCapacity,
		CPUAllocatable:          node.CPUAllocatable,
		MemAllocatable:          node.MemAllocatable,
		FSTypeRoot:              node.FSTypeRoot,
		DiskRoot:                node.DiskRoot,
		DiskDocker:              node.DiskDocker,
		CreatedAt:               node.CreatedAt.String(),
		UpdatedAt:               node.UpdatedAt.String(),
	}
}
