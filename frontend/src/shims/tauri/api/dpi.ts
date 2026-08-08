// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/dpi（类型占位，仅 dev 工具用）
// ============================================================

export class LogicalSize {
  constructor(
    public width: number,
    public height: number
  ) {}
}

export class PhysicalSize {
  constructor(
    public width: number,
    public height: number
  ) {}
}

export class LogicalPosition {
  constructor(
    public x: number,
    public y: number
  ) {}
}

export class PhysicalPosition {
  constructor(
    public x: number,
    public y: number
  ) {}
}

export function toLogical(size: PhysicalSize, scaleFactor: number): LogicalSize {
  return new LogicalSize(size.width / scaleFactor, size.height / scaleFactor);
}

export function toPhysical(size: LogicalSize, scaleFactor: number): PhysicalSize {
  return new PhysicalSize(size.width * scaleFactor, size.height * scaleFactor);
}

export type Size = LogicalSize | PhysicalSize;
export type Position = LogicalPosition | PhysicalPosition;
