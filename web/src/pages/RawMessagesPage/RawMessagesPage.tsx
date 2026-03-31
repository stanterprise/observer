import { useEffect, useState, useCallback, useMemo } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import {
  ArrowLeft,
  Download,
  FileText,
  Search,
  X,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { cn } from "@/lib/utils";
import type {
  RawMessagesRunDocument,
  RetainedMessage,
} from "@/types/rawMessages";

// ─── Event-type badge colours ────────────────────────────────────────────────

const EVENT_TYPE_COLORS: Record<string, string> = {
  "run.start": "bg-blue-100 text-blue-800 border-blue-200",
  "run.end": "bg-blue-100 text-blue-800 border-blue-200",
  "suite.begin": "bg-purple-100 text-purple-800 border-purple-200",
  "suite.end": "bg-purple-100 text-purple-800 border-purple-200",
  "test.begin": "bg-green-100 text-green-800 border-green-200",
  "test.end": "bg-green-100 text-green-800 border-green-200",
  "test.failure": "bg-red-100 text-red-800 border-red-200",
  "test.error": "bg-red-100 text-red-800 border-red-200",
  "step.begin": "bg-yellow-100 text-yellow-800 border-yellow-200",
  "step.end": "bg-yellow-100 text-yellow-800 border-yellow-200",
  stdout: "bg-gray-100 text-gray-800 border-gray-200",
  stderr: "bg-orange-100 text-orange-800 border-orange-200",
  heartbeat: "bg-teal-100 text-teal-800 border-teal-200",
};

function eventTypeColor(eventType: string): string {
  return (
    EVENT_TYPE_COLORS[eventType] ?? "bg-gray-100 text-gray-700 border-gray-200"
  );
}

// ─── Collapsible JSON payload viewer ─────────────────────────────────────────

function PayloadViewer({ payload }: { payload: unknown }) {
  const [expanded, setExpanded] = useState(false);

  const formatted = useMemo(() => {
    try {
      return JSON.stringify(payload, null, 2);
    } catch {
      return String(payload);
    }
  }, [payload]);

  if (payload === null || payload === undefined) {
    return <span className="text-gray-400 italic text-xs">empty</span>;
  }

  return (
    <div>
      <button
        onClick={() => setExpanded((v) => !v)}
        className="inline-flex items-center gap-1 text-xs font-medium text-blue-600 hover:text-blue-800 transition-colors"
        aria-expanded={expanded}
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5" />
        )}
        {expanded ? "Collapse" : "Show payload"}
      </button>
      {expanded && (
        <pre className="mt-2 p-3 rounded-md bg-gray-900 text-gray-100 text-xs overflow-x-auto whitespace-pre-wrap break-all max-h-96 leading-relaxed">
          {formatted}
        </pre>
      )}
    </div>
  );
}

// ─── Message row ─────────────────────────────────────────────────────────────

