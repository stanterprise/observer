import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import {
  getAttachmentUrl,
  decodeInlineContent,
  getInlineMediaUrl,
  formatBytes,
} from "./utils";

type AttachmentsCardProps = {
  attachments: Array<Record<string, any>>;
  setActiveAttachment: (attachment: any) => void;
};

export default function AttachmentsCard({
  attachments,
  setActiveAttachment,
}: AttachmentsCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Attachments</CardTitle>
        <p className="text-sm text-(--stitch-on-surface-muted) mt-1">
          {attachments.length} attachment
          {attachments.length > 1 ? "s" : ""} associated with this test
        </p>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {attachments.map((attachment, index) => {
            const url = getAttachmentUrl(attachment);
            const storageType = attachment.storage || "inline";
            const content = decodeInlineContent(attachment);
            const previewLimit = 400;
            const preview = content ? content.slice(0, previewLimit) : "";
            const isImage =
              typeof attachment.mime_type === "string" &&
              attachment.mime_type.startsWith("image/");
            const isVideo =
              typeof attachment.mime_type === "string" &&
              attachment.mime_type.startsWith("video/");
            const isAudio =
              typeof attachment.mime_type === "string" &&
              attachment.mime_type.startsWith("audio/");
            const inlineMediaUrl = getInlineMediaUrl(attachment);
            const mediaUrl = url || inlineMediaUrl;
            const canPreview = isImage || isVideo || isAudio || content;
            const handleAttachmentClick = () => {
              if (!canPreview) return;
              setActiveAttachment({
                attachment,
                url,
                inlineUrl: inlineMediaUrl,
                isImage,
                isVideo,
                isAudio,
                contentText: content,
              });
            };

            return (
              <div
                key={`${attachment.storage_key || attachment.uri || attachment.name || "attachment"}-${index}`}
                className="border border-(--stitch-outline) rounded-lg p-4 bg-(--stitch-surface-card)"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <p className="font-medium text-(--stitch-on-surface) truncate">
                      {attachment.name || "Attachment"}
                    </p>
                    <p className="text-xs text-(--stitch-on-surface-subtle) mt-1">
                      {attachment.mime_type || "unknown"} •{" "}
                      {formatBytes(attachment.size)} • {storageType}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    {url && (
                      <a
                        href={url}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-(--stitch-primary) border border-(--status-running-border) rounded-md hover:bg-(--stitch-primary-soft) transition-colors"
                      >
                        Open
                      </a>
                    )}
                    {canPreview && (
                      <button
                        type="button"
                        onClick={handleAttachmentClick}
                        className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-(--stitch-on-surface) border border-(--stitch-outline) rounded-md hover:bg-(--stitch-surface-low) transition-colors"
                      >
                        View
                      </button>
                    )}
                  </div>
                </div>
                {isImage && mediaUrl && (
                  <button
                    type="button"
                    onClick={handleAttachmentClick}
                    className="mt-3 block"
                  >
                    <img
                      src={mediaUrl}
                      alt={attachment.name || "Attachment"}
                      className="max-h-64 rounded-md border border-(--stitch-outline) bg-(--stitch-surface-low)"
                    />
                  </button>
                )}
                {storageType === "inline" && !isImage && preview && (
                  <div className="mt-3 bg-(--stitch-surface-low) border border-(--stitch-outline) rounded-md p-3">
                    <pre className="text-xs text-(--stitch-on-surface-muted) whitespace-pre-wrap wrap-break-word">
                      {preview}
                    </pre>
                    {content.length > previewLimit && (
                      <p className="text-xs text-(--stitch-on-surface-subtle) mt-2">
                        Preview truncated
                      </p>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
