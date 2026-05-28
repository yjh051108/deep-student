use super::*;

impl ChatV2Pipeline {
    /// 构建系统提示
    ///
    /// 使用 prompt_builder 模块统一格式化，采用 XML 标签分隔各部分，
    /// 统一引用格式为 `[类型-编号]`，并添加使用指引。
    /// 如果有 Canvas 笔记，也会一并注入。
    pub(crate) async fn build_system_prompt(&self, ctx: &PipelineContext) -> String {
        let canvas_note = self.build_canvas_note_info(ctx).await;

        // 读取 scoped 记忆上下文（如果 VFS 可用）
        let memory_context = self.load_memory_prompt_context(&ctx.options).await;

        prompt_builder::build_system_prompt_with_memory_context(
            &ctx.options,
            &ctx.retrieved_sources,
            canvas_note,
            memory_context,
        )
    }

    /// 从 MemoryService 读取用户画像 + 分类摘要（双模检索的 LLM 直读模式）
    ///
    /// 受 memU dual-mode retrieval 启发：
    /// - LLM 直读模式（本方法）：将分类文件注入 system prompt，每次对话都有
    /// - 向量搜索模式（memory_search 工具）：LLM 按需主动搜索
    async fn load_memory_prompt_context(
        &self,
        options: &crate::chat_v2::types::SendOptions,
    ) -> Option<prompt_builder::MemoryPromptContext> {
        use crate::memory::{MemoryCategoryManager, MemoryConfig, MemoryService};
        use crate::vfs::lance_store::VfsLanceStore;

        if options.memory_enabled == Some(false) {
            return None;
        }

        let vfs_db = self.vfs_db.as_ref()?;
        let mem_cfg = MemoryConfig::new(vfs_db.clone());
        if mem_cfg.is_privacy_mode().ok()? {
            return None;
        }
        let lance_store = VfsLanceStore::new(vfs_db.clone())
            .ok()
            .map(std::sync::Arc::new)?;
        let svc = MemoryService::new(vfs_db.clone(), lance_store, self.llm_manager.clone());

        let root_id = match svc.get_root_folder_id() {
            Ok(Some(id)) => id,
            _ => return None,
        };

        let topic_root = crate::memory::topic_memory_root(
            options.group_id.as_deref(),
            options.group_name.as_deref(),
        );
        let scope_paths = crate::memory::visible_scope_roots(
            options.group_id.as_deref(),
            options.group_name.as_deref(),
        );

        // 加载 scoped 分类摘要文件（Memory Category Layer）
        let cat_mgr = MemoryCategoryManager::new(vfs_db.clone(), self.llm_manager.clone());
        let categories = match cat_mgr.load_category_summaries_for_paths(&root_id, &scope_paths) {
            Ok(categories) => categories,
            Err(e) => {
                log::debug!(
                    "[ChatV2::pipeline] Failed to load scoped category summaries: {}",
                    e
                );
                Vec::new()
            }
        };

        let mut global_sections = Vec::new();
        let mut topic_sections = Vec::new();
        for (cat_name, content) in categories {
            let section = format!("### {}\n{}", cat_name, content);
            if crate::memory::is_folder_path_within_scope(
                &cat_name,
                crate::memory::GLOBAL_MEMORY_FOLDER,
            ) {
                global_sections.push(section);
            } else {
                let topic_roots = crate::memory::topic_memory_roots(
                    options.group_id.as_deref(),
                    options.group_name.as_deref(),
                );
                if topic_roots
                    .iter()
                    .any(|root| crate::memory::is_folder_path_within_scope(&cat_name, root))
                {
                    topic_sections.push(section);
                }
            }
        }

        fn join_limited(sections: &[String], max_chars: usize) -> Option<String> {
            if sections.is_empty() {
                return None;
            }
            let mut total_chars = 0usize;
            let mut kept_sections = Vec::new();
            for section in sections {
                let section_chars = section.chars().count();
                if total_chars + section_chars > max_chars && !kept_sections.is_empty() {
                    break;
                }
                total_chars += section_chars + 2;
                kept_sections.push(section.as_str());
            }
            let combined = kept_sections.join("\n\n");
            if kept_sections.len() < sections.len() {
                Some(format!(
                    "{}\n\n（记忆摘要已截断 {}/{} 个分类，完整信息请使用 memory_search 工具检索）",
                    combined,
                    kept_sections.len(),
                    sections.len()
                ))
            } else {
                Some(combined)
            }
        }

        let global_profile = join_limited(&global_sections, 1200);
        let topic_profile = join_limited(&topic_sections, 1600);
        if global_profile.is_none() && topic_profile.is_none() && topic_root.is_none() {
            return None;
        }

        Some(prompt_builder::MemoryPromptContext::new(
            options.group_name.clone().or(options.group_id.clone()),
            topic_root,
            crate::memory::GLOBAL_MEMORY_FOLDER.to_string(),
            global_profile,
            topic_profile,
        ))
    }

