package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upUpdateResourcePoolDeviceMatchingPolicy, downUpdateResourcePoolDeviceMatchingPolicy)
}

// 更新资源池设备匹配策略表，将查询条件改为关联查询模板
func upUpdateResourcePoolDeviceMatchingPolicy(tx *sql.Tx) error {
	// 1. 添加新的 query_template_id 列
	_, err := tx.Exec(`
		ALTER TABLE resource_pool_device_matching_policy 
		ADD COLUMN query_template_id INT UNSIGNED NOT NULL DEFAULT 0 
		COMMENT '关联的查询模板ID' AFTER action_type;
	`)
	if err != nil {
		return err
	}

	// 2. 为每个策略创建对应的查询模板
	rows, err := tx.Query(`
		SELECT id, name, description, query_groups, created_by, updated_by 
		FROM resource_pool_device_matching_policy
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, description, queryGroups, createdBy, updatedBy string
		
		if err := rows.Scan(&id, &name, &description, &queryGroups, &createdBy, &updatedBy); err != nil {
			return err
		}
		
		// 为每个策略创建一个对应的查询模板
		templateName := name + " - 匹配模板"
		
		// 插入新的查询模板
		var templateID int
		err := tx.QueryRow(`
			INSERT INTO query_template (name, description, groups, created_by, updated_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())
			RETURNING id
		`, templateName, description, queryGroups, createdBy, updatedBy).Scan(&templateID)
		
		if err != nil {
			return err
		}
		
		// 更新策略关联到新创建的查询模板
		_, err = tx.Exec(`
			UPDATE resource_pool_device_matching_policy
			SET query_template_id = ?
			WHERE id = ?
		`, templateID, id)
		
		if err != nil {
			return err
		}
	}

	// 3. 删除旧的 query_groups 列
	_, err = tx.Exec(`
		ALTER TABLE resource_pool_device_matching_policy 
		DROP COLUMN query_groups;
	`)
	
	return err
}

// 回滚更改
func downUpdateResourcePoolDeviceMatchingPolicy(tx *sql.Tx) error {
	// 1. 添加回 query_groups 列
	_, err := tx.Exec(`
		ALTER TABLE resource_pool_device_matching_policy 
		ADD COLUMN query_groups TEXT NOT NULL 
		COMMENT '查询条件组，JSON格式' AFTER action_type;
	`)
	if err != nil {
		return err
	}

	// 2. 从关联的查询模板中恢复查询条件
	rows, err := tx.Query(`
		SELECT p.id, t.groups 
		FROM resource_pool_device_matching_policy p
		JOIN query_template t ON p.query_template_id = t.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var groups string
		
		if err := rows.Scan(&id, &groups); err != nil {
			return err
		}
		
		// 更新策略的查询条件
		_, err = tx.Exec(`
			UPDATE resource_pool_device_matching_policy
			SET query_groups = ?
			WHERE id = ?
		`, groups, id)
		
		if err != nil {
			return err
		}
	}

	// 3. 删除 query_template_id 列
	_, err = tx.Exec(`
		ALTER TABLE resource_pool_device_matching_policy 
		DROP COLUMN query_template_id;
	`)
	
	return err
}
