//! VFS reference-mode helpers retained for remaining legacy Rust paths.
//!
//! The Tauri command wrappers for resource refs have moved to the Go/Wails
//! `VfsService`. This module keeps text extraction and path helper code that
//! still has Rust callers during the backend retirement.

use std::sync::Arc;

use rusqlite::{params, Connection, OptionalExtension};
use tracing::{debug, info, warn};

use crate::document_parser::DocumentParser;
use crate::vfs::canonical_folder_item_type;
use crate::vfs::database::VfsDatabase;
use crate::vfs::error::VfsResult;
use crate::vfs::indexing::VfsChunker;
use crate::vfs::ocr_utils::{join_ocr_pages_text, parse_ocr_pages_json};
use crate::vfs::types::VfsResourceType;

fn is_usable_ocr_text(text: &str) -> bool {
    !text.trim().is_empty() && VfsChunker::is_text_quality_acceptable(text)
}

fn filter_usable_ocr_pages(pages: Vec<Option<String>>) -> Vec<Option<String>> {
    pages
        .into_iter()
        .map(|page| page.filter(|text| is_usable_ocr_text(text)))
        .collect()
}

fn has_enough_usable_ocr_pages(pages: &[Option<String>]) -> bool {
    if pages.is_empty() {
        return false;
    }
    pages.iter().filter(|page| page.is_some()).count() * 2 >= pages.len()
}

/// 获取资源的完整路径
///
/// 使用递归 CTE 查询构建路径
fn get_resource_path_with_conn(
    conn: &Connection,
    source_id: &str,
    resource_type: &VfsResourceType,
) -> VfsResult<String> {
    // ★ FIX: 使用 source_id 作为路径末段而非标题
    // 之前使用标题（如 "有机合成完整笔记"）会导致前端 dstu.get(path) 时
    // extract_resource_info 无法从中提取 resource ID，报错：
    // "Invalid DSTU path: Path must contain a resource ID: {title}"
    // node.name 已包含人类可读标题用于显示，path 应包含可解析的 resource ID

    let resource_type_string = resource_type.to_string();
    let canonical_item_type = canonical_folder_item_type(&resource_type_string);

    // 查找资源所在的文件夹
    let folder_id: Option<String> = conn
        .query_row(
            "SELECT folder_id FROM folder_items WHERE item_id = ?1 AND item_type = ?2 AND deleted_at IS NULL",
            params![source_id, canonical_item_type],
            |row| row.get(0),
        )
        .optional()?
        .flatten();

    match folder_id {
        Some(fid) => {
            // 构建文件夹路径
            let folder_path = build_folder_path_with_conn(conn, &fid)?;
            Ok(format!("{}/{}", folder_path, source_id))
        }
        None => {
            // 资源在根级
            Ok(source_id.to_string())
        }
    }
}

/// 构建文件夹路径（向上追溯到根）
fn build_folder_path_with_conn(conn: &Connection, folder_id: &str) -> VfsResult<String> {
    // 使用 CTE 向上追溯到根
    let mut stmt = conn.prepare(
        r#"
        WITH RECURSIVE folder_path AS (
            SELECT id, parent_id, title, 1 as depth
            FROM folders WHERE id = ?1
            UNION ALL
            SELECT f.id, f.parent_id, f.title, fp.depth + 1
            FROM folders f JOIN folder_path fp ON f.id = fp.parent_id
            WHERE fp.depth < 11
        )
        SELECT title FROM folder_path ORDER BY depth DESC
        "#,
    )?;

    let titles: Vec<String> = stmt
        .query_map(params![folder_id], |row| row.get(0))?
        .collect::<Result<Vec<String>, _>>()?;

    Ok(titles.join("/"))
}

