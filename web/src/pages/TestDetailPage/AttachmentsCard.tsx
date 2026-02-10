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
        <p className="text-sm text-gray-600 mt-1">
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
                className="border border-gray-200 rounded-lg p-4 bg-white"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <p className="font-medium text-gray-900 truncate">
                      {attachment.name || "Attachment"}
                    </p>
                    <p className="text-xs text-gray-500 mt-1">
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
                        className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-blue-600 border border-blue-200 rounded-md hover:bg-blue-50 transition-colors"
                      >
                        Open
                      </a>
                    )}
                    {canPreview && (
                      <button
                        type="button"
                        onClick={handleAttachmentClick}
                        className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-gray-900 border border-gray-200 rounded-md hover:bg-gray-50 transition-colors"
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
                      className="max-h-64 rounded-md border border-gray-200 bg-gray-50"
                    />
                  </button>
                )}
                {storageType === "inline" && !isImage && preview && (
                  <div className="mt-3 bg-gray-50 border border-gray-200 rounded-md p-3">
                    <pre className="text-xs text-gray-700 whitespace-pre-wrap wrap-break-word">
                      {preview}
                    </pre>
                    {content.length > previewLimit && (
                      <p className="text-xs text-gray-500 mt-2">
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
