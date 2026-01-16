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
      <Tag className="h-4 w-4 text-gray-400" />
      {tags.map((tag, index) => (
        <span
          key={index}
          className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 border border-blue-200"
        >
          {tag}
        </span>
      ))}
    </div>
  );
};
