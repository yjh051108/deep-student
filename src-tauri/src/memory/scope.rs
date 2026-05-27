use serde::{Deserialize, Serialize};

pub const GLOBAL_MEMORY_FOLDER: &str = "全局";
pub const TOPIC_MEMORY_PREFIX: &str = "课题";

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MemoryScope {
    Topic,
    Global,
}

impl MemoryScope {
    pub fn from_arg(raw: Option<&str>) -> Result<Self, String> {
        match raw.map(|s| s.trim().to_lowercase()).as_deref() {
            Some("global") => Ok(Self::Global),
            Some("topic") | None | Some("") => Ok(Self::Topic),
            Some(other) => Err(format!(
                "Invalid memory scope '{}': expected topic or global",
                other
            )),
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Topic => "topic",
            Self::Global => "global",
        }
    }
}

pub fn sanitize_scope_segment(value: &str) -> String {
    let sanitized: String = value
        .chars()
        .map(|ch| {
            if ch.is_control() || matches!(ch, '/' | '\\') {
                '_'
            } else {
                ch
            }
        })
        .collect();
    sanitized.trim_matches('_').trim().to_string()
}

pub fn topic_memory_root(group_id: Option<&str>, group_name: Option<&str>) -> Option<String> {
    let label = group_name.or(group_id)?.trim();
    if label.is_empty() {
        return None;
    }
    let segment = sanitize_scope_segment(label);
    if segment.is_empty() {
        None
    } else {
        Some(format!("{}/{}", TOPIC_MEMORY_PREFIX, segment))
    }
}

pub fn join_memory_folder_paths(base: &str, child: Option<&str>) -> String {
    let child = child.map(str::trim).filter(|value| !value.is_empty());
    match child {
        Some(child) => format!(
            "{}/{}",
            base.trim_matches('/'),
            child.trim_start_matches('/')
        ),
        None => base.trim_matches('/').to_string(),
    }
}

pub fn scoped_folder_path(
    group_id: Option<&str>,
    group_name: Option<&str>,
    scope: MemoryScope,
    folder: Option<&str>,
) -> Option<String> {
    let base = match scope {
        MemoryScope::Global => Some(GLOBAL_MEMORY_FOLDER.to_string()),
        MemoryScope::Topic => topic_memory_root(group_id, group_name),
    }?;
    Some(join_memory_folder_paths(&base, folder))
}

pub fn visible_scope_roots(group_id: Option<&str>, group_name: Option<&str>) -> Vec<String> {
    let mut roots = vec![GLOBAL_MEMORY_FOLDER.to_string()];
    if let Some(topic_root) = topic_memory_root(group_id, group_name) {
        roots.push(topic_root);
    }
    roots
}

pub fn is_folder_path_within_scope(folder_path: &str, scope_root: &str) -> bool {
    folder_path == scope_root
        || folder_path
            .strip_prefix(scope_root)
            .map(|rest| rest.starts_with('/'))
            .unwrap_or(false)
}

pub fn classify_folder_scope(
    folder_path: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
) -> Option<MemoryScope> {
    if is_folder_path_within_scope(folder_path, GLOBAL_MEMORY_FOLDER) {
        return Some(MemoryScope::Global);
    }
    let topic_root = topic_memory_root(group_id, group_name)?;
    if is_folder_path_within_scope(folder_path, &topic_root) {
        Some(MemoryScope::Topic)
    } else {
        None
    }
}

pub fn is_folder_path_visible(
    folder_path: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
) -> bool {
    classify_folder_scope(folder_path, group_id, group_name).is_some()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sanitizes_topic_names() {
        assert_eq!(
            topic_memory_root(Some("id"), Some("微机/原理\\A")).as_deref(),
            Some("课题/微机_原理_A")
        );
    }

    #[test]
    fn classifies_visible_scopes() {
        assert_eq!(
            classify_folder_scope("全局/偏好", Some("g1"), Some("微机原理")),
            Some(MemoryScope::Global)
        );
        assert_eq!(
            classify_folder_scope("课题/微机原理/经历", Some("g1"), Some("微机原理")),
            Some(MemoryScope::Topic)
        );
        assert_eq!(
            classify_folder_scope("课题/电磁场/经历", Some("g1"), Some("微机原理")),
            None
        );
    }
}