    /// 构建 Canvas 笔记信息
    async fn build_canvas_note_info(
        &self,
        ctx: &PipelineContext,
    ) -> Option<prompt_builder::CanvasNoteInfo> {
        let note_id = ctx.options.canvas_note_id.as_ref()?;
        let notes_mgr = self.notes_manager.as_ref()?;
        match notes_mgr.get_note(note_id) {
            Ok(note) => {
                let word_count = note.content_md.chars().count();
                log::info!(
                    "[ChatV2::pipeline] Canvas mode: loaded note '{}' ({} chars, is_long={})",
                    note.title,
                    word_count,
                    word_count >= 3000
                );
                Some(prompt_builder::CanvasNoteInfo::new(
                    note_id.clone(),
                    note.title,
                    note.content_md,
                ))
            }
            Err(e) => {
                log::warn!(
                    "[ChatV2::pipeline] Canvas mode: failed to read note {}: {}",
                    note_id,
                    e
                );
                None
            }
        }
    }

    /// 构建当前用户消息（用于 LLM 调用）
    ///
    /// ★ 2025-12-10 统一改造：移除 ctx.attachments 的直接处理
    /// 所有附件现在通过 user_context_refs 传递，图片和文档内容已在前端 formatToBlocks 中处理
    ///
    /// ## 统一上下文注入系统（Prompt 8）
    /// 使用 `get_combined_user_content()` 合并上下文内容和用户输入，
    /// 将 formattedBlocks 中的文本拼接到用户内容前面，图片添加到 image_base64。
    ///
    /// ## ★ 文档25：多模态图文交替支持
    /// 当上下文引用包含图片时，使用 `get_content_blocks_ordered()` 获取有序内容块，
    /// 填充 `multimodal_content` 字段以保持图文交替顺序。
    pub(crate) fn build_current_user_message(&self, ctx: &PipelineContext) -> LegacyChatMessage {
        // ★ 文档25：检查上下文引用是否包含图片（需要图文交替）
        let has_context_images = ctx.user_context_refs.iter().any(|r| {
            r.formatted_blocks
                .iter()
                .any(|b| matches!(b, ContentBlock::Image { .. }))
        });

        // ★ 2025-12-10 统一改造：所有内容都通过 user_context_refs 传递
        // 不再从 ctx.attachments 提取图片和文档

        let (combined_content, image_base64, multimodal_content) = if has_context_images {
            let (text_fallback_content, _) = ctx.get_combined_user_content();

            // 使用 get_content_blocks_ordered() 获取图文交替的内容块
            let ordered_blocks = ctx.get_content_blocks_ordered();

            // 转换为 MultimodalContentPart 数组
            let multimodal_parts: Vec<MultimodalContentPart> = ordered_blocks
                .into_iter()
                .map(|block| match block {
                    ContentBlock::Text { text } => MultimodalContentPart::text(text),
                    ContentBlock::Image { media_type, base64 } => {
                        MultimodalContentPart::image(media_type, base64)
                    }
                })
                .collect();

            log::info!(
                "[ChatV2::pipeline] build_current_user_message: Using multimodal mode with {} parts from context refs",
                multimodal_parts.len()
            );

            // 关键修复：即使构造 multimodal_content，也保留文本 fallback。
            // 这样文本模型或错误路由到非多模态配置时，不会因为 content 为空而丢失上下文。
            (text_fallback_content, None, Some(multimodal_parts))
        } else {
            // 传统模式：使用 get_combined_user_content()
            let (combined_content, context_images) = ctx.get_combined_user_content();

            let image_base64: Option<Vec<String>> = if context_images.is_empty() {
                None
            } else {
                Some(context_images)
            };

            (combined_content, image_base64, None)
        };

        // ★ 2025-12-10 统一改造：doc_attachments 不再从 ctx.attachments 构建
        // 文档内容现在通过 user_context_refs 的 formattedBlocks 传递（已由 formatToBlocks 解析）

        LegacyChatMessage {
            role: "user".to_string(),
            content: combined_content,
            timestamp: chrono::Utc::now(),
            thinking_content: None,
            thought_signature: None,
            rag_sources: None,
            memory_sources: None,
            graph_sources: None,
            web_search_sources: None,
            image_paths: None,
            image_base64,
            doc_attachments: None, // ★ 文档附件现在通过 user_context_refs 传递
            multimodal_content,    // ★ 文档25：多模态图文交替内容
            tool_call: None,
            tool_result: None,
            overrides: None,
            relations: None,
            persistent_stable_id: None,
            metadata: None,
        }
    }
}
