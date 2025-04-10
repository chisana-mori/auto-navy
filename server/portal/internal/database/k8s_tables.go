package database

import (
	"log"

	"gorm.io/gorm"
)

// CreateK8sTables 创建K8s相关表
func CreateK8sTables(db *gorm.DB) error {
	// 检查表是否存在
	var count int64

	// 检查k8s_cluster表是否存在
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='k8s_cluster'").Scan(&count)
	if count == 0 {
		log.Println("创建k8s_cluster表...")
		createK8sClusterSQL := `
		CREATE TABLE k8s_cluster (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at datetime,
			updated_at datetime,
			name varchar(255) NOT NULL,
			region varchar(50),
			endpoint varchar(255),
			status varchar(50),
			deleted varchar(255)
		)`
		if err := db.Exec(createK8sClusterSQL).Error; err != nil {
			return err
		}
	}

	// 检查k8s_node表是否存在
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='k8s_node'").Scan(&count)
	if count == 0 {
		log.Println("创建k8s_node表...")
		createK8sNodeSQL := `
		CREATE TABLE k8s_node (
			id BIGINT PRIMARY KEY AUTOINCREMENT NOT NULL,
			created_at datetime,
			updated_at datetime,
			nodename varchar(191) NOT NULL,
			hostip varchar(191),
			role varchar(191),
			osimage varchar(128),
			kernelversion varchar(64),
			kubeletversion varchar(64),
			kubeproxyversion varchar(64),
			containerruntimeversion varchar(64),
			architecture varchar(64),
			cpulogic varchar(191),
			memlogic varchar(191),
			cpucapacity varchar(191),
			memcapacity varchar(191),
			cpuallocatable varchar(191),
			memallocatable varchar(191),
			fstyperoot varchar(191),
			diskroot varchar(191),
			diskdocker varchar(191),
			diskkubelet varchar(191),
			nodecreated varchar(191),
			status varchar(191),
			k8s_cluster_id bigint unsigned,
			gpu varchar(64),
			disk_count int,
			disk_detail varchar(512),
			network_speed int,
			deleted varchar(255)
		)`
		if err := db.Exec(createK8sNodeSQL).Error; err != nil {
			return err
		}
	}

	// 检查k8s_node_label表是否存在
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='k8s_node_label'").Scan(&count)
	if count == 0 {
		log.Println("创建k8s_node_label表...")
		createK8sNodeLabelSQL := `
		CREATE TABLE k8s_node_label (
			id BIGINT PRIMARY KEY AUTOINCREMENT NOT NULL,
			created_at datetime,
			updated_at datetime,
			key varchar(191),
			value varchar(191),
			status varchar(191),
			node_id bigint unsigned,
			deleted varchar(255)
		)`
		if err := db.Exec(createK8sNodeLabelSQL).Error; err != nil {
			return err
		}
	}

	// 检查k8s_node_taint表是否存在
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='k8s_node_taint'").Scan(&count)
	if count == 0 {
		log.Println("创建k8s_node_taint表...")
		createK8sNodeTaintSQL := `
		CREATE TABLE k8s_node_taint (
			id BIGINT PRIMARY KEY AUTOINCREMENT NOT NULL,
			created_at datetime,
			updated_at datetime,
			key varchar(191),
			value varchar(191),
			effect varchar(191),
			status varchar(191),
			node_id bigint unsigned,
			deleted varchar(255)
		)`
		if err := db.Exec(createK8sNodeTaintSQL).Error; err != nil {
			return err
		}
	}

	// 检查表结构
	var k8sClusterResult map[string]interface{}
	db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='k8s_cluster'").Scan(&k8sClusterResult)
	log.Printf("k8s_cluster表结构: %v", k8sClusterResult)

	var k8sNodeResult map[string]interface{}
	db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='k8s_node'").Scan(&k8sNodeResult)
	log.Printf("k8s_node表结构: %v", k8sNodeResult)

	var k8sNodeLabelResult map[string]interface{}
	db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='k8s_node_label'").Scan(&k8sNodeLabelResult)
	log.Printf("k8s_node_label表结构: %v", k8sNodeLabelResult)

	var k8sNodeTaintResult map[string]interface{}
	db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='k8s_node_taint'").Scan(&k8sNodeTaintResult)
	log.Printf("k8s_node_taint表结构: %v", k8sNodeTaintResult)

	return nil
}
