import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, Paperclip, Trash2, Upload, X } from "lucide-react";
import { api } from "../api/client";
import { type AuthUser, hasAnyRole } from "../api/auth";

interface Props {
  sampleResultId: string;
  user: AuthUser;
  onClose: () => void;
}

type Tab = "comments" | "attachments";

export function SampleResultDetails({ sampleResultId, user, onClose }: Props) {
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>("comments");
  const [commentBody, setCommentBody] = useState("");
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [commentError, setCommentError] = useState<string | null>(null);

  const canComment = hasAnyRole(user, ["admin", "operator", "reviewer"]);
  const canUpload = hasAnyRole(user, ["admin", "operator"]);
  const canDelete = hasAnyRole(user, ["admin", "reviewer"]);

  const commentsQuery = useQuery({
    queryKey: ["comments", sampleResultId],
    queryFn: () => api.listComments(sampleResultId),
  });

  const attachmentsQuery = useQuery({
    queryKey: ["attachments", sampleResultId],
    queryFn: () => api.listAttachments(sampleResultId),
  });

  const createComment = useMutation({
    mutationFn: (body: string) => api.createComment(sampleResultId, body),
    onSuccess: () => {
      setCommentBody("");
      setCommentError(null);
      queryClient.invalidateQueries({ queryKey: ["comments", sampleResultId] });
    },
    onError: (err: Error) => setCommentError(err.message),
  });

  const uploadMutation = useMutation({
    mutationFn: (file: File) => api.uploadAttachment(sampleResultId, file),
    onSuccess: () => {
      setUploadError(null);
      queryClient.invalidateQueries({ queryKey: ["attachments", sampleResultId] });
    },
    onError: (err: Error) => setUploadError(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteAttachment(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["attachments", sampleResultId] }),
  });

  function handleDownload(attachmentId: string, filename: string) {
    api
      .downloadAttachment(attachmentId)
      .then((blob) => {
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
      })
      .catch((err: Error) => setUploadError(err.message));
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    uploadMutation.mutate(file);
    e.target.value = "";
  }

  function formatBytes(n: number): string {
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
    return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl bg-white shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
          <h3 className="text-lg font-semibold text-slate-900">
            Sample result details
          </h3>
          <button
            onClick={onClose}
            className="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="border-b border-slate-200">
          <nav className="flex px-6">
            <TabButton
              active={tab === "comments"}
              onClick={() => setTab("comments")}
              count={commentsQuery.data?.length}
            >
              Comments
            </TabButton>
            <TabButton
              active={tab === "attachments"}
              onClick={() => setTab("attachments")}
              count={attachmentsQuery.data?.length}
            >
              Attachments
            </TabButton>
          </nav>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-4">
          {tab === "comments" && (
            <div>
              {commentsQuery.isLoading ? (
                <p className="py-4 text-center text-sm text-slate-500">Loading...</p>
              ) : commentsQuery.data?.length === 0 ? (
                <p className="py-4 text-center text-sm text-slate-400">
                  No comments yet
                </p>
              ) : (
                <ul className="space-y-3">
                  {commentsQuery.data?.map((c) => (
                    <li
                      key={c.id}
                      className="rounded-lg border border-slate-200 bg-slate-50 p-3"
                    >
                      <div className="flex items-center justify-between text-xs text-slate-500">
                        <span className="font-mono">
                          {c.author_id.slice(0, 8)}…
                        </span>
                        <span>{new Date(c.created_at).toLocaleString()}</span>
                      </div>
                      <p className="mt-1 whitespace-pre-wrap text-sm text-slate-800">
                        {c.body}
                      </p>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {tab === "attachments" && (
            <div>
              {attachmentsQuery.isLoading ? (
                <p className="py-4 text-center text-sm text-slate-500">Loading...</p>
              ) : attachmentsQuery.data?.length === 0 ? (
                <p className="py-4 text-center text-sm text-slate-400">
                  No attachments yet
                </p>
              ) : (
                <ul className="space-y-2">
                  {attachmentsQuery.data?.map((a) => (
                    <li
                      key={a.id}
                      className="flex items-center justify-between rounded-lg border border-slate-200 bg-slate-50 px-3 py-2"
                    >
                      <div className="flex min-w-0 flex-1 items-center gap-2">
                        <Paperclip className="h-4 w-4 shrink-0 text-slate-400" />
                        <div className="min-w-0">
                          <p className="truncate text-sm font-medium text-slate-800">
                            {a.filename}
                          </p>
                          <p className="text-xs text-slate-500">
                            {a.content_type} · {formatBytes(a.size_bytes)} ·{" "}
                            {new Date(a.uploaded_at).toLocaleString()}
                          </p>
                        </div>
                      </div>
                      <div className="ml-3 flex shrink-0 items-center gap-1">
                        <button
                          onClick={() => handleDownload(a.id, a.filename)}
                          className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-2 py-1 text-xs text-slate-700 hover:bg-slate-50"
                        >
                          <Download className="h-3 w-3" />
                          Download
                        </button>
                        {canDelete && (
                          <button
                            onClick={() => deleteMutation.mutate(a.id)}
                            disabled={deleteMutation.isPending}
                            className="inline-flex items-center gap-1 rounded-md border border-red-200 bg-white px-2 py-1 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50"
                          >
                            <Trash2 className="h-3 w-3" />
                          </button>
                        )}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </div>

        {tab === "comments" && canComment && (
          <div className="border-t border-slate-200 bg-slate-50 px-6 py-3">
            {commentError && (
              <p className="mb-2 text-xs text-red-600">{commentError}</p>
            )}
            <textarea
              value={commentBody}
              onChange={(e) => setCommentBody(e.target.value)}
              rows={2}
              placeholder="Add a comment..."
              maxLength={4000}
              className="block w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <div className="mt-2 flex justify-end">
              <button
                onClick={() => createComment.mutate(commentBody.trim())}
                disabled={
                  createComment.isPending || commentBody.trim().length === 0
                }
                className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {createComment.isPending ? "Posting..." : "Post comment"}
              </button>
            </div>
          </div>
        )}

        {tab === "attachments" && canUpload && (
          <div className="border-t border-slate-200 bg-slate-50 px-6 py-3">
            {uploadError && (
              <p className="mb-2 text-xs text-red-600">{uploadError}</p>
            )}
            <label className="flex cursor-pointer items-center gap-3 text-sm text-slate-700">
              <span className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700">
                <Upload className="h-4 w-4" />
                {uploadMutation.isPending ? "Uploading..." : "Upload file"}
              </span>
              <input
                type="file"
                accept=".pdf,.png,.jpg,.jpeg,.csv,.xlsx"
                onChange={handleFileChange}
                disabled={uploadMutation.isPending}
                className="hidden"
              />
              <span className="text-xs text-slate-500">
                PDF, PNG, JPEG, CSV, or XLSX · max 10 MB
              </span>
            </label>
          </div>
        )}
      </div>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  count,
  children,
}: {
  active: boolean;
  onClick: () => void;
  count?: number;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`-mb-px border-b-2 px-3 py-3 text-sm font-medium transition ${
        active
          ? "border-blue-600 text-blue-600"
          : "border-transparent text-slate-500 hover:text-slate-700"
      }`}
    >
      {children}
      {count !== undefined && (
        <span className="ml-1.5 rounded-full bg-slate-100 px-1.5 py-0.5 text-xs text-slate-500">
          {count}
        </span>
      )}
    </button>
  );
}
