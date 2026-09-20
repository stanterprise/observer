import { useEffect, useRef, useState, type DragEvent } from "react";
import { AlertTriangle, FileArchive, Upload } from "lucide-react";
import Dialog from "@/components/Dialog";
import { apiUrl } from "@/lib/config";

type ImportCapability = {
  producer: string;
  format: string;
  reportVersion: string;
  uploadPath: string;
  fileExtensions: string[];
  multipleFiles: boolean;
  maxFiles: number;
  maxRequestBytes: number;
  externalAttachments: boolean;
};

type ImportResult = {
  runId: string;
  created: boolean;
  warnings: Array<{ code: string; message: string; file?: string }>;
};

export function ImportRunDialog({
  onClose,
  onImported,
}: {
  onClose: () => void;
  onImported: (result: ImportResult) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [capabilities, setCapabilities] = useState<ImportCapability[]>([]);
  const [selectedPath, setSelectedPath] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [name, setName] = useState("");
  const [loadingCapabilities, setLoadingCapabilities] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    void (async () => {
      try {
        const response = await fetch(apiUrl("/v1/imports"), {
          signal: controller.signal,
        });
        if (!response.ok) throw new Error("Report import options are unavailable");
        const payload = (await response.json()) as { data: ImportCapability[] };
        setCapabilities(payload.data);
        setSelectedPath(payload.data[0]?.uploadPath ?? "");
      } catch (fetchError) {
        if (!(fetchError instanceof DOMException && fetchError.name === "AbortError")) {
          setError(fetchError instanceof Error ? fetchError.message : "Failed to load import options");
        }
      } finally {
        setLoadingCapabilities(false);
      }
    })();
    return () => controller.abort();
  }, []);

  const capability = capabilities.find((item) => item.uploadPath === selectedPath);

  const selectFiles = (selected: File[]) => {
    if (!capability) return;
    const extensions = capability.fileExtensions.map((value) => value.toLowerCase());
    const invalid = selected.find(
      (file) => !extensions.some((extension) => file.name.toLowerCase().endsWith(extension)),
    );
    if (invalid) {
      setError(`${invalid.name} is not a supported ${capability.producer} ${capability.format} report.`);
      return;
    }
    if (selected.length > capability.maxFiles) {
      setError(`Select at most ${capability.maxFiles} report files.`);
      return;
    }
    const total = selected.reduce((sum, file) => sum + file.size, 0);
    if (total > capability.maxRequestBytes) {
      setError(`The selected reports exceed the ${formatBytes(capability.maxRequestBytes)} upload limit.`);
      return;
    }
    setFiles(selected);
    setError(null);
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!uploading) selectFiles(Array.from(event.dataTransfer.files));
  };

  const handleImport = async () => {
    if (!capability || files.length === 0) return;
    setUploading(true);
    setError(null);
    try {
      const form = new FormData();
      if (name.trim()) form.append("name", name.trim());
      files.forEach((file) => form.append("reports", file, file.name));
      const response = await fetch(apiUrl(capability.uploadPath), { method: "POST", body: form });
      const payload = (await response.json().catch(() => null)) as
        | { data?: ImportResult; error?: { message?: string } }
        | null;
      if (!response.ok || !payload?.data) {
        throw new Error(payload?.error?.message || `Import failed (${response.status})`);
      }
      onImported(payload.data);
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : "Import failed");
    } finally {
      setUploading(false);
    }
  };

  return (
    <Dialog
      title="Import test run"
      description="Upload a completed test report. All selected Playwright shards are combined into one Observer run."
      icon={<Upload className="h-5 w-5" />}
      size="lg"
      onCancel={onClose}
      onConfirm={handleImport}
      onSuccessButtonContent={uploading ? "Importing…" : "Import run"}
      confirmDisabled={loadingCapabilities || files.length === 0}
      disabled={uploading}
      closeOnOverlayClick={!uploading}
      closeOnEscape={!uploading}
      showCloseButton
    >
      <div className="space-y-2">
        <label className="block text-sm font-medium text-(--stitch-on-surface)" htmlFor="report-type">
          Report type
        </label>
        <select
          id="report-type"
          value={selectedPath}
          onChange={(event) => {
            setSelectedPath(event.target.value);
            setFiles([]);
          }}
          disabled={loadingCapabilities || uploading}
          className="w-full rounded-md border border-(--stitch-outline) bg-(--stitch-surface) px-3 py-2 text-(--stitch-on-surface)"
        >
          {capabilities.map((item) => (
            <option key={item.uploadPath} value={item.uploadPath}>
              {item.producer} {item.format} ({item.reportVersion})
            </option>
          ))}
        </select>
      </div>

      <div className="space-y-2">
        <label className="block text-sm font-medium text-(--stitch-on-surface)" htmlFor="import-name">
          Run name <span className="font-normal text-(--stitch-on-surface-muted)">(optional)</span>
        </label>
        <input
          id="import-name"
          value={name}
          maxLength={256}
          onChange={(event) => setName(event.target.value)}
          disabled={uploading}
          placeholder="Uses the report name when left blank"
          className="w-full rounded-md border border-(--stitch-outline) bg-(--stitch-surface) px-3 py-2 text-(--stitch-on-surface)"
        />
      </div>

      <div
        onDragOver={(event) => event.preventDefault()}
        onDrop={handleDrop}
        className="rounded-md border-2 border-dashed border-(--stitch-outline) bg-(--stitch-surface-low) p-6 text-center"
      >
        <FileArchive className="mx-auto h-8 w-8 text-(--stitch-primary)" />
        <p className="mt-3 text-sm text-(--stitch-on-surface)">
          Drop {capability?.multipleFiles ? "one or more reports" : "a report"} here, or
        </p>
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={!capability || uploading}
          className="mt-3 rounded-md border border-(--stitch-outline) bg-(--stitch-surface-card) px-4 py-2 text-sm font-medium text-(--stitch-on-surface) hover:bg-(--stitch-surface)"
        >
          Choose files
        </button>
        <input
          ref={inputRef}
          className="hidden"
          type="file"
          accept={capability?.fileExtensions.join(",")}
          multiple={capability?.multipleFiles}
          onChange={(event) => selectFiles(Array.from(event.target.files ?? []))}
        />
        {capability && (
          <p className="mt-3 text-xs text-(--stitch-on-surface-muted)">
            {capability.fileExtensions.join(", ")} · up to {capability.maxFiles} files · {formatBytes(capability.maxRequestBytes)} total
          </p>
        )}
      </div>

      {files.length > 0 && (
        <div className="rounded-md border border-(--stitch-outline) p-3 text-sm text-(--stitch-on-surface)">
          {files.length} file{files.length === 1 ? "" : "s"} selected ({formatBytes(files.reduce((sum, file) => sum + file.size, 0))})
        </div>
      )}
      {capability && !capability.externalAttachments && (
        <div className="flex gap-2 rounded-md bg-(--status-warning-soft) p-3 text-sm text-(--stitch-on-surface)">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          Large report attachments require an attachment storage driver on the server.
        </div>
      )}
      {error && <div className="rounded-md bg-(--status-failure-soft) p-3 text-sm text-(--status-failure)">{error}</div>}
    </Dialog>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB"];
  let value = bytes / 1024;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) { value /= 1024; index += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[index]}`;
}
