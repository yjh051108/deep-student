// shadcn/ui Button 组件
// 对照原版 src/components/ui/shad/Button.tsx
// 简化版：不依赖 buttonPrimitiveContract，直接使用 cva + 标准 Tailwind 类

import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  // 基础类：inline-flex + 圆角 + 边框 + 字号 + 过渡 + focus 环 + 禁用态
  "inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-[var(--button-radius)] border text-[13px] font-medium leading-none transition-[background-color,border-color,color] duration-150 ease-out outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-0 disabled:pointer-events-none disabled:opacity-50 select-none motion-reduce:transition-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg]:text-inherit",
  {
    variants: {
      variant: {
        // 主按钮：实心 primary
        default:
          "border-transparent bg-primary text-primary-foreground hover:bg-[var(--button-prominent-hover-bg)] active:bg-[var(--button-prominent-active-bg)]",
        // 销毁按钮：destructive 实心
        destructive:
          "border-transparent bg-destructive text-destructive-foreground hover:bg-[var(--button-destructive-hover-bg)] active:bg-[var(--button-destructive-active-bg)]",
        // 描边按钮
        outline:
          "border-[var(--button-outline-border)] bg-[var(--button-outline-bg)] text-foreground hover:bg-[var(--button-outline-hover-bg)] hover:text-foreground active:bg-[var(--button-outline-active-bg)]",
        // 次要按钮：accent 实心
        secondary:
          "border-transparent bg-[var(--button-tonal-bg)] text-foreground hover:bg-[var(--button-tonal-hover-bg)] active:bg-[var(--button-tonal-active-bg)]",
        // 幽灵按钮：透明
        ghost:
          "border-transparent bg-transparent text-muted-foreground hover:bg-[var(--button-plain-hover-bg)] hover:text-foreground active:bg-[var(--button-plain-active-bg)]",
        // 链接按钮
        link:
          "border-transparent bg-transparent text-primary underline-offset-4 hover:underline",
      },
      size: {
        // 默认高度：触摸目标 / 桌面端按钮高度
        default:
          "h-[var(--touch-target-size)] px-[var(--button-padding-x)] text-[13px] lg:h-[var(--button-height)]",
        sm: "h-[var(--touch-target-size)] px-[var(--button-padding-x-sm)] text-xs lg:h-[var(--button-height-sm)]",
        lg: "h-[var(--touch-target-size)] px-[var(--button-padding-x-lg)] text-sm lg:h-[var(--button-height-lg)]",
        icon:
          "h-[var(--touch-target-size)] w-[var(--touch-target-size)] rounded-[var(--button-radius)] p-0 lg:h-[var(--button-icon-size)] lg:w-[var(--button-icon-size)]",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        data-shad-button=""
        data-size={size ?? "default"}
        className={cn(buttonVariants({ variant, size }), className)}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";

export { Button, buttonVariants };