/// 获取图片的 OCR 文本
///
/// 从 files 表关联的 resources.ocr_text 获取图片的 OCR 文本
///
/// ## 参数
/// - `conn`: 数据库连接
/// - `source_id`: 图片附件 ID (file_xxx, att_xxx)
///
/// ## 返回
/// - `Some(String)`: OCR 文本
/// - `None`: 没有 OCR 文本
pub fn get_image_ocr_text_with_conn(conn: &Connection, source_id: &str) -> Option<String> {
    info!(
        "[OCR_DIAG] get_image_ocr_text_with_conn: querying OCR text for source_id={}",
        source_id
    );

    // ★ 诊断：先检查 files 表中是否存在该 source_id
    let check_files_sql = r#"
        SELECT a.id, a.resource_id, a.file_name
        FROM files a
        WHERE (a.id = ?1 OR a.resource_id = ?1)
          AND a.status = 'active' AND a.deleted_at IS NULL
        ORDER BY CASE WHEN a.id = ?1 THEN 0 ELSE 1 END
        LIMIT 1
    "#;
    match conn.query_row(check_files_sql, params![source_id], |row| {
        Ok((
            row.get::<_, String>(0)?,
            row.get::<_, Option<String>>(1)?,
            row.get::<_, Option<String>>(2)?,
        ))
    }) {
        Ok((file_id, resource_id, file_name)) => {
            info!(
                "[OCR_DIAG] files table match: source_id={} -> file_id={}, resource_id={:?}, file_name={:?}",
                source_id, file_id, resource_id, file_name
            );
        }
        Err(e) => {
            warn!(
                "[OCR_DIAG] files table NO MATCH for source_id={}: {}. This means the SQL JOIN will fail and no OCR text can be retrieved.",
                source_id, e
            );
        }
    }

    // ★ 诊断：检查关联的 resource 是否有 ocr_text
    let check_ocr_sql = r#"
        SELECT r.id, r.ocr_text IS NOT NULL AS has_ocr, LENGTH(r.ocr_text) AS ocr_len
        FROM files a
        JOIN resources r ON a.resource_id = r.id
        WHERE (a.id = ?1 OR a.resource_id = ?1)
          AND a.status = 'active' AND a.deleted_at IS NULL AND r.deleted_at IS NULL
        ORDER BY CASE WHEN a.id = ?1 THEN 0 ELSE 1 END
        LIMIT 1
    "#;
    match conn.query_row(check_ocr_sql, params![source_id], |row| {
        Ok((
            row.get::<_, String>(0)?,
            row.get::<_, bool>(1)?,
            row.get::<_, Option<i64>>(2)?,
        ))
    }) {
        Ok((resource_id, has_ocr, ocr_len)) => {
            info!(
                "[OCR_DIAG] resource OCR status: source_id={} -> resource_id={}, has_ocr={}, ocr_len={:?}",
                source_id, resource_id, has_ocr, ocr_len
            );
        }
        Err(e) => {
            warn!(
                "[OCR_DIAG] resource OCR check failed for source_id={}: {}",
                source_id, e
            );
        }
    }

    // 尝试从 files 表关联的 resource 获取 OCR 文本
    let sql = r#"
        SELECT r.ocr_text
        FROM files a
        JOIN resources r ON a.resource_id = r.id
        WHERE (a.id = ?1 OR a.resource_id = ?1)
          AND a.status = 'active' AND a.deleted_at IS NULL AND r.deleted_at IS NULL
        ORDER BY CASE WHEN a.id = ?1 THEN 0 ELSE 1 END
        LIMIT 1
    "#;

    match conn.query_row(sql, params![source_id], |row| {
        row.get::<_, Option<String>>(0)
    }) {
        Ok(Some(text)) if is_usable_ocr_text(&text) => {
            info!(
                "[OCR_DIAG] OCR text FOUND for source_id={}, len={}, preview=\"{}\"",
                source_id,
                text.len(),
                text.chars().take(100).collect::<String>()
            );
            Some(text)
        }
        Ok(Some(_)) => {
            warn!(
                "[OCR_DIAG] OCR text exists but is empty or below the quality gate for source_id={}",
                source_id
            );
            None
        }
        Ok(None) => {
            warn!(
                "[OCR_DIAG] OCR text is NULL in database for source_id={}. Possible causes: (1) OCR pipeline not yet completed, (2) OCR failed silently, (3) image was not processed",
                source_id
            );
            None
        }
        Err(e) => {
            warn!(
                "[OCR_DIAG] OCR text query FAILED for source_id={}: {}. Possible cause: source_id not found in files table (JOIN returned no rows)",
                source_id, e
            );
            None
        }
    }
}

