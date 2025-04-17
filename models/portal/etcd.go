package portal

type K8sETCD struct {
	BaseModel
	Instance                   string `gorm:"default:''"` // Note: Assuming 'default' was intended
	Role                       string `gorm:"column:job"`
	ServerId                   string `gorm:"default:''"` // Note: Assuming 'default' was intended
	HasLeader                  string `gorm:"default:''"` // Note: Assuming 'default' was intended
	ChangesLeader              string `gorm:"default:''"` // Note: Assuming 'default' was intended
	DbTotalSize                string `gorm:"default:''"` // Note: Assuming 'default' was intended
	ProcessMemory              string `gorm:"default:''"` // Note: Assuming 'default' was intended
	NetworkPeerReceivedSumRate string `gorm:"default:''"` // Note: Assuming 'default' was intended
	NetworkPeerSentSumRate     string `gorm:"default:''"` // Note: Assuming 'default' was intended
	NetworkGrpcReceivedRate    string `gorm:"default:''"` // Note: Assuming 'default' was intended
	NetworkGrpcSentRate        string `gorm:"default:''"` // Note: Assuming 'default' was intended
	ProposalsFailedRate        string `gorm:"default:''"` // Note: Assuming 'default' was intended
	ProposalsPendingRate       string `gorm:"column:proposals_pending_rate"`
	ProposalsCommittedRate     string `gorm:"default:''"` // Note: Assuming 'default' was intended
	ProposalsAppliedRate       string `gorm:"default:''"` // Note: Assuming 'default' was intended
	ClusterID                  int    `gorm:"column:k8s_cluster_id"`
}

func (K8sETCD) TableName() string {
	return "k8s_etcd"
}
