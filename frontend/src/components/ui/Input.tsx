// shadcn/ui Input 组件
// 对照原版 src/components/ui/shad/Input.tsx

import * as React from "react";
import { cn } from "@/lib/utils";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        ref={ref}
        className={cn(
          // 基础类：flex h-10 w-full + 圆角 + 边框 + 输入色 + 占位符 + focus 环 + 禁用态
          "flex h-[var(--touch-target-size)] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm text-foreground transition-colors",
          "placeholder:text-muted-foreground",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-0",
          "disabled:cursor-not-allowed disabled:opacity-50",
          "file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground",
          className
        )}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";

export { Input };