/// 获取文件的 extracted_text 字段
///
/// 公开供其他模块调用（统一文本抽取策略）
pub fn get_extracted_text_with_conn(conn: &Connection, source_id: &str) -> Option<String> {
    let sql = r#"
        SELECT extracted_text
        FROM files
        WHERE (id = ?1 OR resource_id = ?1)
          AND status = 'active' AND deleted_at IS NULL
        ORDER BY CASE WHEN id = ?1 THEN 0 ELSE 1 END
        LIMIT 1
    "#;
    conn.query_row(sql, params![source_id], |row| row.get(0))
        .ok()
        .flatten()
        .filter(|t: &String| !t.trim().is_empty())
}

/// 从 ocr_pages_json 获取 PDF 的页级 OCR 文本
///
/// 将所有非空页的 OCR 文本拼接返回
///
/// 公开供其他模块调用（统一文本抽取策略）
pub fn get_ocr_pages_text_with_conn(conn: &Connection, source_id: &str) -> Option<String> {
    let sql = r#"
        SELECT ocr_pages_json
        FROM files
        WHERE (id = ?1 OR resource_id = ?1)
          AND status = 'active' AND deleted_at IS NULL
        ORDER BY CASE WHEN id = ?1 THEN 0 ELSE 1 END
        LIMIT 1
    "#;

    let ocr_json: Option<String> = conn
        .query_row(sql, params![source_id], |row| row.get(0))
        .ok()
        .flatten();

    let ocr_json = ocr_json?;
    if ocr_json.trim().is_empty() {
        return None;
    }

    let pages = filter_usable_ocr_pages(parse_ocr_pages_json(&ocr_json));
    if pages.is_empty() {
        return None;
    }
    if !has_enough_usable_ocr_pages(&pages) {
        return None;
    }

    join_ocr_pages_text(&pages, "第", "页")
}

/// 根据 sourceId 前缀获取资源类型信息
///
/// 返回 (VfsResourceType, 表名, 标题列名)
///
/// ★ 注意：附件类型 (att_) 需要查询数据库才能确定是 Image 还是 File
/// ★ 2026-02-09 修复：essay_session_ 必须在 essay_ 之前检查，否则会被错误匹配到 essays 表
fn get_source_id_type(source_id: &str) -> Option<(VfsResourceType, &'static str, &'static str)> {
    if source_id.starts_with("note_") {
        Some((VfsResourceType::Note, "notes", "title"))
    } else if source_id.starts_with("mm_") {
        Some((VfsResourceType::MindMap, "mindmaps", "title"))
    } else if source_id.starts_with("tb_") {
        // ★ 2026-02-09 修复：tb_ 映射为 Textbook（与前端 inferTypeFromSourceId 一致）
        // 数据仍存储在 files 表（Migration032 后教材统一到 files 表）
        Some((VfsResourceType::Textbook, "files", "file_name"))
    } else if source_id.starts_with("file_") || source_id.starts_with("att_") {
        Some((VfsResourceType::File, "files", "file_name"))
    } else if source_id.starts_with("exam_") {
        Some((
            VfsResourceType::Exam,
            "exam_sheets",
            "COALESCE(exam_name, id)",
        ))
    } else if source_id.starts_with("tr_") {
        Some((
            VfsResourceType::Translation,
            "translations",
            "COALESCE(title, id)",
        ))
    } else if source_id.starts_with("essay_session_") {
        // ★ 2026-02-09: essay_session_ 必须在 essay_ 之前检查！
        // essay_sessions 表没有 resource_id 列，需要在调用方做特殊处理
        Some((
            VfsResourceType::Essay,
            "essay_sessions",
            "COALESCE(title, id)",
        ))
    } else if source_id.starts_with("essay_") {
        Some((VfsResourceType::Essay, "essays", "COALESCE(title, id)"))
    } else {
        None
    }
}
// ============================================================================
// 统一文本抽取策略（公共函数）
// ============================================================================

