import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { fetchRoomDetail, fetchMessages, getArchiveDownloadUrl, fetchRoomEvaluations } from '../api/dashboard';
import type { RoomDetail, Message, ContextEvaluation } from '../api/dashboard';
import { StatePanel } from '../components/StatePanel';
import { MessageList } from '../components/MessageList';
import { DetailSkeleton } from '../components/Skeleton';
import {
  ArrowLeft,
  User,
  Calendar,
  AlertTriangle,
  DollarSign,
  Coins,
  Zap,
  Download,
  Archive,
  ChevronRight,
  Sparkles,
} from 'lucide-react';

interface SnapshotSection {
  label: string;
  count?: number;
  children: React.ReactNode;
}

const SnapshotSection: React.FC<SnapshotSection> = ({ label, count, children }) => {
  const [open, setOpen] = useState(false);
  return (
    <div className="border border-slate-800 rounded-xl overflow-hidden bg-slate-950/60">
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center justify-between px-3 py-2 bg-slate-900 hover:bg-slate-800/80 transition-colors text-left"
      >
        <span className="text-[11px] font-bold text-slate-300 uppercase tracking-wider">
          {label}
          {count !== undefined ? ` (${count})` : ''}
        </span>
        <ChevronRight className={`h-3.5 w-3.5 text-slate-400 transition-transform ${open ? 'rotate-90' : ''}`} />
      </button>
      {open && (
        <div className="p-3 bg-slate-950 text-xs font-mono text-slate-300 leading-relaxed overflow-x-auto">
          {children}
        </div>
      )}
    </div>
  );
};

const ContextSnapshotViewer: React.FC<{ snapshot: any }> = ({ snapshot }) => {
  if (!snapshot) return <p className="text-xs text-slate-500 italic">No snapshot data</p>;
  const s = snapshot;
  return (
    <div className="space-y-2 mt-3">
      <div className="grid grid-cols-2 gap-2 text-xs">
        {s.summary && (
          <div className="col-span-2 bg-slate-900/80 border border-slate-800 rounded-xl p-2.5">
            <span className="font-bold text-slate-400 uppercase tracking-wider block text-[10px] mb-1">
              Summary
            </span>
            <p className="text-slate-200 text-xs">{s.summary}</p>
          </div>
        )}
        {s.active_topic && (
          <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-2.5">
            <span className="font-bold text-slate-400 uppercase tracking-wider block text-[10px] mb-1">
              Active Topic
            </span>
            <p className="text-slate-200 text-xs">{s.active_topic}</p>
          </div>
        )}
        {s.strategy_used && (
          <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-2.5">
            <span className="font-bold text-slate-400 uppercase tracking-wider block text-[10px] mb-1">
              Strategy
            </span>
            <p className="text-slate-200 text-xs">{s.strategy_used}</p>
          </div>
        )}
      </div>

      {s.state && Object.keys(s.state).length > 0 && (
        <SnapshotSection label="State Variables" count={Object.keys(s.state).length}>
          <pre>{JSON.stringify(s.state, null, 2)}</pre>
        </SnapshotSection>
      )}

      {s.relevant_messages && s.relevant_messages.length > 0 && (
        <SnapshotSection label="Relevant Messages" count={s.relevant_messages.length}>
          <div className="space-y-2">
            {s.relevant_messages.map((m: any, i: number) => (
              <div
                key={i}
                className={`p-2 rounded-lg text-xs ${
                  m.role === 'user' ? 'bg-indigo-950/60 text-indigo-200 border border-indigo-500/20' : 'bg-slate-900 text-slate-300'
                }`}
              >
                <span className="font-bold uppercase tracking-wider text-[10px] text-slate-400 block mb-0.5">
                  {m.role}
                </span>
                <span>{m.content}</span>
              </div>
            ))}
          </div>
        </SnapshotSection>
      )}

      {s.memories && s.memories.length > 0 && (
        <SnapshotSection label="Memories" count={s.memories.length}>
          <div className="space-y-2">
            {s.memories.map((m: any, i: number) => (
              <div key={i} className="p-2.5 rounded-lg bg-amber-950/40 border border-amber-500/30 text-amber-200 text-xs">
                <p>{m.content}</p>
                {m.metadata && m.metadata !== '{}' && (
                  <p className="text-amber-400/70 mt-1 text-[10px] font-mono">{m.metadata}</p>
                )}
              </div>
            ))}
          </div>
        </SnapshotSection>
      )}

      {s.entities && s.entities.length > 0 && (
        <SnapshotSection label="Entities" count={s.entities.length}>
          <pre>{JSON.stringify(s.entities, null, 2)}</pre>
        </SnapshotSection>
      )}

      {s.references && Object.keys(s.references).length > 0 && (
        <SnapshotSection label="References" count={Object.keys(s.references).length}>
          <pre>{JSON.stringify(s.references, null, 2)}</pre>
        </SnapshotSection>
      )}
    </div>
  );
};

