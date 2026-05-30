/**
 * Learning Hub 导航上下文
 *
 * 用于在 Topbar 和 LearningHubPage 之间共享文件夹导航状态。
 * 
 * ★ 文档28 Prompt 8: 集成真实路径导航系统
 */

import React, { createContext, useContext, useState, useCallback, useMemo, useEffect } from 'react';
import type { RealPathBreadcrumbItem } from './hooks/useFolderNavigation';
import { useFinderStore } from './stores/finderStore';

// ============================================================================
// 📱 全局导航 Ref（解决 App.tsx 无法访问 Context 的问题）
// ============================================================================

/**
 * 全局导航状态，用于在 Provider 外部（App.tsx）访问导航状态
 * 这是必要的，因为 App.tsx 渲染 Provider，所以它本身不在 Provider 内部
 */
interface GlobalNavigationState {
  canGoBack: boolean;
  canGoForward: boolean;
  canGoUp: boolean;
  goBack: () => void;
  goForward: () => void;
  goUp: () => void;
}

const globalNavigationRef: { current: GlobalNavigationState | null } = { current: null };
const finderNavigationRef: { current: GlobalNavigationState | null } = { current: null };
const localBackRef: { current: Pick<GlobalNavigationState, 'canGoBack' | 'goBack'> | null } = { current: null };

function publishGlobalLearningHubNavigation() {
  const finderState = finderNavigationRef.current;
  if (!finderState) {
    globalNavigationRef.current = null;
    window.dispatchEvent(new CustomEvent(LEARNING_HUB_NAV_STATE_CHANGED, { detail: null }));
    return;
  }

  const localBack = localBackRef.current;
  const state: GlobalNavigationState = localBack?.canGoBack
    ? { ...finderState, canGoBack: true, goBack: localBack.goBack }
    : finderState;

  globalNavigationRef.current = state;
  window.dispatchEvent(new CustomEvent(LEARNING_HUB_NAV_STATE_CHANGED, { detail: state }));
}

export function setLearningHubLocalBackHandler(
  state: Pick<GlobalNavigationState, 'canGoBack' | 'goBack'> | null
): () => void {
  localBackRef.current = state;
  publishGlobalLearningHubNavigation();
  return () => {
    if (localBackRef.current === state) {
      localBackRef.current = null;
      publishGlobalLearningHubNavigation();
    }
  };
}

/**
 * 获取全局导航状态（供 App.tsx 使用）
 */
export function getGlobalLearningHubNavigation(): GlobalNavigationState | null {
  return globalNavigationRef.current;
}

/**
 * 导航状态变化事件名
 */
export const LEARNING_HUB_NAV_STATE_CHANGED = 'learningHubNavStateChanged';

/**
 * 订阅导航状态变化（供 App.tsx 使用）
 * @param callback 状态变化回调
 * @returns 取消订阅函数
 */
export function subscribeLearningHubNavigation(
  callback: (state: GlobalNavigationState | null) => void
): () => void {
  const handler = (evt: Event) => {
    const customEvt = evt as CustomEvent<GlobalNavigationState>;
    callback(customEvt.detail);
  };
  window.addEventListener(LEARNING_HUB_NAV_STATE_CHANGED, handler);
  return () => window.removeEventListener(LEARNING_HUB_NAV_STATE_CHANGED, handler);
}

interface LearningHubNavigationContextValue {
  /** 当前文件夹 ID */
  currentFolderId: string | null;
  /** 当前文件夹路径 */
  currentFolderPath: string;
  /** 面包屑列表（真实路径版） */
  breadcrumbs: RealPathBreadcrumbItem[];
  /** 导航到文件夹（记录历史） */
  navigateTo: (folderId: string | null) => void;
  /** 导航到面包屑位置 */
  navigateToBreadcrumb: (index: number) => void;
  /** 是否可以后退 */
  canGoBack: boolean;
  /** 是否可以前进 */
  canGoForward: boolean;
  /** 是否可以返回父目录 */
  canGoUp: boolean;
  /** 后退 */
  goBack: () => void;
  /** 前进 */
  goForward: () => void;
  /** 返回父目录 */
  goUp: () => void;
  /** 是否在 Learning Hub 页面 */
  isInLearningHub: boolean;
  /** 设置是否在 Learning Hub 页面 */
  setIsInLearningHub: (value: boolean) => void;
}

