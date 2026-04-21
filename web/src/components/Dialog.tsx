import {
  useEffect,
  useId,
  type InputHTMLAttributes,
  type MouseEvent,
  type ReactNode,
} from "react";
import { X } from "lucide-react";
import { Card, CardContent } from "./Card";
import { cn } from "@/lib/utils";

type DialogSize = "sm" | "md" | "lg" | "xl";
type DialogActionVariant = "primary" | "danger" | "secondary";

type DialogInputConfig = {
  id?: string;
  label?: string;
  type?: InputHTMLAttributes<HTMLInputElement>["type"];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  required?: boolean;
  autoFocus?: boolean;
};

interface DialogProps {
  title: string;
  text?: ReactNode;
  description?: ReactNode;
  icon?: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
  input?: DialogInputConfig;
  value?: string;
  disabled?: boolean;
  onChange?: (value: string) => void;
  onConfirm?: () => void;
  onCancel?: () => void;
  onClose?: () => void;
  onSuccessClick?: () => void;
  onCancelClick?: () => void;
  onSuccessButtonContent?: ReactNode;
  cancelButtonContent?: ReactNode;
  showCancelButton?: boolean;
  showCloseButton?: boolean;
  closeOnOverlayClick?: boolean;
  closeOnEscape?: boolean;
  confirmDisabled?: boolean;
  confirmVariant?: DialogActionVariant;
  placeholder?: string;
  size?: DialogSize;
  className?: string;
  bodyClassName?: string;
  footerClassName?: string;
}

export default function Dialog({
  title,
  text,
  description,
  icon,
  children,
  footer,
  input,
  value,
  onChange,
  onConfirm,
  onCancel,
  onClose,
  onSuccessClick,
  onSuccessButtonContent,
  onCancelClick,
  cancelButtonContent,
  showCancelButton = true,
  showCloseButton = false,
  closeOnOverlayClick = true,
  closeOnEscape = true,
  confirmDisabled,
  confirmVariant = "primary",
  placeholder,
  disabled,
  size = "md",
  className,
  bodyClassName,
  footerClassName,
}: DialogProps) {
  const descriptionContent = description ?? text;
  const inputId = useId();
  const resolvedInput =
    input ??
    (typeof onChange === "function"
      ? {
          label: "Marker Value",
          value: value ?? "",
          onChange,
          placeholder,
          disabled,
          required: true,
        }
      : undefined);
  const canConfirm = resolvedInput
    ? resolvedInput.required === false || resolvedInput.value.trim().length > 0
    : true;
  const handleClose = onClose ?? onCancel ?? onCancelClick;
  const handleConfirm = onConfirm ?? onSuccessClick;

  useEffect(() => {
    if (!closeOnEscape) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        handleClose?.();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [closeOnEscape, handleClose]);

  const handleOverlayClick = () => {
    if (closeOnOverlayClick) {
      handleClose?.();
    }
  };

  const sizeClasses: Record<DialogSize, string> = {
    sm: "max-w-sm",
    md: "max-w-md",
    lg: "max-w-2xl",
    xl: "max-w-4xl",
  };

  const confirmButtonClasses: Record<DialogActionVariant, string> = {
    primary:
      "text-(--stitch-on-primary) hover:brightness-105 focus-visible:ring-(--stitch-primary)",
    danger:
      "bg-(--status-failure) text-white hover:brightness-105 focus-visible:ring-(--status-failure)",
    secondary:
      "bg-(--stitch-surface-low) text-(--stitch-on-surface) hover:bg-(--stitch-surface-card) focus-visible:ring-(--stitch-primary)",
  };

  const confirmButtonStyle =
    confirmVariant === "primary"
      ? {
          backgroundImage:
            "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-strong))",
        }
      : undefined;
  const hasFooter = Boolean(
    footer ?? handleConfirm ?? (showCancelButton && handleClose),
  );

  const stopPropagation = (event: MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4 backdrop-blur-sm"
      onClick={handleOverlayClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby={`${inputId}-title`}
      aria-describedby={
        descriptionContent ? `${inputId}-description` : undefined
      }
    >
      <div onClick={stopPropagation}>
        <Card
          className={cn(
            "relative w-full overflow-hidden rounded-md bg-(--stitch-surface-card)",
            sizeClasses[size],
            className,
          )}
        >
          <CardContent className="p-6">
            {showCloseButton && (
              <button
                type="button"
                onClick={handleClose}
                className="absolute right-4 top-4 inline-flex h-9 w-9 items-center justify-center rounded-md text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-low) hover:text-(--stitch-on-surface) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2"
                aria-label="Close dialog"
                disabled={disabled}
              >
                <X className="h-4 w-4" />
              </button>
            )}

            <div className="space-y-5">
              <div className="flex items-start gap-4">
                {icon ? (
                  <div className="mt-0.5 shrink-0 rounded-md bg-(--stitch-surface-low) p-2 text-(--stitch-primary)">
                    {icon}
                  </div>
                ) : null}
                <div className="min-w-0 flex-1">
                  <h3
                    id={`${inputId}-title`}
                    className="text-lg font-semibold text-(--stitch-on-surface)"
                  >
                    {title}
                  </h3>
                  {descriptionContent ? (
                    <div
                      id={`${inputId}-description`}
                      className="mt-2 text-sm leading-6 text-(--stitch-on-surface-muted)"
                    >
                      {descriptionContent}
                    </div>
                  ) : null}
                </div>
              </div>

              <div className={cn("space-y-4", bodyClassName)}>
                {resolvedInput ? (
                  <div className="space-y-2">
                    <label
                      htmlFor={resolvedInput.id ?? `${inputId}-input`}
                      className="block text-sm font-medium text-(--stitch-on-surface)"
                    >
                      {resolvedInput.label ?? "Value"}
                    </label>
                    <input
                      id={resolvedInput.id ?? `${inputId}-input`}
                      type={resolvedInput.type ?? "text"}
                      value={resolvedInput.value}
                      onChange={(event) =>
                        resolvedInput.onChange(event.target.value)
                      }
                      placeholder={resolvedInput.placeholder}
                      className="w-full rounded-md border border-(--stitch-outline) bg-(--stitch-surface) px-3 py-2 text-(--stitch-on-surface) outline-none transition-colors focus:ring-2 focus:ring-(--stitch-primary)"
                      disabled={disabled || resolvedInput.disabled}
                      autoFocus={resolvedInput.autoFocus}
                    />
                  </div>
                ) : null}

                {children}
              </div>

              {hasFooter ? (
                <div
                  className={cn(
                    "flex flex-wrap items-center justify-end gap-3",
                    footerClassName,
                  )}
                >
                  {footer ?? (
                    <>
                      {showCancelButton && handleClose ? (
                        <button
                          type="button"
                          onClick={handleClose}
                          className="rounded-md border border-(--stitch-outline) px-4 py-2 text-(--stitch-on-surface) transition-colors hover:bg-(--stitch-surface-low) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2"
                          disabled={disabled}
                        >
                          {cancelButtonContent ?? "Cancel"}
                        </button>
                      ) : null}

                      {handleConfirm ? (
                        <button
                          type="button"
                          onClick={handleConfirm}
                          className={cn(
                            "flex items-center gap-2 rounded-md px-4 py-2 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
                            confirmButtonClasses[confirmVariant],
                          )}
                          style={confirmButtonStyle}
                          disabled={disabled || confirmDisabled || !canConfirm}
                        >
                          {onSuccessButtonContent || "Confirm"}
                        </button>
                      ) : null}
                    </>
                  )}
                </div>
              ) : null}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
