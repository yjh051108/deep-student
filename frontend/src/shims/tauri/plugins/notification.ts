// Tauri → Wails 适配层：@tauri-apps/plugin-notification
// Web Notification API 等价实现
export async function isPermissionGranted(): Promise<boolean> {
  if (!('Notification' in window)) return false;
  return Notification.permission === 'granted';
}
export async function requestPermission(): Promise<boolean> {
  if (!('Notification' in window)) return false;
  const result = await Notification.requestPermission();
  return result === 'granted';
}
export async function sendNotification(options: { title?: string; body?: string; icon?: string }): Promise<void> {
  if (!('Notification' in window)) return;
  if (Notification.permission !== 'granted') return;
  try {
    new Notification(options.title ?? 'Deep Student', {
      body: options.body,
      icon: options.icon,
    });
  } catch {
    // 忽略：WebView 可能不支持
  }
}
export async function removeActiveNotification(): Promise<void> {}
