import { invoke, isInjectedNativeRuntime, isTauriRuntime, isWailsRuntime } from '@/runtime/native';

import type { ApiConfig, ModelAssignments } from '@/types';

import { loadVoiceInputConfig } from './config';
import { resolveVoiceInputModelAssignment } from './modelSelection';
import type { VoiceInputRuntimeConfig } from './types';

const EMPTY_ASSIGNMENTS: Pick<ModelAssignments, 'voice_input_asr_model_config_id'> = {
  voice_input_asr_model_config_id: null,
};

function isNativeRuntime(): boolean {
  return isInjectedNativeRuntime() || isTauriRuntime() || isWailsRuntime();
}

export async function loadVoiceInputRuntimeConfig(): Promise<VoiceInputRuntimeConfig> {
  const behaviorConfig = await loadVoiceInputConfig();

  if (!isNativeRuntime()) {
    return {
      ...behaviorConfig,
      assignedModel: resolveVoiceInputModelAssignment(EMPTY_ASSIGNMENTS, []),
    };
  }

  const [assignments, apis] = await Promise.all([
    invoke<Pick<ModelAssignments, 'voice_input_asr_model_config_id'>>('get_model_assignments').catch(
      () => EMPTY_ASSIGNMENTS
    ),
    invoke<ApiConfig[]>('get_api_configurations').catch(() => []),
  ]);

  return {
    ...behaviorConfig,
    assignedModel: resolveVoiceInputModelAssignment(assignments, apis),
  };
}