export const RoomDetailPage: React.FC = () => {
  const { roomId } = useParams<{ roomId: string }>();
  const [detail, setDetail] = useState<RoomDetail | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [evaluations, setEvaluations] = useState<ContextEvaluation[]>([]);
  const [hasMore, setHasMore] = useState<boolean>(false);
  const [loadingMore, setLoadingMore] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [downloading, setDownloading] = useState(false);

  useEffect(() => {
    async function loadRoom() {
      if (!roomId) return;
      try {
        setLoading(true);
        setError(null);
        const [data, evalsResp] = await Promise.all([
          fetchRoomDetail(roomId),
          fetchRoomEvaluations(roomId).catch((e) => {
            console.error('Evals not available', e);
            return { evaluations: [] };
          }),
        ]);
        setDetail(data);
        setMessages(data.messages || []);
        setEvaluations(evalsResp.evaluations || []);
        setHasMore(data.messages ? data.messages.length >= 50 : false);
      } catch (err: any) {
        setError(err instanceof Error ? err.message : 'Failed to load room details');
      } finally {
        setLoading(false);
      }
    }
    loadRoom();
  }, [roomId]);

  const handleLoadMore = async () => {
    if (!roomId || messages.length === 0) return;
    try {
      setLoadingMore(true);
      const oldestSeq = messages[0].sequence;
      const resp = await fetchMessages(roomId, 20, oldestSeq);
      setMessages((prev) => [...resp.messages, ...prev]);
      setHasMore(resp.has_more);
    } catch (err) {
      console.error('Failed to load older messages:', err);
    } finally {
      setLoadingMore(false);
    }
  };

  const handleDownloadArchive = async () => {
    if (!roomId) return;
    try {
      setDownloading(true);
      setError(null);
      const { url } = await getArchiveDownloadUrl(roomId);

      const a = document.createElement('a');
      a.href = url;
      a.download = `${roomId}_archive.json.gz`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    } catch (err: any) {
      setError(err instanceof Error ? err.message : 'Failed to download archive');
    } finally {
      setDownloading(false);
    }
  };

  return (
    <div className="space-y-6 font-sans">
      {/* Back Link */}
      <div>
        <Link
          to="/dashboard"
          className="inline-flex items-center space-x-1.5 text-xs font-semibold text-indigo-400 hover:text-indigo-300 transition-colors group"
        >
          <ArrowLeft className="h-4 w-4 group-hover:-translate-x-1 transition-transform" />
          <span>Back to Overview</span>
        </Link>
      </div>

      {/* Error State */}
      {error && (
        <div className="bg-red-950/80 border border-red-500/50 text-red-200 rounded-2xl p-4 flex items-center space-x-3 text-xs">
          <AlertTriangle className="h-5 w-5 text-red-400 shrink-0" />
          <div>
            <strong className="font-bold">Error:</strong> {error}
          </div>
        </div>
      )}

      {/* Header Info */}
      {detail?.room && (
        <div className="bg-slate-900/70 border border-slate-800 rounded-2xl p-6 shadow-xl space-y-4 backdrop-blur-xl">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <span className="text-xs font-mono font-bold uppercase tracking-wider text-indigo-400 bg-indigo-950/80 px-3 py-1 rounded-full border border-indigo-500/30">
                Room Inspection
              </span>
              {detail.room.status && detail.room.status !== 'active' && (
                <span
                  className={`ml-2 text-xs font-mono font-bold uppercase tracking-wider px-3 py-1 rounded-full border ${
                    detail.room.status === 'archived'
                      ? 'bg-slate-800 text-slate-400 border-slate-700'
                      : detail.room.status === 'archiving'
                      ? 'bg-amber-950/80 text-amber-300 border-amber-800/50'
                      : 'bg-blue-950/80 text-blue-300 border-blue-800/50'
                  }`}
                >
                  {detail.room.status}
                </span>
              )}
              <h1 className="text-2xl font-extrabold text-white tracking-tight mt-2 font-mono">
                {detail.room.id}
              </h1>
            </div>

            <div className="flex flex-wrap gap-3 text-xs font-mono text-slate-400 border-t sm:border-t-0 pt-3 sm:pt-0 border-slate-800">
              <div className="flex items-center space-x-1.5 bg-slate-950 border border-slate-800 px-3 py-1.5 rounded-xl">
                <User className="h-3.5 w-3.5 text-indigo-400" />
                <span>User: {detail.room.external_user_id || 'anonymous'}</span>
              </div>

              <div className="flex items-center space-x-1.5 bg-slate-950 border border-slate-800 px-3 py-1.5 rounded-xl">
                <Calendar className="h-3.5 w-3.5 text-slate-500" />
                <span>Created: {new Date(detail.room.created_at).toLocaleDateString()}</span>
              </div>
            </div>
          </div>

          {/* Per-Room Token & Cost Analytics Cards */}
          {detail.metrics && (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-3 border-t border-slate-800">
              <div className="bg-slate-950/60 p-3.5 rounded-xl border border-slate-800 flex items-center justify-between">
                <div>
                  <div className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider">
                    Est. Room Cost
                  </div>
                  <div className="text-sm font-bold font-mono text-emerald-400 mt-0.5">
                    ${(detail.metrics.total_cost || 0).toFixed(6)}
                  </div>
                </div>
                <DollarSign className="h-5 w-5 text-emerald-400" />
              </div>

              <div className="bg-slate-950/60 p-3.5 rounded-xl border border-slate-800 flex items-center justify-between">
                <div>
                  <div className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider">
                    Avg Tokens / Turn
                  </div>
                  <div className="text-sm font-bold font-mono text-slate-200 mt-0.5">
                    {Math.round(detail.metrics.avg_tokens_per_turn || 0)}
                  </div>
                </div>
                <Coins className="h-5 w-5 text-amber-400" />
              </div>

              <div className="bg-slate-950/60 p-3.5 rounded-xl border border-slate-800 flex items-center justify-between">
                <div>
                  <div className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider">
                    Total Room Tokens
                  </div>
                  <div className="text-sm font-bold font-mono text-indigo-400 mt-0.5">
                    {detail.metrics.total_tokens || 0}
                  </div>
                </div>
                <Zap className="h-5 w-5 text-indigo-400" />
              </div>
            </div>
          )}
        </div>
      )}

      {/* Main Grid */}
      {loading ? (
        <DetailSkeleton />
      ) : (
        detail && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <StatePanel state={detail.state} />
            {detail.room.status === 'archived' ? (
              <div className="bg-slate-900/60 rounded-2xl border border-slate-800 p-8 flex flex-col items-center justify-center text-center space-y-4 shadow-xl backdrop-blur-xl">
                <div className="w-14 h-14 bg-slate-950 rounded-2xl flex items-center justify-center border border-slate-800 text-slate-400">
                  <Archive className="h-7 w-7" />
                </div>
                <h3 className="text-base font-bold text-white">Room Archived in Cold Storage</h3>
                <p className="text-xs text-slate-400 max-w-sm leading-relaxed">
                  To minimize database costs, messages for this room have been zipped and stored in MinIO S3 object storage.
                </p>
                <div className="pt-2">
                  <button
                    onClick={handleDownloadArchive}
                    disabled={downloading}
                    className="inline-flex items-center justify-center space-x-2 px-5 py-2.5 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 rounded-xl transition-all shadow-lg shadow-indigo-600/25"
                  >
                    {downloading ? (
                      <span className="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    ) : (
                      <Download className="h-4 w-4" />
                    )}
                    <span>{downloading ? 'Preparing Download...' : 'Download .json.gz Archive'}</span>
                  </button>
                </div>
              </div>
            ) : (
              <MessageList
                messages={messages}
                hasMore={hasMore}
                onLoadMore={handleLoadMore}
                loadingMore={loadingMore}
              />
            )}
          </div>
        )
      )}

      {/* Evaluations Log */}
      {!loading && detail && evaluations.length > 0 && (
        <div className="mt-8 bg-slate-900/70 border border-slate-800 rounded-2xl p-6 shadow-xl space-y-4">
          <div className="flex items-center space-x-2 text-white">
            <Sparkles className="h-5 w-5 text-indigo-400" />
            <h2 className="text-lg font-bold">Context Evaluations Log</h2>
          </div>
          <p className="text-xs text-slate-400">
            Out-of-band context evaluations measured dynamically.
          </p>

          <div className="space-y-4">
            {evaluations.map((ev) => (
              <div key={ev.id} className="border border-slate-800 rounded-xl p-4 bg-slate-950/60 space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center space-x-2">
                    <span className="text-[11px] font-mono text-slate-400">
                      {new Date(ev.created_at).toLocaleString()}
                    </span>
                    {ev.is_new_topic && (
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-indigo-950 text-indigo-300 border border-indigo-500/30">
                        Topic Shift
                      </span>
                    )}
                  </div>
                  <div className="flex gap-2 text-[10px] font-bold uppercase tracking-wider">
                    <span
                      className={`px-2 py-0.5 rounded-lg border ${
                        ev.relevance_score >= 0.8
                          ? 'bg-emerald-950/80 text-emerald-300 border-emerald-800/50'
                          : 'bg-amber-950/80 text-amber-300 border-amber-800/50'
                      }`}
                    >
                      Relevance: {(ev.relevance_score * 100).toFixed(0)}%
                    </span>
                    <span
                      className={`px-2 py-0.5 rounded-lg border ${
                        ev.completeness_score >= 0.8
                          ? 'bg-emerald-950/80 text-emerald-300 border-emerald-800/50'
                          : 'bg-amber-950/80 text-amber-300 border-amber-800/50'
                      }`}
                    >
                      Completeness: {(ev.completeness_score * 100).toFixed(0)}%
                    </span>
                    <span
                      className={`px-2 py-0.5 rounded-lg border ${
                        ev.token_efficiency_score >= 0.8
                          ? 'bg-emerald-950/80 text-emerald-300 border-emerald-800/50'
                          : 'bg-amber-950/80 text-amber-300 border-amber-800/50'
                      }`}
                    >
                      Efficiency: {(ev.token_efficiency_score * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>

                <div>
                  <span className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider block mb-1">
                    Query
                  </span>
                  <p className="text-xs font-medium text-slate-200 bg-slate-900 p-2.5 border border-slate-800 rounded-xl">
                    {ev.query}
                  </p>
                </div>

                <div>
                  <span className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider block mb-1">
                    Reasoning
                  </span>
                  <p className="text-xs text-slate-300 bg-slate-900 p-2.5 border border-slate-800 rounded-xl italic leading-relaxed">
                    {ev.reasoning}
                  </p>
                </div>

                <details className="text-xs">
                  <summary className="cursor-pointer font-semibold text-indigo-400 hover:text-indigo-300 list-none flex items-center space-x-1">
                    <span>View Measured Context Snapshot</span>
                  </summary>
                  <ContextSnapshotViewer snapshot={ev.context_snapshot} />
                </details>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
