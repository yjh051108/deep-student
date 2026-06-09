//! OCR 引擎相关命令
//!
//! 提供 OCR 引擎配置和管理的 Tauri 命令。

use crate::commands::AppState;
use crate::llm_manager::OcrModelConfig;
use crate::models::AppError;
use crate::ocr_adapters::{OcrAdapterFactory, OcrEngineType};
use serde::Serialize;
use tauri::State;

type Result<T> = std::result::Result<T, AppError>;

/// M14 fix: PaddleOCR-VL 自动迁移到 1.5 版本（共享函数）
///
/// 返回 true 表示有变更需要保存
pub fn migrate_paddle_ocr_models(models: &mut [OcrModelConfig]) -> bool {
    let mut changed = false;
    for model in models.iter_mut() {
        // 跳过显式配置为旧版的引擎（paddle_ocr_vl_v1 故意使用 PaddleOCR-VL）
        if model.model == "PaddlePaddle/PaddleOCR-VL" && model.engine_type != "paddle_ocr_vl_v1" {
            model.model = "PaddlePaddle/PaddleOCR-VL-1.5".to_string();
            if model.name.contains("PaddleOCR-VL") && !model.name.contains("1.5") {
                model.name = model.name.replace("PaddleOCR-VL", "PaddleOCR-VL-1.5");
            }
            changed = true;
        }
    }
    changed
}

/// 自动迁移：将旧版 GLM-4.1V 替换为 GLM-4.6V
///
/// GLM-4.1V-9B-Thinking (9B) 是低质量模型，已被 GLM-4.6V (106B MoE) 替代。
/// 返回 true 表示有变更需要保存。
pub fn migrate_glm_ocr_models(models: &mut [OcrModelConfig]) -> bool {
    let mut changed = false;
    for model in models.iter_mut() {
        if model.engine_type == "glm4v_ocr" && model.model.to_lowercase().contains("glm-4.1v") {
            model.model = "zai-org/GLM-4.6V".to_string();
            if model.name.contains("4.1V") || model.name.contains("4.1v") {
                model.name = model.name.replace("4.1V", "4.6V").replace("4.1v", "4.6V");
            }
            changed = true;
            println!("[OCR] 已自动迁移 GLM-4.1V → GLM-4.6V");
        }
    }
    changed
}

/// 设置 OCR 引擎类型
#[tauri::command]
pub async fn set_ocr_engine_type(engine_type: String, state: State<'_, AppState>) -> Result<bool> {
    // M5 fix: 严格验证引擎类型，拒绝非法输入
    let parsed = OcrEngineType::try_from_str(&engine_type).ok_or_else(|| {
        AppError::validation(format!(
            "Unknown OCR engine type: '{}'. Valid types: deepseek_ocr, paddle_ocr_vl, paddle_ocr_vl_v1, generic_vlm",
            engine_type
        ))
    })?;

    let db = &state.database;
    db.save_setting("ocr.engine_type", parsed.as_str())
        .map_err(|e| AppError::database(format!("保存 OCR 引擎配置失败: {}", e)))?;

    Ok(true)
}

/// 根据模型名称推断 OCR 引擎类型
#[tauri::command]
pub async fn infer_ocr_engine_from_model(model: String) -> Result<String> {
    let engine = OcrAdapterFactory::infer_engine_from_model(&model);
    Ok(engine.as_str().to_string())
}

/// 验证模型是否适合指定的 OCR 引擎
#[tauri::command]
pub async fn validate_ocr_model(
    model: String,
    engine_type: String,
) -> Result<ValidateOcrModelResponse> {
    let engine = OcrEngineType::from_str(&engine_type);
    let is_valid = OcrAdapterFactory::validate_model_for_engine(&model, engine);
    let recommended = engine.recommended_model().to_string();

    Ok(ValidateOcrModelResponse {
        is_valid,
        recommended_model: if !is_valid { Some(recommended) } else { None },
    })
}

/// 验证结果
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ValidateOcrModelResponse {
    pub is_valid: bool,
    pub recommended_model: Option<String>,
}

/// 获取 OCR 引擎的 prompt 模板
#[tauri::command]
pub async fn get_ocr_prompt_template(engine_type: String, mode: String) -> Result<String> {
    use crate::ocr_adapters::OcrMode;

    let engine = OcrEngineType::from_str(&engine_type);
    let mode = OcrMode::from_str(&mode);

    let adapter = OcrAdapterFactory::create(engine);
    let prompt = adapter.build_prompt(mode);

    Ok(prompt)
}