const LearningHubNavigationContext = createContext<LearningHubNavigationContextValue | null>(null);

export const LearningHubNavigationProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isInLearningHub, setIsInLearningHub] = useState(false);

  const {
    historyIndex,
    history,
    goBack: finderGoBack,
    goForward: finderGoForward,
    goUp: finderGoUp,
    currentPath,
    enterFolder,
    jumpToBreadcrumb,
  } = useFinderStore();

  const breadcrumbs = useMemo<RealPathBreadcrumbItem[]>(
    () => currentPath.breadcrumbs.map((crumb) => ({
      folderId: crumb.id,
      name: crumb.name,
      fullPath: crumb.dstuPath,
    })),
    [currentPath.breadcrumbs]
  );
  const currentFolderPath = breadcrumbs[breadcrumbs.length - 1]?.fullPath || '/';

  // 仅从 finderStore 的历史栈计算导航能力。
  // 移动端“关闭应用 / 返回上级目录”由 LearningHubPage 顶栏箭头单独处理，
  // 全局后退只代表历史后退，避免语义混用。
  const canGoBack = historyIndex > 0;
  const canGoForward = historyIndex < history.length - 1;
  const canGoUp = currentPath.viewKind === 'folder' && currentPath.breadcrumbs.length > 0;

  // goBack/goForward 直接使用 finderStore 的方法
  const goBack = useCallback(() => {
    finderGoBack();
  }, [finderGoBack]);

  const goForward = useCallback(() => {
    finderGoForward();
  }, [finderGoForward]);

  const goUp = useCallback(() => {
    finderGoUp();
  }, [finderGoUp]);

  const navigateTo = useCallback((folderId: string | null) => {
    if (folderId) {
      void enterFolder(folderId);
      return;
    }
    jumpToBreadcrumb(-1);
  }, [enterFolder, jumpToBreadcrumb]);

  const navigateToBreadcrumb = useCallback((index: number) => {
    jumpToBreadcrumb(index);
  }, [jumpToBreadcrumb]);

  // 📱 同步导航状态到全局 ref（供 App.tsx 使用）
  useEffect(() => {
    const state: GlobalNavigationState = {
      canGoBack,
      canGoForward,
      canGoUp,
      goBack,
      goForward,
      goUp,
    };
    finderNavigationRef.current = state;
    publishGlobalLearningHubNavigation();
  }, [canGoBack, canGoForward, canGoUp, goBack, goForward, goUp]);

  const value = useMemo<LearningHubNavigationContextValue>(() => ({
    currentFolderId: currentPath.folderId,
    currentFolderPath,
    breadcrumbs,
    navigateTo,
    navigateToBreadcrumb,
    canGoBack,
    canGoForward,
    canGoUp,
    goBack,
    goForward,
    goUp,
    isInLearningHub,
    setIsInLearningHub,
  }), [
    currentPath.folderId,
    currentFolderPath,
    breadcrumbs,
    navigateTo,
    navigateToBreadcrumb,
    canGoBack,
    canGoForward,
    canGoUp,
    goBack,
    goForward,
    goUp,
    isInLearningHub,
  ]);

  return (
    <LearningHubNavigationContext.Provider value={value}>
      {children}
    </LearningHubNavigationContext.Provider>
  );
};

export const useLearningHubNavigation = (): LearningHubNavigationContextValue => {
  const context = useContext(LearningHubNavigationContext);
  if (!context) {
    throw new Error('useLearningHubNavigation must be used within a LearningHubNavigationProvider');
  }
  return context;
};

/**
 * 安全版本：如果不在 Provider 内则返回默认值
 */
export const useLearningHubNavigationSafe = (): LearningHubNavigationContextValue | null => {
  return useContext(LearningHubNavigationContext);
};

export default LearningHubNavigationContext;
