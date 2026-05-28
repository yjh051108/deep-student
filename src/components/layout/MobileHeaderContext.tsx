/**
 * MobileHeaderContext - 移动端顶部栏状态管理
 *
 * 提供统一的移动端顶栏配置管理：
 * - 各页面通过 useMobileHeader hook 设置自己的 header 配置
 * - App.tsx 级别的 UnifiedMobileHeader 从 context 读取配置并渲染
 * - ★ 支持视图级别的配置隔离，只有活跃视图的配置才会生效
 */

import React, { createContext, useContext, useState, useCallback, useLayoutEffect, useMemo, useRef, type ReactNode } from 'react';

/** 移动端顶栏配置 */
export interface MobileHeaderConfig {
  /** 是否暂时隐藏整个移动端顶栏 */
  hidden?: boolean;
  /** 标题（字符串形式） */
  title?: string;
  /** 自定义标题节点（优先级高于 title，用于面包屑等复杂渲染） */
  titleNode?: ReactNode;
  /** 副标题 */
  subtitle?: string;
  /** 右侧操作区域 */
  rightActions?: ReactNode;
  /** 是否显示菜单按钮（用于打开次级侧边栏） */
  showMenu?: boolean;
  /** 点击菜单按钮的回调 */
  onMenuClick?: () => void;
  /** 是否显示返回箭头（替代菜单图标） */
  showBackArrow?: boolean;
  /** 隐藏全局回退按钮（仅当页面不希望显示任何左侧导航按钮时使用） */
  suppressGlobalBackButton?: boolean;
}

/** Context 值类型 */
interface MobileHeaderContextValue {
  /** 当前配置 */
  config: MobileHeaderConfig;
  /** 设置配置（带视图 ID） */
  setConfig: (viewId: string, config: MobileHeaderConfig) => void;
  /** 重置配置 */
  resetConfig: () => void;
  /** 设置活跃视图（由 App.tsx 调用） */
  setActiveView: (viewId: string) => void;
}

const defaultConfig: MobileHeaderConfig = {
  hidden: false,
  title: '',
  subtitle: undefined,
  rightActions: undefined,
  showMenu: false,
  onMenuClick: undefined,
};

const MobileHeaderContext = createContext<MobileHeaderContextValue | null>(null);

function getReactNodeSignature(node: ReactNode): string {
  if (node == null || typeof node === 'boolean') {
    return '';
  }
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node);
  }
  if (Array.isArray(node)) {
    return node.map(getReactNodeSignature).join('|');
  }
  if (React.isValidElement(node)) {
    const typeName = typeof node.type === 'string'
      ? node.type
      : ((node.type as { displayName?: string; name?: string }).displayName
        || (node.type as { displayName?: string; name?: string }).name
        || 'Component');
    const props = node.props as Record<string, unknown>;
    const propSignature = Object.keys(props)
      .filter((key) => key !== 'children' && typeof props[key] !== 'function')
      .sort()
      .map((key) => `${key}:${String(props[key])}`)
      .join(',');
    return `${typeName}:${node.key ?? ''}{${propSignature}}(${getReactNodeSignature(props.children as ReactNode)})`;
  }
  return typeof node;
}

function areHeaderConfigsEqual(a: MobileHeaderConfig, b: MobileHeaderConfig): boolean {
  return Object.is(a.hidden, b.hidden)
    && Object.is(a.title, b.title)
    && getReactNodeSignature(a.titleNode) === getReactNodeSignature(b.titleNode)
    && Object.is(a.subtitle, b.subtitle)
    && getReactNodeSignature(a.rightActions) === getReactNodeSignature(b.rightActions)
    && Object.is(a.showMenu, b.showMenu)
    && Object.is(a.showBackArrow, b.showBackArrow)
    && Object.is(a.suppressGlobalBackButton, b.suppressGlobalBackButton);
}

