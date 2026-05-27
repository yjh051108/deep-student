pub mod audit_log;
pub mod auto_extractor;
pub mod category_manager;
pub mod compressor;
pub mod config;
pub mod evolution;
pub mod handlers;
pub mod llm_decision;
pub mod query_rewriter;
pub mod reranker;
pub mod scope;
pub mod service;

pub use audit_log::{MemoryAuditLogItem, MemoryAuditLogger, MemoryOpSource, MemoryOpType, OpTimer};
pub use auto_extractor::MemoryAutoExtractor;
pub use category_manager::MemoryCategoryManager;
pub use compressor::MemoryCompressor;
pub use config::{AutoExtractFrequency, MemoryConfig};
pub use evolution::MemoryEvolution;
pub use handlers::*;
pub use llm_decision::{
    MemoryDecisionResponse, MemoryEvent, MemoryLLMDecision, SimilarMemorySummary,
};
pub use query_rewriter::{MemoryQueryRewriter, QueryRewriteResult};
pub use reranker::MemoryReranker;
pub use scope::{
    classify_folder_scope, is_folder_path_visible, is_folder_path_within_scope,
    join_memory_folder_paths, sanitize_scope_segment, scoped_folder_path, topic_memory_root,
    visible_scope_roots, MemoryScope, GLOBAL_MEMORY_FOLDER, TOPIC_MEMORY_PREFIX,
};
pub use service::{
    MemoryListItem, MemoryPurpose, MemorySearchResult, MemoryService, MemoryType, SmartWriteOutput,
    WriteMode,
};
