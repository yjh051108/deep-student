//! DSTU 访达协议层 (DS-Tauri-Unified Finder Protocol)
//!
//! DSTU 是 VFS 与上层应用之间的统一访问接口，类似于操作系统的文件管理器协议。
//! 提供文件系统语义的统一接口，使所有模块（笔记、教材、题目集等）可以通过相同的 API 访问资源。
//!
//! ## 设计目标
//! 1. **统一访问接口**：所有模块通过 DSTU 访问资源，消除各模块直接访问不同数据库的混乱
//! 2. **文件系统语义**：使用路径（path）定位资源，支持目录遍历、移动、复制等操作
//! 3. **解耦存储实现**：DSTU 不关心 VFS 内部实现，便于未来扩展
//!
//! ## 路径规范
//! ```text
//! 路径格式：/{folder_path}/{resource_id}
//!
//! 示例：
//! - /高考复习/函数/note_abc123   → 在"高考复习/函数"文件夹下的笔记
//! - /我的教材/tb_xyz789          → 在"我的教材"文件夹下的教材
//! - /exam_sheet_001              → 根目录下的题目集（无文件夹）
//! - /                            → 根目录
//! - /@trash                      → 回收站（虚拟路径）
//! ```
//!
//! ## 模块结构
//! - `types` - DSTU 类型定义（DstuNode、DstuNodeType 等）
//! - `error` - 错误类型（DstuError、DstuResult）
//! - `path_parser` - 路径解析器
//! - `handlers` - Tauri 命令处理器（Prompt 5 实现）

pub mod error;
pub mod exam_formatter;
pub mod export; // 统一资源导出模块
pub mod handler_utils; // 路径工具和节点转换器
pub mod handlers;
pub mod path_parser;
pub mod path_types; // 新增：契约 C1 类型定义
pub mod types;

// ============================================================================
// 重导出核心类型
// ============================================================================

// 错误类型
pub use error::{DstuError, DstuResult};

// 路径解析器（辅助函数）
pub use path_parser::{build_simple_resource_path, get_parent_path, get_path_name, is_parent_path};

// 路径解析器（新 API，契约 B/C1）
pub use path_parser::{
    build_real_path, extract_folder_path, extract_resource_id, get_resource_type, is_valid_path,
    is_valid_resource_id, parse_real_path, RealParsedPath, RESOURCE_ID_PREFIXES,
    VIRTUAL_PATH_TYPES,
};

// 路径类型（契约 C1）
pub use path_types::{
    get_resource_type_from_id, is_virtual_path_type, ParsedPath as NewParsedPath,
};

// 核心类型
pub use types::{
    BatchMoveRequest,
    DstuCreateOptions,
    DstuListOptions,
    DstuNode,
    DstuNodeType,
    // 契约 C: 真实路径架构类型（文档 28）
    DstuParsedPath,
    DstuWatchEvent,
    DstuWatchEventType,
    PathCacheEntry,
    ResourceLocation,
};

// handlers 导出（Prompt 5 实现）
pub use handlers::{
    dstu_copy,
    // 题目集识别多模态内容获取（文档 25 实现）
    dstu_get_exam_content,
    dstu_move,
    dstu_move_many,
    // 文件夹内搜索
    dstu_search_in_folder,
};