/** Provider 组件 */
export const MobileHeaderProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  // 当前显示的配置
  const [config, setConfigState] = useState<MobileHeaderConfig>(defaultConfig);
  // 当前活跃视图
  const activeViewRef = useRef<string>('');
  // 各视图的配置缓存
  const configCacheRef = useRef<Map<string, MobileHeaderConfig>>(new Map());

  // 设置配置（带视图 ID）
  const setConfig = useCallback((viewId: string, newConfig: MobileHeaderConfig) => {
    // 缓存该视图的配置
    const cachedConfig = configCacheRef.current.get(viewId);
    if (!cachedConfig || !areHeaderConfigsEqual(cachedConfig, newConfig)) {
      configCacheRef.current.set(viewId, newConfig);
    }
    // 只有当前活跃视图才立即应用配置
    if (activeViewRef.current === viewId) {
      setConfigState((prev) => (
        areHeaderConfigsEqual(prev, newConfig) ? prev : newConfig
      ));
    }
  }, []);

  // 设置活跃视图
  const setActiveView = useCallback((viewId: string) => {
    if (activeViewRef.current === viewId) {
      return;
    }
    activeViewRef.current = viewId;
    // 应用该视图缓存的配置
    const cachedConfig = configCacheRef.current.get(viewId);
    if (cachedConfig) {
      setConfigState((prev) => (
        areHeaderConfigsEqual(prev, cachedConfig) ? prev : cachedConfig
      ));
    } else {
      // 如果没有缓存（懒加载组件还没加载），先显示空配置，页面加载后会更新
      setConfigState((prev) => (
        areHeaderConfigsEqual(prev, defaultConfig) ? prev : defaultConfig
      ));
    }
  }, []);

  const resetConfig = useCallback(() => {
    setConfigState((prev) => (
      areHeaderConfigsEqual(prev, defaultConfig) ? prev : defaultConfig
    ));
  }, []);

  const contextValue = useMemo<MobileHeaderContextValue>(() => ({
    config,
    setConfig,
    resetConfig,
    setActiveView,
  }), [config, setConfig, resetConfig, setActiveView]);

  return (
    <MobileHeaderContext.Provider value={contextValue}>
      {children}
    </MobileHeaderContext.Provider>
  );
};

/** 获取 Context（内部使用） */
export const useMobileHeaderContext = (): MobileHeaderContextValue => {
  const ctx = useContext(MobileHeaderContext);
  if (!ctx) {
    throw new Error('useMobileHeaderContext must be used within MobileHeaderProvider');
  }
  return ctx;
};

/** 安全版本，在非 Provider 内返回 null */
export const useMobileHeaderContextSafe = (): MobileHeaderContextValue | null => {
  return useContext(MobileHeaderContext);
};

/**
 * useMobileHeader - 页面级别设置移动端顶栏配置
 *
 * @param viewId - 视图 ID（如 'learning-hub', 'chat-v2'），用于配置隔离
 * @param config - 顶栏配置
 * @param deps - 依赖数组，当依赖变化时更新配置
 *
 * @example
 * ```tsx
 * // 基础用法
 * useMobileHeader('settings', { title: '设置' });
 *
 * // 带右侧操作
 * useMobileHeader('learning-hub', {
 *   title: '学习资源',
 *   rightActions: <Button>刷新</Button>
 * }, []);
 *
 * // 动态标题
 * useMobileHeader('chat-v2', {
 *   title: currentSession?.title || '聊天',
 * }, [currentSession?.title]);
 * ```
 */
export function useMobileHeader(viewId: string, config: MobileHeaderConfig, deps: React.DependencyList = []): void {
  const ctx = useContext(MobileHeaderContext);
  const configRef = useRef(config);
  configRef.current = config;
  const viewIdRef = useRef(viewId);
  viewIdRef.current = viewId;

  // 使用 useLayoutEffect 同步更新配置
  useLayoutEffect(() => {
    if (ctx) {
      ctx.setConfig(viewIdRef.current, configRef.current);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  // 首次挂载时立即设置配置
  useLayoutEffect(() => {
    if (ctx) {
      ctx.setConfig(viewIdRef.current, configRef.current);
    }
  }, []); // 仅首次挂载时执行
}

/**
 * useSetMobileHeaderActiveView - 供 App.tsx 调用，设置当前活跃视图
 */
export function useSetMobileHeaderActiveView(): (viewId: string) => void {
  const ctx = useContext(MobileHeaderContext);
  return useCallback((viewId: string) => {
    ctx?.setActiveView(viewId);
  }, [ctx]);
}

/**
 * MobileHeaderActiveViewSync - 在 MobileHeaderProvider 内部同步 activeView
 *
 * 必须放在 MobileHeaderProvider 内部使用，因为需要访问 Context
 */
export const MobileHeaderActiveViewSync: React.FC<{ activeView: string }> = ({ activeView }) => {
  const ctx = useContext(MobileHeaderContext);
  const setActiveView = ctx?.setActiveView;

  useLayoutEffect(() => {
    if (setActiveView) {
      setActiveView(activeView);
    }
  }, [activeView, setActiveView]);

  return null;
};

export default MobileHeaderProvider;
