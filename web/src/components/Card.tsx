import { cn } from "../lib/utils";

interface CardContainerProps {
  children: React.ReactNode;
  className?: string;
  style?: React.CSSProperties;
}

interface CardSectionProps {
  children: React.ReactNode;
  className?: string;
}

export function Card({ children, className, style }: CardContainerProps) {
  return (
    <div
      className={cn("rounded-lg bg-(--stitch-surface-card)", className)}
      style={style}
    >
      {children}
    </div>
  );
}

export function CardHeader({ children, className }: CardSectionProps) {
  return <div className={cn("px-6 py-4", className)}>{children}</div>;
}

export function CardTitle({ children, className }: CardSectionProps) {
  return (
    <h3
      className={cn(
        "text-lg font-semibold text-(--stitch-on-surface)",
        className,
      )}
    >
      {children}
    </h3>
  );
}

export function CardContent({ children, className }: CardSectionProps) {
  return <div className={cn("px-6 py-4", className)}>{children}</div>;
}