function MessageRow({ msg, index }: { msg: RetainedMessage; index: number }) {
  const receivedAt = useMemo(() => {
    try {
      return new Date(msg.receivedAt).toLocaleString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        fractionalSecondDigits: 3,
      });
    } catch {
      return msg.receivedAt;
    }
  }, [msg.receivedAt]);

  return (
    <div className="border border-gray-200 rounded-lg bg-white hover:border-gray-300 transition-colors">
      <div className="p-4">
        {/* Header row */}
        <div className="flex flex-wrap items-start gap-3 mb-3">
          <span className="text-xs text-gray-400 font-mono w-8 shrink-0 pt-0.5">
            #{index + 1}
          </span>
          <span
            className={cn(
              "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold border shrink-0",
              eventTypeColor(msg.eventType),
            )}
          >
            {msg.eventType}
          </span>
          <span className="text-xs text-gray-500 font-mono truncate flex-1 min-w-0">
            {msg.subject}
          </span>
          <div className="flex items-center gap-3 shrink-0">
            {msg.sequence !== undefined && msg.sequence > 0 && (
              <span className="text-xs text-gray-400">
                seq&nbsp;{msg.sequence}
              </span>
            )}
            <span className="text-xs text-gray-500">{receivedAt}</span>
          </div>
        </div>
        {/* Payload */}
        <div className="pl-11">
          <PayloadViewer payload={msg.payload} />
        </div>
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export function RawMessagesPage() {
  const { runId } = useParams<{ runId: string }>();

  const [doc, setDoc] = useState<RawMessagesRunDocument | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notEnabled, setNotEnabled] = useState(false);

  // Filter state
  const [searchText, setSearchText] = useState("");
  const [selectedEventTypes, setSelectedEventTypes] = useState<Set<string>>(
    new Set(),
  );

  const fetchMessages = useCallback(async (id: string) => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl(`/runs/${id}/raw-messages`));
      if (response.status === 404) {
        const body = await response.text();
        if (body.includes("retention")) {
          setNotEnabled(true);
        } else {
          setError("No retained messages found for this run.");
        }
        return;
      }
      if (!response.ok) {
        throw new Error(`Failed to fetch raw messages: ${response.statusText}`);
      }
      const data: RawMessagesRunDocument = await response.json();
      setDoc(data);
      setError(null);
      setNotEnabled(false);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch raw messages",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (runId) {
      fetchMessages(runId);
    }
  }, [runId, fetchMessages]);

  // All event types present in the document
  const availableEventTypes = useMemo(() => {
    if (!doc) return [];
    const types = new Set<string>();
    doc.messages.forEach((m) => {
      if (m.eventType) types.add(m.eventType);
    });
    return Array.from(types).sort();
  }, [doc]);

  // Filtered messages with their original indices preserved for display
  const filteredMessages = useMemo(() => {
    if (!doc)
      return [] as Array<{ msg: RetainedMessage; originalIndex: number }>;
    return doc.messages.reduce<
      Array<{ msg: RetainedMessage; originalIndex: number }>
    >((acc, msg, i) => {
      if (
        selectedEventTypes.size > 0 &&
        !selectedEventTypes.has(msg.eventType)
      ) {
        return acc;
      }
      if (searchText.trim() !== "") {
        const q = searchText.toLowerCase();
        const inType = msg.eventType.toLowerCase().includes(q);
        const inSubject = msg.subject.toLowerCase().includes(q);
        const inPayload = JSON.stringify(msg.payload).toLowerCase().includes(q);
        if (!inType && !inSubject && !inPayload) return acc;
      }
      acc.push({ msg, originalIndex: i });
      return acc;
    }, []);
  }, [doc, selectedEventTypes, searchText]);

  // Message counts by event type (for unfiltered totals in filter pills)
  const countsByEventType = useMemo(() => {
    if (!doc) return {} as Record<string, number>;
    return doc.messages.reduce<Record<string, number>>((acc, msg) => {
      acc[msg.eventType] = (acc[msg.eventType] ?? 0) + 1;
      return acc;
    }, {});
  }, [doc]);

  const toggleEventType = (eventType: string) => {
    setSelectedEventTypes((prev) => {
      const next = new Set(prev);
      if (next.has(eventType)) {
        next.delete(eventType);
      } else {
        next.add(eventType);
      }
      return next;
    });
  };

  const clearFilters = () => {
    setSearchText("");
    setSelectedEventTypes(new Set());
  };

  const exportJsonLines = () => {
    if (!doc || !runId) return;

    const lines = filteredMessages.map(({ msg }) => JSON.stringify(msg));
    const content = lines.join("\n") + (lines.length > 0 ? "\n" : "");
    const blob = new Blob([content], {
      type: "application/x-ndjson;charset=utf-8",
    });

    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    const filename = `raw-messages-${runId}-${timestamp}.jsonl`;

    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const hasActiveFilters =
    searchText.trim() !== "" || selectedEventTypes.size > 0;

  // ── Loading skeleton ──────────────────────────────────────────────────────
  if (loading) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <div className="flex items-center gap-4">
          <div className="h-10 w-10 bg-gray-200 rounded-lg animate-pulse" />
          <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
        </div>
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div
              key={i}
              className="h-16 bg-gray-100 rounded-lg animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  // ── Retention not enabled ─────────────────────────────────────────────────
  if (notEnabled) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <Link
          to={`/suite_runs/${runId}`}
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 transition-colors group"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Run Detail</span>
        </Link>
        <Card className="border-yellow-200 bg-yellow-50/50">
          <CardContent className="py-12">
            <div className="text-center max-w-md mx-auto">
              <div className="mx-auto h-16 w-16 rounded-full bg-yellow-100 flex items-center justify-center mb-4">
                <FileText className="h-8 w-8 text-yellow-600" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                Message Retention Not Enabled
              </h3>
              <p className="text-sm text-gray-600">
                To retain raw NATS messages, start the processor with{" "}
                <code className="bg-gray-100 px-1 py-0.5 rounded text-xs font-mono">
                  --retain-messages
                </code>{" "}
                or set the{" "}
                <code className="bg-gray-100 px-1 py-0.5 rounded text-xs font-mono">
                  RETAIN_MESSAGES=true
                </code>{" "}
                environment variable.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  // ── Error / not found ─────────────────────────────────────────────────────
  if (error || !doc) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <Link
          to={`/suite_runs/${runId}`}
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 transition-colors group"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Run Detail</span>
        </Link>
        <Card className="border-red-200 bg-red-50/50">
          <CardContent className="py-12">
            <div className="text-center max-w-md mx-auto">
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                {error ?? "No Messages Found"}
              </h3>
              <p className="text-sm text-gray-600">
                No retained messages were found for run{" "}
                <code className="font-mono text-xs bg-gray-100 px-1 py-0.5 rounded">
                  {runId}
                </code>
                .
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  // ── Main content ──────────────────────────────────────────────────────────
  return (
    <div className="space-y-6 pb-8 animate-in fade-in duration-300">
      {/* Page header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div className="flex items-center gap-4">
          <Link
            to={`/suite_runs/${runId}`}
            className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-white border border-gray-200 text-gray-700 hover:bg-gray-50 hover:border-gray-300 transition-all shadow-sm hover:shadow group"
            aria-label="Back to run detail"
          >
            <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 tracking-tight">
              Raw Message Audit
            </h1>
            <p className="text-sm text-gray-500 mt-0.5 font-mono">
              {doc.runId}
            </p>
          </div>
        </div>

        {/* Header actions */}
        <div className="flex items-center gap-3">
          <Link
            to="/suite_runs/raw-messages"
            className="inline-flex items-center gap-2 px-3 py-1.5 bg-white border border-gray-200 rounded-lg text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 hover:border-gray-300 transition-colors"
          >
            Directory
          </Link>
          <button
            onClick={exportJsonLines}
            className="inline-flex items-center gap-2 px-3 py-1.5 bg-blue-600 text-white rounded-lg text-sm font-medium shadow-sm hover:bg-blue-700 transition-colors"
          >
            <Download className="h-4 w-4" />
            Export JSONL
          </button>
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-gray-200 rounded-lg text-sm font-medium text-gray-700 shadow-sm">
            <FileText className="h-4 w-4 text-gray-400" />
            {doc.messages.length} message{doc.messages.length !== 1 ? "s" : ""}
          </span>
        </div>
      </div>

      {/* Summary card */}
      <Card>
        <CardContent className="py-4">
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">
                Total Messages
              </p>
              <p className="text-2xl font-bold text-gray-900">
                {doc.messages.length}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">
                Event Types
              </p>
              <p className="text-2xl font-bold text-gray-900">
                {availableEventTypes.length}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">
                First Received
              </p>
              <p className="text-sm font-medium text-gray-700">
                {doc.messages.length > 0
                  ? new Date(doc.messages[0].receivedAt).toLocaleString(
                      undefined,
                      {
                        month: "short",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                        second: "2-digit",
                      },
                    )
                  : "—"}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">
                Last Received
              </p>
              <p className="text-sm font-medium text-gray-700">
                {doc.messages.length > 0
                  ? new Date(
                      doc.messages[doc.messages.length - 1].receivedAt,
                    ).toLocaleString(undefined, {
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                      second: "2-digit",
                    })
                  : "—"}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Filters */}
      <div className="space-y-3">
        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
          <input
            type="text"
            placeholder="Search by event type, subject, or payload content…"
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            className="w-full pl-10 pr-10 py-2.5 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
          />
          {searchText && (
            <button
              onClick={() => setSearchText("")}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
              aria-label="Clear search"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>

        {/* Event type filter pills */}
        {availableEventTypes.length > 0 && (
          <div className="flex flex-wrap gap-2 items-center">
            <span className="text-sm text-gray-500 font-medium">Type:</span>
            {availableEventTypes.map((eventType) => {
              const active = selectedEventTypes.has(eventType);
              return (
                <button
                  key={eventType}
                  onClick={() => toggleEventType(eventType)}
                  className={cn(
                    "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border transition-all",
                    active
                      ? eventTypeColor(eventType) +
                          " ring-2 ring-offset-1 ring-blue-400"
                      : "bg-white text-gray-600 border-gray-200 hover:bg-gray-50",
                  )}
                >
                  {eventType}
                  <span className="opacity-60">
                    {countsByEventType[eventType] ?? 0}
                  </span>
                </button>
              );
            })}
            {hasActiveFilters && (
              <button
                onClick={clearFilters}
                className="inline-flex items-center gap-1 text-xs text-gray-500 hover:text-gray-700 transition-colors ml-1"
              >
                <X className="h-3.5 w-3.5" />
                Clear filters
              </button>
            )}
          </div>
        )}
      </div>

      {/* Filtered count */}
      {hasActiveFilters && (
        <p className="text-sm text-gray-500">
          Showing {filteredMessages.length} of {doc.messages.length} messages
        </p>
      )}

      {/* Message list */}
      {filteredMessages.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center text-gray-500">
            {hasActiveFilters
              ? "No messages match the current filters."
              : "No messages in this run."}
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {filteredMessages.map(({ msg, originalIndex }) => (
            <MessageRow
              key={`${msg.subject}-${originalIndex}`}
              msg={msg}
              index={originalIndex}
            />
          ))}
        </div>
      )}
    </div>
  );
}
