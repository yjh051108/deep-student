//! Transitional Chat V2 replay helpers.
//!
//! The legacy send/retry/edit/continue Tauri command surface has been retired in
//! favor of the Go/Wails `ChatService`. Keep this small helper only while the old
//! Rust variant retry commands still depend on the replay snapshot merge logic.

use crate::chat_v2::types::{MessageMeta, ReplayMode, SendOptions};

pub(crate) fn apply_original_skill_snapshot_overrides(
    mut options: SendOptions,
    preferred_meta: Option<&MessageMeta>,
    fallback_meta: Option<&MessageMeta>,
) -> SendOptions {
    if options.replay_mode != Some(ReplayMode::Original) {
        return options;
    }

    let snapshot = preferred_meta
        .and_then(|meta| {
            meta.skill_snapshot_after
                .as_ref()
                .or(meta.skill_snapshot_before.as_ref())
        })
        .or_else(|| {
            fallback_meta.and_then(|meta| {
                meta.skill_snapshot_after
                    .as_ref()
                    .or(meta.skill_snapshot_before.as_ref())
            })
        });

    let runtime_snapshot = preferred_meta
        .and_then(|meta| {
            meta.skill_runtime_after
                .as_ref()
                .or(meta.skill_runtime_before.as_ref())
        })
        .or_else(|| {
            fallback_meta.and_then(|meta| {
                meta.skill_runtime_after
                    .as_ref()
                    .or(meta.skill_runtime_before.as_ref())
            })
        });

    if snapshot.is_none() && runtime_snapshot.is_none() {
        return options;
    }

    let mut replay_pinned_skill_ids = runtime_snapshot
        .map(|snapshot| snapshot.active_skill_ids.clone())
        .unwrap_or_default();
    if let Some(snapshot) = snapshot {
        replay_pinned_skill_ids = snapshot.manual_pinned_skill_ids.clone();
    }
    replay_pinned_skill_ids.sort();
    replay_pinned_skill_ids.dedup();

    if !replay_pinned_skill_ids.is_empty() {
        options.active_skill_ids = Some(replay_pinned_skill_ids);
    }

    if let Some(runtime_snapshot) = runtime_snapshot {
        if !runtime_snapshot.skill_contents.is_empty() {
            options.skill_contents = Some(runtime_snapshot.skill_contents.clone());
            options.replay_skill_contents = Some(runtime_snapshot.skill_contents.clone());
        }
        if !runtime_snapshot.skill_dependencies.is_empty() {
            options.skill_dependencies = Some(runtime_snapshot.skill_dependencies.clone());
        }
        if !runtime_snapshot.skill_embedded_tools.is_empty() {
            options.skill_embedded_tools = Some(runtime_snapshot.skill_embedded_tools.clone());
        }
        if !runtime_snapshot.mcp_tool_schemas.is_empty() {
            options.mcp_tool_schemas = Some(runtime_snapshot.mcp_tool_schemas.clone());
        }
        if !runtime_snapshot.selected_mcp_servers.is_empty() {
            options.mcp_tools = Some(runtime_snapshot.selected_mcp_servers.clone());
        }
    } else if let Some(snapshot) = snapshot {
        if !snapshot.effective_allowed_external_servers.is_empty() {
            options.mcp_tools = Some(snapshot.effective_allowed_external_servers.clone());
        }
    }

    options
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::chat_v2::types::{MessageMeta, ReplaySkillPayloadSnapshot, SkillStateSnapshot};

    #[test]
    fn apply_original_skill_snapshot_overrides_restores_manual_pins_only() {
        let options = SendOptions {
            replay_mode: Some(ReplayMode::Original),
            ..Default::default()
        };
        let meta = MessageMeta {
            skill_snapshot_after: Some(SkillStateSnapshot {
                manual_pinned_skill_ids: vec!["manual-a".to_string()],
                mode_required_bundle_ids: vec!["mode-a".to_string()],
                agentic_session_skill_ids: vec!["agentic-a".to_string()],
                branch_local_skill_ids: vec!["branch-a".to_string()],
                effective_allowed_internal_tools: vec!["builtin-web_search".to_string()],
                effective_allowed_external_tools: vec!["mcp_fetch".to_string()],
                effective_allowed_external_servers: vec!["server-a".to_string()],
                ..Default::default()
            }),
            ..Default::default()
        };

        let updated = apply_original_skill_snapshot_overrides(options, Some(&meta), None);
        assert_eq!(
            updated.active_skill_ids.unwrap(),
            vec!["manual-a".to_string()]
        );
        assert_eq!(updated.mcp_tools.unwrap(), vec!["server-a".to_string()]);
    }

    #[test]
    fn apply_original_skill_snapshot_overrides_restores_runtime_skill_payload() {
        let options = SendOptions {
            replay_mode: Some(ReplayMode::Original),
            ..Default::default()
        };
        let meta = MessageMeta {
            skill_runtime_after: Some(ReplaySkillPayloadSnapshot {
                active_skill_ids: vec!["runtime-skill".to_string()],
                skill_contents: std::collections::HashMap::from([(
                    "runtime-skill".to_string(),
                    "runtime body".to_string(),
                )]),
                skill_dependencies: std::collections::HashMap::from([(
                    "runtime-skill".to_string(),
                    vec!["dep-a".to_string()],
                )]),
                skill_embedded_tools: std::collections::HashMap::new(),
                mcp_tool_schemas: vec![crate::chat_v2::types::McpToolSchema {
                    name: "fetch".to_string(),
                    server_id: Some("server-a".to_string()),
                    description: Some("fetch from server a".to_string()),
                    input_schema: Some(serde_json::json!({ "type": "object" })),
                }],
                selected_mcp_servers: vec!["server-a".to_string()],
            }),
            ..Default::default()
        };

        let updated = apply_original_skill_snapshot_overrides(options, Some(&meta), None);
        assert_eq!(
            updated.active_skill_ids.unwrap(),
            vec!["runtime-skill".to_string()]
        );
        assert_eq!(
            updated
                .skill_contents
                .unwrap()
                .get("runtime-skill")
                .map(String::as_str),
            Some("runtime body")
        );
        assert_eq!(
            updated
                .replay_skill_contents
                .unwrap()
                .get("runtime-skill")
                .map(String::as_str),
            Some("runtime body")
        );
        assert_eq!(
            updated
                .skill_dependencies
                .unwrap()
                .get("runtime-skill")
                .cloned()
                .unwrap(),
            vec!["dep-a".to_string()]
        );
        assert_eq!(updated.mcp_tools.unwrap(), vec!["server-a".to_string()]);
        assert_eq!(
            updated.mcp_tool_schemas.unwrap()[0].server_id.as_deref(),
            Some("server-a")
        );
    }

    #[test]
    fn apply_original_skill_snapshot_overrides_keeps_current_replay_unchanged() {
        let options = SendOptions {
            replay_mode: Some(ReplayMode::Current),
            active_skill_ids: Some(vec!["current-skill".to_string()]),
            ..Default::default()
        };
        let meta = MessageMeta {
            skill_runtime_after: Some(ReplaySkillPayloadSnapshot {
                active_skill_ids: vec!["historical-skill".to_string()],
                ..Default::default()
            }),
            ..Default::default()
        };

        let updated = apply_original_skill_snapshot_overrides(options, Some(&meta), None);
        assert_eq!(
            updated.active_skill_ids.unwrap(),
            vec!["current-skill".to_string()]
        );
    }

    #[test]
    fn apply_original_skill_snapshot_overrides_prefers_preferred_meta_over_fallback() {
        let options = SendOptions {
            replay_mode: Some(ReplayMode::Original),
            ..Default::default()
        };
        let preferred = MessageMeta {
            skill_runtime_after: Some(ReplaySkillPayloadSnapshot {
                active_skill_ids: vec!["variant-skill".to_string()],
                ..Default::default()
            }),
            ..Default::default()
        };
        let fallback = MessageMeta {
            skill_runtime_after: Some(ReplaySkillPayloadSnapshot {
                active_skill_ids: vec!["message-skill".to_string()],
                ..Default::default()
            }),
            ..Default::default()
        };

        let updated =
            apply_original_skill_snapshot_overrides(options, Some(&preferred), Some(&fallback));
        assert_eq!(
            updated.active_skill_ids.unwrap(),
            vec!["variant-skill".to_string()]
        );
    }
}
