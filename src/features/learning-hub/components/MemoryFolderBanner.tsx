/**
 * MemoryFolderBanner - 记忆文件夹专属工具栏
 *
 * ★ 记忆系统改造：当 Finder 导航到记忆根文件夹（或其子文件夹）时显示，
 * 提供不涉及记忆作用域写入/导出的安全入口（自动提取频率、审计日志、树预览）。
 */

import React, { useState, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { cn } from '@/lib/utils';
import {
  ClockCounterClockwise,
  CircleNotch,
  GitBranch,
  List,
} from '@phosphor-icons/react';
import { NotionButton } from '@/components/ui/NotionButton';
import { MemoryIcon } from '../icons/ResourceIcons';
import { showGlobalNotification } from '@/components/UnifiedNotification';
import {
  getMemoryConfig,
  getMemoryAuditLogs,
  setMemoryAutoExtractFrequency,
  type AutoExtractFrequency,
  type MemoryConfig,
  type MemoryAuditLogItem,
} from '@/api/memoryApi';
import { CustomScrollArea } from '@/components/custom-scroll-area';

const AUDIT_LOG_PAGE_SIZE = 30;

interface MemoryFolderBannerProps {
  className?: string;
  isTreeView?: boolean;
  onToggleTreeView?: () => void;
}

export const MemoryFolderBanner: React.FC<MemoryFolderBannerProps> = React.memo(({
  className,
  isTreeView = false,
  onToggleTreeView,
}) => {
  const { t } = useTranslation(['learningHub', 'common']);

  const [config, setConfig] = useState<MemoryConfig | null>(null);

  // 面板状态
  const [showAuditLog, setShowAuditLog] = useState(false);
  const [auditLogs, setAuditLogs] = useState<MemoryAuditLogItem[]>([]);
  const [isLoadingAuditLog, setIsLoadingAuditLog] = useState(false);

  // 加载配置
  useEffect(() => {
    getMemoryConfig().then(setConfig).catch(() => {});
  }, []);

  // 自动提取频率
  const handleFrequencyChange = useCallback(async (freq: AutoExtractFrequency) => {
    if (config?.autoExtractFrequency === freq) return;
    try {
      await setMemoryAutoExtractFrequency(freq);
      const updated = await getMemoryConfig();
      setConfig(updated);
      showGlobalNotification('success', t('memory.frequency_changed', '自动提取频率已更新'));
    } catch {
      showGlobalNotification('error', t('memory.frequency_change_error', '设置失败'));
    }
  }, [config?.autoExtractFrequency, t]);

  // 审计日志
  const handleToggleAuditLog = useCallback(async () => {
    if (showAuditLog) { setShowAuditLog(false); return; }
    setShowAuditLog(true);
    setIsLoadingAuditLog(true);
    try {
      const logs = await getMemoryAuditLogs({ limit: AUDIT_LOG_PAGE_SIZE, offset: 0 });
      setAuditLogs(logs);
    } catch {
      showGlobalNotification('error', t('memory.audit_load_error', '加载操作日志失败'));
    } finally {
      setIsLoadingAuditLog(false);
    }
  }, [showAuditLog, t]);

  if (!config) return null;

  return (
    <div className={cn('border-b border-border/40', className)}>
      {/* 工具栏 */}
      <div className="flex items-center gap-1.5 px-3 py-1.5">
        <MemoryIcon size={14} className="text-muted-foreground shrink-0" />
        <span className="text-[11px] text-muted-foreground mr-1">{t('memory.auto_extract', '自动提取')}:</span>
        <div className="flex items-center gap-0.5">
          {([
            { value: 'off' as const, label: t('memory.freq_off', '关闭') },
            { value: 'balanced' as const, label: t('memory.freq_balanced', '平衡') },
            { value: 'aggressive' as const, label: t('memory.freq_aggressive', '积极') },
          ]).map((opt) => (
            <button
              key={opt.value}
              onClick={() => handleFrequencyChange(opt.value)}
              className={cn(
                'px-1.5 py-0.5 rounded text-[10px] transition-colors',
                config.autoExtractFrequency === opt.value
                  ? 'bg-primary/15 text-primary font-medium'
                  : 'text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground'
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>

        <div className="flex-1" />

        <NotionButton variant="ghost" size="icon" iconOnly
          onClick={handleToggleAuditLog}
          className={cn('!h-6 !w-6', showAuditLog && 'text-primary bg-primary/10')}
          title={t('memory.audit_log', '操作日志')}
        >
          <ClockCounterClockwise size={14} />
        </NotionButton>

        <NotionButton variant="ghost" size="icon" iconOnly
          onClick={onToggleTreeView}
          className={cn('!h-6 !w-6', isTreeView && 'text-primary bg-primary/10')}
          title={isTreeView ? '列表视图' : '树状图预览'}
        >
          {isTreeView ? <List size={14} /> : <GitBranch size={14} />}
        </NotionButton>
      </div>

      {/* 审计日志面板 */}
      {showAuditLog && (
        <div className="border-t border-border/30 px-3 py-2 bg-muted/10">
          <div className="text-[11px] font-medium text-muted-foreground mb-1.5">{t('memory.audit_log', '操作日志')}</div>
          {isLoadingAuditLog ? (
            <div className="flex items-center justify-center py-4"><CircleNotch size={16} className="animate-spin text-muted-foreground" /></div>
          ) : auditLogs.length === 0 ? (
            <div className="text-[11px] text-muted-foreground/60 py-2">{t('memory.audit_empty', '暂无操作日志')}</div>
          ) : (
            <CustomScrollArea className="max-h-48">
              <div className="space-y-1">
                {auditLogs.map((log, i) => (
                  <div key={i} className="flex items-start gap-2 text-[10px] py-0.5">
                    <span className="text-muted-foreground/50 tabular-nums shrink-0">{new Date(log.timestamp).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}</span>
                    <span className={cn('shrink-0', log.success ? 'text-emerald-500' : 'text-rose-500')}>{log.success ? '✓' : '✗'}</span>
                    <span className="text-foreground/70 truncate">{log.title || log.event}</span>
                  </div>
                ))}
              </div>
            </CustomScrollArea>
          )}
        </div>
      )}
    </div>
  );
});

MemoryFolderBanner.displayName = 'MemoryFolderBanner';

export default MemoryFolderBanner;