/// ★ 统一文本抽取策略（T02 修复）
///
/// 对 File/Textbook 类型资源，按以下优先级获取文本：
/// 1. 获取 OCR 结果 (`ocr_pages_json`)
/// 2. 获取直接解析结果 (`extracted_text`)
/// 3. 如果两者都不足，尝试 `DocumentParser` 实时解析
/// 4. 取"更长者"作为最终结果
///
/// ## 参数
/// - `conn`: 数据库连接
/// - `source_id`: 资源 ID（file_xxx, tb_xxx, att_xxx）
/// - `file_name`: 文件名（用于 DocumentParser 确定类型）
/// - `base64_content`: 可选的 base64 编码文件内容（用于 DocumentParser 回退）
///
/// ## 返回
/// - `Some(String)`: 提取的文本内容
/// - `None`: 无法提取文本（返回前应生成占位提示）
pub fn extract_file_text_with_strategy(
    conn: &Connection,
    source_id: &str,
    file_name: &str,
    base64_content: Option<&str>,
) -> Option<String> {
    const TEXT_THRESHOLD: usize = 1000;

    // 1. 获取 OCR 结果 (ocr_pages_json)
    let ocr_text = get_ocr_pages_text_with_conn(conn, source_id);
    let ocr_len = ocr_text.as_ref().map(|t| t.len()).unwrap_or(0);
    debug!(
        "[TextExtract] OCR text for source_id={}: len={}",
        source_id, ocr_len
    );

    // 2. 获取直接解析结果 (extracted_text)
    let extracted_text = get_extracted_text_with_conn(conn, source_id);
    let mut parsed_text = extracted_text.clone();

    // 3. 如果没有 extracted_text 或内容过短，尝试 DocumentParser 实时解析
    if parsed_text.is_none() || parsed_text.as_ref().map(|t| t.len()).unwrap_or(0) < TEXT_THRESHOLD
    {
        if let Some(base64) = base64_content {
            if !base64.is_empty() {
                debug!(
                    "[TextExtract] Trying DocumentParser for source_id={}, base64_len={}",
                    source_id,
                    base64.len()
                );
                let parser = DocumentParser::new();
                if let Ok(text) = parser.extract_text_from_base64(file_name, base64) {
                    if text.len() > parsed_text.as_ref().map(|t| t.len()).unwrap_or(0) {
                        debug!("[TextExtract] DocumentParser result: {} chars", text.len());
                        parsed_text = Some(text);
                    }
                }
            }
        }
    }
    let parsed_len = parsed_text.as_ref().map(|t| t.len()).unwrap_or(0);
    debug!(
        "[TextExtract] Parsed text for source_id={}: len={}",
        source_id, parsed_len
    );

    // 4. 取大者（OCR 和 解析结果比较）
    if ocr_len > parsed_len {
        debug!(
            "[TextExtract] Using OCR text (larger): {} > {}",
            ocr_len, parsed_len
        );
        ocr_text
    } else if parsed_len > 0 {
        debug!(
            "[TextExtract] Using parsed text: {} >= {}",
            parsed_len, ocr_len
        );
        parsed_text
    } else if ocr_len > 0 {
        debug!("[TextExtract] Using OCR text (only available): {}", ocr_len);
        ocr_text
    } else {
        debug!(
            "[TextExtract] No text available for source_id={}",
            source_id
        );
        None
    }
}

// ============================================================================
// 公共辅助函数（供其他模块调用）
// ============================================================================

/// ★ 获取资源的文件夹路径（供 DSTU 层调用）
///
/// 根据 sourceId 获取资源的完整文件夹层级路径。
///
/// ## 参数
/// - `vfs_db`: VFS 数据库实例
/// - `source_id`: 资源 ID（note_xxx, tb_xxx 等）
///
/// ## 返回
/// - `Ok(String)`: 完整路径，如 "高考复习/函数/note_abc123"（末段是 source_id，非标题）
/// - `Err(VfsError)`: 错误信息
pub fn get_resource_path_internal(vfs_db: &Arc<VfsDatabase>, source_id: &str) -> VfsResult<String> {
    let conn = vfs_db.get_conn_safe()?;

    // 获取资源类型
    let resource_type = match get_source_id_type(source_id) {
        Some((rt, _, _)) => rt,
        None => return Ok(source_id.to_string()), // 未知类型，返回 sourceId 作为路径
    };

    get_resource_path_with_conn(&conn, source_id, &resource_type)
}

