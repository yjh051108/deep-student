-- ============================================================================
-- V20260529: 合并重复的附件根文件夹
-- ============================================================================
--
-- 背景：
-- - 早期版本可能会重复创建多个同名“附件”根文件夹
-- - 配置表 `memory_config.attachment_root_folder_id` 只应该指向一个活动根目录
-- - 其余同名根目录应合并到统一根目录后软删除
--
-- 处理策略：
-- 1. 优先采用已配置且仍然活动的根目录作为 canonical
-- 2. 否则选取最早创建的活动“附件”根目录
-- 3. 将重复根目录中的活跃 folder_items 迁移到 canonical
-- 4. 软删除重复根目录及其残留 folder_items
-- ============================================================================

-- 选择 canonical 根目录并回写配置。没有旧配置时先补一条配置。
INSERT INTO memory_config (key, value, updated_at)
SELECT 'attachment_root_folder_id', canonical.id, datetime('now')
FROM (
    SELECT id
    FROM (
        SELECT id, 0 AS priority, created_at
        FROM folders
        WHERE id = (
            SELECT value
            FROM memory_config
            WHERE key = 'attachment_root_folder_id'
            LIMIT 1
        )
          AND deleted_at IS NULL
        UNION ALL
        SELECT id, 1 AS priority, created_at
        FROM folders
        WHERE parent_id IS NULL
          AND title = '附件'
          AND deleted_at IS NULL
    )
    ORDER BY priority ASC, created_at ASC
    LIMIT 1
) AS canonical
WHERE NOT EXISTS (
    SELECT 1
    FROM memory_config
    WHERE key = 'attachment_root_folder_id'
)
  AND canonical.id IS NOT NULL;

UPDATE memory_config
SET value = (
    SELECT id
    FROM (
        SELECT id, 0 AS priority, created_at
        FROM folders
        WHERE id = (
            SELECT value
            FROM memory_config
            WHERE key = 'attachment_root_folder_id'
            LIMIT 1
        )
          AND deleted_at IS NULL
        UNION ALL
        SELECT id, 1 AS priority, created_at
        FROM folders
        WHERE parent_id IS NULL
          AND title = '附件'
          AND deleted_at IS NULL
    )
    ORDER BY priority ASC, created_at ASC
    LIMIT 1
)
WHERE key = 'attachment_root_folder_id'
  AND EXISTS (
      SELECT 1
      FROM folders
      WHERE parent_id IS NULL
        AND title = '附件'
        AND deleted_at IS NULL
  );

-- 将重复根目录中未删除的条目迁移到 canonical
UPDATE folder_items
SET folder_id = (
        SELECT value
        FROM memory_config
        WHERE key = 'attachment_root_folder_id'
        LIMIT 1
    ),
    updated_at = CAST(strftime('%s', 'now') AS INTEGER) * 1000
WHERE deleted_at IS NULL
  AND folder_id IN (
      SELECT id
      FROM folders
      WHERE parent_id IS NULL
        AND title = '附件'
        AND deleted_at IS NULL
        AND id <> (
            SELECT value
            FROM memory_config
            WHERE key = 'attachment_root_folder_id'
            LIMIT 1
        )
  )
  AND NOT EXISTS (
      SELECT 1
      FROM folder_items AS canonical_item
      WHERE canonical_item.folder_id = (
              SELECT value
              FROM memory_config
              WHERE key = 'attachment_root_folder_id'
              LIMIT 1
          )
        AND canonical_item.item_type = folder_items.item_type
        AND canonical_item.item_id = folder_items.item_id
        AND canonical_item.deleted_at IS NULL
  );

-- 软删除重复根目录中的残留条目
UPDATE folder_items
SET deleted_at = datetime('now'),
    updated_at = CAST(strftime('%s', 'now') AS INTEGER) * 1000
WHERE deleted_at IS NULL
  AND folder_id IN (
      SELECT id
      FROM folders
      WHERE parent_id IS NULL
        AND title = '附件'
        AND deleted_at IS NULL
        AND id <> (
            SELECT value
            FROM memory_config
            WHERE key = 'attachment_root_folder_id'
            LIMIT 1
        )
  );

-- 软删除重复根目录本身
UPDATE folders
SET deleted_at = datetime('now'),
    updated_at = CAST(strftime('%s', 'now') AS INTEGER) * 1000
WHERE parent_id IS NULL
  AND title = '附件'
  AND deleted_at IS NULL
  AND id <> (
      SELECT value
      FROM memory_config
      WHERE key = 'attachment_root_folder_id'
      LIMIT 1
  );
