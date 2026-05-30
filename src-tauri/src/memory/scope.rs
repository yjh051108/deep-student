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

fn topic_memory_root_from_label(label: &str) -> Option<String> {
    let label = label.trim();
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

pub fn topic_memory_root(group_id: Option<&str>, group_name: Option<&str>) -> Option<String> {
    group_name
        .and_then(topic_memory_root_from_label)
        .or_else(|| group_id.and_then(topic_memory_root_from_label))
}

pub fn legacy_topic_memory_root(
    group_id: Option<&str>,
    group_name: Option<&str>,
) -> Option<String> {
    let id_root = group_id.and_then(topic_memory_root_from_label)?;
    match topic_memory_root(group_id, group_name) {
        Some(primary) if primary == id_root => None,
        _ => Some(id_root),
    }
}

pub fn topic_memory_roots(group_id: Option<&str>, group_name: Option<&str>) -> Vec<String> {
    let mut roots = Vec::new();
    if let Some(root) = topic_memory_root(group_id, group_name) {
        roots.push(root);
    }
    if let Some(root) = legacy_topic_memory_root(group_id, group_name) {
        if !roots.iter().any(|existing| existing == &root) {
            roots.push(root);
        }
    }
    roots
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
    for topic_root in topic_memory_roots(group_id, group_name) {
        roots.push(topic_root);
    }
    roots
}

pub fn readable_scope_roots(group_id: Option<&str>, group_name: Option<&str>) -> Vec<String> {
    let roots = visible_scope_roots(group_id, group_name);
    if roots.len() > 1 {
        return roots;
    }
    vec![
        GLOBAL_MEMORY_FOLDER.to_string(),
        TOPIC_MEMORY_PREFIX.to_string(),
    ]
}

pub fn is_folder_path_within_scope(folder_path: &str, scope_root: &str) -> bool {
    folder_path_scope_candidates(folder_path)
        .into_iter()
        .any(|candidate| {
            candidate == scope_root
                || candidate
                    .strip_prefix(scope_root)
                    .map(|rest| rest.starts_with('/'))
                    .unwrap_or(false)
        })
}

fn folder_path_scope_candidates(folder_path: &str) -> Vec<&str> {
    let trimmed = folder_path.trim_matches('/');
    if trimmed.is_empty() {
        return Vec::new();
    }

    let mut candidates = vec![trimmed];
    for marker in [GLOBAL_MEMORY_FOLDER, TOPIC_MEMORY_PREFIX] {
        let mut offset = 0;
        for segment in trimmed.split('/') {
            if segment == marker {
                let suffix = &trimmed[offset..];
                if suffix != trimmed && !candidates.iter().any(|candidate| *candidate == suffix) {
                    candidates.push(suffix);
                }
            }
            offset += segment.len() + 1;
        }
    }
    candidates
}

pub fn classify_folder_scope(
    folder_path: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
) -> Option<MemoryScope> {
    if is_folder_path_within_scope(folder_path, GLOBAL_MEMORY_FOLDER) {
        return Some(MemoryScope::Global);
    }
    for topic_root in topic_memory_roots(group_id, group_name) {
        if is_folder_path_within_scope(folder_path, &topic_root) {
            return Some(MemoryScope::Topic);
        }
    }
    None
}

pub fn is_folder_path_visible(
    folder_path: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
) -> bool {
    classify_folder_scope(folder_path, group_id, group_name).is_some()
}

pub fn is_folder_path_readable(
    folder_path: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
) -> bool {
    readable_scope_roots(group_id, group_name)
        .iter()
        .any(|root| is_folder_path_within_scope(folder_path, root))
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
        assert_eq!(
            legacy_topic_memory_root(Some("id"), Some("微机/原理\\A")).as_deref(),
            Some("课题/id")
        );
        assert_eq!(
            topic_memory_roots(Some("id"), Some("微机/原理\\A")),
            vec!["课题/微机_原理_A".to_string(), "课题/id".to_string()]
        );
    }

    #[test]
    fn classifies_visible_scopes() {
        assert_eq!(
            classify_folder_scope("全局/偏好", Some("g1"), Some("微机原理")),
            Some(MemoryScope::Global)
        );
        assert_eq!(
            classify_folder_scope("长期记忆/全局/偏好", Some("g1"), Some("微机原理")),
            Some(MemoryScope::Global)
        );
        assert_eq!(
            classify_folder_scope("长期记忆/全局", Some("g1"), Some("微机原理")),
            Some(MemoryScope::Global)
        );
        assert_eq!(
            classify_folder_scope("课题/g1/经历", Some("g1"), Some("微机原理")),
            Some(MemoryScope::Topic)
        );
        assert_eq!(
            classify_folder_scope("长期记忆/课题/g1/经历", Some("g1"), Some("微机原理")),
            Some(MemoryScope::Topic)
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

    #[test]
    fn general_session_reads_global_and_all_topics() {
        assert_eq!(
            readable_scope_roots(None, None),
            vec!["全局".to_string(), "课题".to_string()]
        );
        assert!(is_folder_path_readable("课题/微机原理/经历", None, None));
        assert!(is_folder_path_readable("长期记忆/课题/电磁场/经历", None, None));
        assert!(is_folder_path_readable("全局/偏好", None, None));
    }
}