// ============================================================================
// 单元测试
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    // ------------------------------------------------------------------------
    // 测试辅助函数：创建内存数据库
    // ------------------------------------------------------------------------

    /// 创建内存数据库并初始化 files 表
    fn create_test_db() -> Connection {
        let conn = Connection::open_in_memory().expect("Failed to create in-memory database");

        // 创建 files 表，包含 OCR 和 extracted_text 字段
        conn.execute(
            r#"
            CREATE TABLE files (
                id TEXT PRIMARY KEY,
                file_name TEXT NOT NULL,
                extracted_text TEXT,
                ocr_pages_json TEXT,
                resource_id TEXT,
                status TEXT NOT NULL DEFAULT 'active',
                deleted_at TEXT
            )
            "#,
            [],
        )
        .expect("Failed to create files table");

        conn.execute(
            r#"
            CREATE TABLE resources (
                id TEXT PRIMARY KEY,
                ocr_text TEXT,
                deleted_at TEXT
            )
            "#,
            [],
        )
        .expect("Failed to create resources table");

        conn
    }

    /// 插入测试文件记录
    fn insert_test_file(
        conn: &Connection,
        id: &str,
        file_name: &str,
        extracted_text: Option<&str>,
        ocr_pages_json: Option<&str>,
    ) {
        conn.execute(
            r#"
            INSERT INTO files (id, file_name, extracted_text, ocr_pages_json)
            VALUES (?1, ?2, ?3, ?4)
            "#,
            params![id, file_name, extracted_text, ocr_pages_json],
        )
        .expect("Failed to insert test file");
    }

    fn soft_delete_test_file(conn: &Connection, id: &str) {
        conn.execute(
            "UPDATE files SET status = 'deleted', deleted_at = ?1 WHERE id = ?2",
            params!["2026-05-30T00:00:00.000Z", id],
        )
        .expect("Failed to soft delete test file");
    }

    fn insert_test_resource(conn: &Connection, id: &str, ocr_text: Option<&str>) {
        conn.execute(
            "INSERT INTO resources (id, ocr_text, deleted_at) VALUES (?1, ?2, NULL)",
            params![id, ocr_text],
        )
        .expect("Failed to insert test resource");
    }

    fn soft_delete_test_resource(conn: &Connection, id: &str) {
        conn.execute(
            "UPDATE resources SET deleted_at = ?1 WHERE id = ?2",
            params!["2026-05-30T00:00:00.000Z", id],
        )
        .expect("Failed to soft delete test resource");
    }

    fn insert_test_file_with_resource_id(
        conn: &Connection,
        id: &str,
        resource_id: &str,
        file_name: &str,
        extracted_text: Option<&str>,
        ocr_pages_json: Option<&str>,
    ) {
        conn.execute(
            r#"
            INSERT INTO files (id, resource_id, file_name, extracted_text, ocr_pages_json)
            VALUES (?1, ?2, ?3, ?4, ?5)
            "#,
            params![id, resource_id, file_name, extracted_text, ocr_pages_json],
        )
        .expect("Failed to insert test file with resource_id");
    }

    // ------------------------------------------------------------------------
    // extract_file_text_with_strategy 测试
    // ------------------------------------------------------------------------

    #[test]
    fn test_extract_file_text_ocr_priority() {
        // 测试：当 OCR 文本比 extracted_text 更长时，应优先使用 OCR
        let conn = create_test_db();

        let ocr_json = r#"["这是第一页的OCR文本，内容很长很长很长很长很长很长很长很长很长很长很长很长", "这是第二页的OCR文本，同样很长很长很长很长很长很长很长很长很长很长很长很长"]"#;
        let extracted = "短文本";

        insert_test_file(
            &conn,
            "file_001",
            "test.pdf",
            Some(extracted),
            Some(ocr_json),
        );

        let result = extract_file_text_with_strategy(&conn, "file_001", "test.pdf", None);

        assert!(result.is_some());
        let text = result.unwrap();
        // OCR 文本应包含页码标记
        assert!(text.contains("--- 第 1 页 ---"));
        assert!(text.contains("--- 第 2 页 ---"));
        assert!(text.contains("这是第一页的OCR文本"));
    }

    #[test]
    fn test_extract_file_text_extracted_fallback() {
        // 测试：当没有 OCR 时，应使用 extracted_text
        let conn = create_test_db();

        let extracted =
            "这是从文档中提取的文本内容，比较长，足够使用。包含很多字符以确保测试有效。";

        insert_test_file(&conn, "file_002", "test.docx", Some(extracted), None);

        let result = extract_file_text_with_strategy(&conn, "file_002", "test.docx", None);

        assert!(result.is_some());
        assert_eq!(result.unwrap(), extracted);
    }

    #[test]
    fn test_extract_file_text_empty_ocr() {
        // 测试：当 OCR 为空数组时，应回退到 extracted_text
        let conn = create_test_db();

        let ocr_json = r#"[null, "", null]"#;
        let extracted = "提取的文本内容";

        insert_test_file(
            &conn,
            "file_003",
            "test.pdf",
            Some(extracted),
            Some(ocr_json),
        );

        let result = extract_file_text_with_strategy(&conn, "file_003", "test.pdf", None);

        assert!(result.is_some());
        assert_eq!(result.unwrap(), extracted);
    }

    #[test]
    fn test_extract_file_text_no_content() {
        // 测试：当既没有 OCR 也没有 extracted_text 时，返回 None
        let conn = create_test_db();

        insert_test_file(&conn, "file_004", "test.pdf", None, None);

        let result = extract_file_text_with_strategy(&conn, "file_004", "test.pdf", None);

        assert!(result.is_none());
    }

    #[test]
    fn test_extract_file_text_nonexistent_file() {
        // 测试：当文件不存在时，返回 None
        let conn = create_test_db();

        let result = extract_file_text_with_strategy(&conn, "file_nonexistent", "test.pdf", None);

        assert!(result.is_none());
    }

    #[test]
    fn test_extract_file_text_by_resource_id() {
        // 测试：当传入的是 resource_id 时，也应能读取 OCR / extracted_text
        let conn = create_test_db();
        let ocr_json = r#"["resource id OCR text"]"#;

        insert_test_file_with_resource_id(
            &conn,
            "file_010",
            "res_010",
            "test.pdf",
            Some("resource id extracted text"),
            Some(ocr_json),
        );

        let by_resource = extract_file_text_with_strategy(&conn, "res_010", "test.pdf", None);
        assert!(by_resource.is_some());
        let text = by_resource.unwrap();
        assert!(text.contains("resource id"));
    }

    #[test]
    fn test_extract_file_text_longer_wins() {
        // 测试：取"更长者"策略 - extracted_text 更长时使用它
        let conn = create_test_db();

        let ocr_json = r#"["短OCR"]"#;
        let extracted = "这是一段非常非常非常非常非常非常非常非常非常非常非常非常非常长的提取文本，远远超过OCR的长度";

        insert_test_file(
            &conn,
            "file_005",
            "test.pdf",
            Some(extracted),
            Some(ocr_json),
        );

        let result = extract_file_text_with_strategy(&conn, "file_005", "test.pdf", None);

        assert!(result.is_some());
        let text = result.unwrap();
        // 应该使用更长的 extracted_text
        assert!(text.contains("非常非常非常"));
        assert!(!text.contains("--- 第")); // 不应包含 OCR 页码标记
    }

    #[test]
    fn test_extract_file_text_whitespace_only_extracted() {
        // 测试：当 extracted_text 只包含空白时，应被忽略
        let conn = create_test_db();

        let ocr_json = r#"["有效的OCR文本内容"]"#;
        let extracted = "   \n\t  ";

        insert_test_file(
            &conn,
            "file_006",
            "test.pdf",
            Some(extracted),
            Some(ocr_json),
        );

        let result = extract_file_text_with_strategy(&conn, "file_006", "test.pdf", None);

        assert!(result.is_some());
        let text = result.unwrap();
        assert!(text.contains("有效的OCR文本内容"));
    }

    // ------------------------------------------------------------------------
    // get_ocr_pages_text_with_conn 测试
    // ------------------------------------------------------------------------

    #[test]
    fn test_get_ocr_pages_text_valid() {
        let conn = create_test_db();

        let ocr_json = r#"["第一页内容", "第二页内容", "第三页内容"]"#;
        insert_test_file(&conn, "file_ocr_1", "test.pdf", None, Some(ocr_json));

        let result = get_ocr_pages_text_with_conn(&conn, "file_ocr_1");

        assert!(result.is_some());
        let text = result.unwrap();
        assert!(text.contains("--- 第 1 页 ---"));
        assert!(text.contains("第一页内容"));
        assert!(text.contains("--- 第 2 页 ---"));
        assert!(text.contains("第二页内容"));
        assert!(text.contains("--- 第 3 页 ---"));
        assert!(text.contains("第三页内容"));
    }

    #[test]
    fn test_get_ocr_pages_text_with_nulls() {
        let conn = create_test_db();

        // 某些页面为 null
        let ocr_json = r#"["第一页", null, "第三页", null]"#;
        insert_test_file(&conn, "file_ocr_2", "test.pdf", None, Some(ocr_json));

        let result = get_ocr_pages_text_with_conn(&conn, "file_ocr_2");

        assert!(result.is_some());
        let text = result.unwrap();
        assert!(text.contains("--- 第 1 页 ---"));
        assert!(text.contains("第一页"));
        assert!(text.contains("--- 第 3 页 ---"));
        assert!(text.contains("第三页"));
        // 不应包含空页
        assert!(!text.contains("--- 第 2 页 ---"));
        assert!(!text.contains("--- 第 4 页 ---"));
    }

    #[test]
    fn test_get_ocr_pages_text_empty_array() {
        let conn = create_test_db();

        let ocr_json = r#"[]"#;
        insert_test_file(&conn, "file_ocr_3", "test.pdf", None, Some(ocr_json));

        let result = get_ocr_pages_text_with_conn(&conn, "file_ocr_3");

        assert!(result.is_none());
    }

    #[test]
    fn test_get_ocr_pages_text_all_empty() {
        let conn = create_test_db();

        let ocr_json = r#"["", "  ", null]"#;
        insert_test_file(&conn, "file_ocr_4", "test.pdf", None, Some(ocr_json));

        let result = get_ocr_pages_text_with_conn(&conn, "file_ocr_4");

        assert!(result.is_none());
    }

    #[test]
    fn test_get_ocr_pages_text_invalid_json() {
        let conn = create_test_db();

        let ocr_json = "not a valid json";
        insert_test_file(&conn, "file_ocr_5", "test.pdf", None, Some(ocr_json));

        let result = get_ocr_pages_text_with_conn(&conn, "file_ocr_5");

        assert!(result.is_none());
    }

    #[test]
    fn test_get_ocr_pages_text_hides_deleted_file() {
        let conn = create_test_db();
        let ocr_json = r#"["deleted OCR text should be hidden"]"#;
        insert_test_file(&conn, "file_ocr_deleted", "test.pdf", None, Some(ocr_json));
        soft_delete_test_file(&conn, "file_ocr_deleted");

        let result = get_ocr_pages_text_with_conn(&conn, "file_ocr_deleted");

        assert!(result.is_none());
    }

    // ------------------------------------------------------------------------
    // get_extracted_text_with_conn 测试
    // ------------------------------------------------------------------------

    #[test]
    fn test_get_extracted_text_valid() {
        let conn = create_test_db();

        insert_test_file(
            &conn,
            "file_ext_1",
            "test.pdf",
            Some("提取的文本内容"),
            None,
        );

        let result = get_extracted_text_with_conn(&conn, "file_ext_1");

        assert!(result.is_some());
        assert_eq!(result.unwrap(), "提取的文本内容");
    }

    #[test]
    fn test_get_extracted_text_empty() {
        let conn = create_test_db();

        insert_test_file(&conn, "file_ext_2", "test.pdf", Some(""), None);

        let result = get_extracted_text_with_conn(&conn, "file_ext_2");

        assert!(result.is_none());
    }

    #[test]
    fn test_get_extracted_text_whitespace_only() {
        let conn = create_test_db();

        insert_test_file(&conn, "file_ext_3", "test.pdf", Some("   \n\t  "), None);

        let result = get_extracted_text_with_conn(&conn, "file_ext_3");

        assert!(result.is_none());
    }

    #[test]
    fn test_get_extracted_text_none() {
        let conn = create_test_db();

        insert_test_file(&conn, "file_ext_4", "test.pdf", None, None);

        let result = get_extracted_text_with_conn(&conn, "file_ext_4");

        assert!(result.is_none());
    }

    #[test]
    fn test_get_extracted_text_hides_deleted_file() {
        let conn = create_test_db();
        insert_test_file(
            &conn,
            "file_ext_deleted",
            "test.pdf",
            Some("deleted extracted text should be hidden"),
            None,
        );
        soft_delete_test_file(&conn, "file_ext_deleted");

        let result = get_extracted_text_with_conn(&conn, "file_ext_deleted");

        assert!(result.is_none());
    }

    #[test]
    fn test_get_image_ocr_text_requires_active_file_and_resource() {
        let conn = create_test_db();
        insert_test_resource(&conn, "res_img_active", Some("active image OCR text"));
        insert_test_file_with_resource_id(
            &conn,
            "file_img_active",
            "res_img_active",
            "active.png",
            None,
            None,
        );

        assert_eq!(
            get_image_ocr_text_with_conn(&conn, "file_img_active").as_deref(),
            Some("active image OCR text")
        );

        soft_delete_test_file(&conn, "file_img_active");
        assert!(get_image_ocr_text_with_conn(&conn, "file_img_active").is_none());

        insert_test_resource(&conn, "res_img_deleted", Some("deleted resource OCR text"));
        insert_test_file_with_resource_id(
            &conn,
            "file_img_resource_deleted",
            "res_img_deleted",
            "resource-deleted.png",
            None,
            None,
        );
        soft_delete_test_resource(&conn, "res_img_deleted");

        assert!(get_image_ocr_text_with_conn(&conn, "file_img_resource_deleted").is_none());
    }

    // ------------------------------------------------------------------------
    // get_source_id_type 测试
    // ------------------------------------------------------------------------

    #[test]
    fn test_get_source_id_type() {
        // 笔记
        let (t, table, col) = get_source_id_type("note_abc123").unwrap();
        assert_eq!(t, VfsResourceType::Note);
        assert_eq!(table, "notes");
        assert_eq!(col, "title");

        // 教材（★ 2026-02-09：tb_ 映射为 Textbook，与前端一致；数据仍在 files 表）
        let (t, table, col) = get_source_id_type("tb_xyz789").unwrap();
        assert_eq!(t, VfsResourceType::Textbook);
        assert_eq!(table, "files");
        assert_eq!(col, "file_name");

        // 题目集
        let (t, table, _) = get_source_id_type("exam_def456").unwrap();
        assert_eq!(t, VfsResourceType::Exam);
        assert_eq!(table, "exam_sheets");

        // 翻译（★ 2026-02-09 改进：title_column 改为 COALESCE(title, id)）
        let (t, table, col) = get_source_id_type("tr_ghi789").unwrap();
        assert_eq!(t, VfsResourceType::Translation);
        assert_eq!(table, "translations");
        assert_eq!(col, "COALESCE(title, id)");

        // ★ 2026-02-09 修复：作文会话必须映射到 essay_sessions 表
        let (t, table, col) = get_source_id_type("essay_session_abc123").unwrap();
        assert_eq!(t, VfsResourceType::Essay);
        assert_eq!(table, "essay_sessions");
        assert_eq!(col, "COALESCE(title, id)");

        // 作文轮次仍然映射到 essays 表
        let (t, table, _) = get_source_id_type("essay_jkl012").unwrap();
        assert_eq!(t, VfsResourceType::Essay);
        assert_eq!(table, "essays");

        // 附件（默认返回 File，实际类型需通过 get_attachment_type_with_conn 查询）
        let (t, table, col) = get_source_id_type("att_mno345").unwrap();
        assert_eq!(t, VfsResourceType::File);
        assert_eq!(table, "files");
        assert_eq!(col, "file_name");

        // 未知
        assert!(get_source_id_type("unknown_abc").is_none());
        assert!(get_source_id_type("abc123").is_none());
    }

    /// ★ 2026-02-09：essay_session_ 前缀不能被 essay_ 吞掉
    #[test]
    fn test_get_source_id_type_essay_session_priority() {
        // essay_session_ 必须匹配到 essay_sessions 表
        let (t, table, _) = get_source_id_type("essay_session_e8ZwCj4Og_").unwrap();
        assert_eq!(t, VfsResourceType::Essay);
        assert_eq!(table, "essay_sessions");

        // 普通 essay_ 仍然匹配到 essays 表
        let (t, table, _) = get_source_id_type("essay_abc123").unwrap();
        assert_eq!(t, VfsResourceType::Essay);
        assert_eq!(table, "essays");

        // essay_session_ 后面带各种字符
        let (t, table, _) = get_source_id_type("essay_session_XyZ_123").unwrap();
        assert_eq!(t, VfsResourceType::Essay);
        assert_eq!(table, "essay_sessions");
    }
}
