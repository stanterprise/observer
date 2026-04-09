import { Tag } from "lucide-react";

type TagListProps = {
  tags?: string[];
  className?: string;
};

export const TagList = ({ tags, className = "" }: TagListProps) => {
  if (!tags || tags.length === 0) {
    return null;
  }

  return (
    <div className={`flex flex-wrap items-center gap-2 ${className}`}>
      <Tag className="h-4 w-4 text-(--stitch-on-surface-subtle)" />
      {tags.map((tag, index) => (
        <span
          key={index}
          className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium"
          style={{
            backgroundColor: "var(--stitch-primary-soft)",
            borderColor: "var(--status-running-border)",
            color: "var(--stitch-primary)",
          }}
        >
          {tag}
        </span>
      ))}
    </div>
  );
};
