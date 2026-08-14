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
    <div className="border border-slate-200 rounded-lg overflow-hidden">
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center justify-between px-3 py-2 bg-slate-100 hover:bg-slate-200 transition-colors text-left"
      >
        <span className="text-xs font-bold text-slate-600 uppercase tracking-wider">
          {label}
          {count !== undefined ? ` (${count})` : ''}
        </span>
        <ChevronRight className={`h-3.5 w-3.5 text-slate-400 transition-transform ${open ? 'rotate-90' : ''}`} />
      </button>
      {open && (
        <div className="p-3 bg-white text-xs font-mono text-slate-700 leading-relaxed overflow-x-auto">
          {children}
        </div>
      )}
    </div>
  );
};

const ContextSnapshotViewer: React.FC<{ snapshot: any }> = ({ snapshot }) => {
  if (!snapshot) return <p className="text-xs text-slate-400 italic">No snapshot data</p>;
  const s = snapshot;
  return (
    <div className="space-y-2 mt-3">
      <div className="grid grid-cols-2 gap-2 text-xs">
        {s.summary && (
          <div className="col-span-2 bg-slate-50 border border-slate-200 rounded-lg p-2.5">
            <span className="font-bold text-slate-500 uppercase tracking-wider block text-[10px] mb-0.5">
              Summary
            </span>
            <p className="text-slate-700">{s.summary}</p>
          </div>
        )}
        {s.active_topic && (
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-2.5">
            <span className="font-bold text-slate-500 uppercase tracking-wider block text-[10px] mb-0.5">
              Active Topic
            </span>
            <p className="text-slate-700">{s.active_topic}</p>
          </div>
        )}
        {s.strategy_used && (
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-2.5">
            <span className="font-bold text-slate-500 uppercase tracking-wider block text-[10px] mb-0.5">
              Strategy
            </span>
            <p className="text-slate-700">{s.strategy_used}</p>
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
                className={`p-2 rounded ${m.role === 'user' ? 'bg-indigo-50 text-indigo-950' : 'bg-slate-50 text-slate-800'}`}
              >
                <span className="font-bold text-slate-500 text-[10px] uppercase block mb-0.5">{m.role}: </span>
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
              <div key={i} className="p-2.5 rounded bg-amber-50 border border-amber-100 text-slate-800">
                <p>{m.content}</p>
                {m.metadata && m.metadata !== '{}' && (
                  <p className="text-slate-400 mt-1 text-[10px]">{m.metadata}</p>
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
          className="inline-flex items-center space-x-1.5 text-xs font-semibold text-indigo-600 hover:text-indigo-700 transition-colors group"
        >
          <ArrowLeft className="h-4 w-4 group-hover:-translate-x-1 transition-transform" />
          <span>Back to Overview</span>
        </Link>
      </div>

      {/* Error State */}
      {error && (
        <div className="bg-rose-50 border border-rose-200 text-rose-800 rounded-xl p-4 flex items-center space-x-3 text-sm">
          <AlertTriangle className="h-5 w-5 text-rose-600 shrink-0" />
          <div>
            <strong className="font-bold">Error:</strong> {error}
          </div>
        </div>
      )}

      {/* Header Info */}
      {detail?.room && (
        <div className="bg-white rounded-xl p-6 border border-slate-200/80 shadow-soft space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <span className="text-xs font-mono font-bold uppercase tracking-wider text-indigo-600 bg-indigo-50 px-2.5 py-1 rounded-full border border-indigo-100">
                Room Inspection
              </span>
              {detail.room.status && detail.room.status !== 'active' && (
                <span
                  className={`ml-2 text-xs font-mono font-bold uppercase tracking-wider px-2.5 py-1 rounded-full border ${
                    detail.room.status === 'archived'
                      ? 'bg-slate-100 text-slate-500 border-slate-200'
                      : detail.room.status === 'archiving'
                      ? 'bg-amber-50 text-amber-700 border-amber-200'
                      : 'bg-blue-50 text-blue-700 border-blue-200'
                  }`}
                >
                  {detail.room.status}
                </span>
              )}
              <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight font-mono mt-2">
                {detail.room.id}
              </h1>
              {detail.room.description && (
                <p className="text-sm font-medium text-slate-600 mt-1 flex items-center space-x-2">
                  <span className="bg-slate-100 text-slate-700 font-sans px-2.5 py-0.5 rounded-md border border-slate-200 text-xs">
                    {detail.room.description}
                  </span>
                </p>
              )}
              {detail.room.parent_room_id && (
                <p className="text-xs text-indigo-600 font-mono mt-1">
                  Parent Room:{' '}
                  <Link
                    to={`/dashboard/rooms/${detail.room.parent_room_id}`}
                    className="underline hover:text-indigo-800"
                  >
                    {detail.room.parent_room_id}
                  </Link>
                </p>
              )}
            </div>

            <div className="flex flex-wrap gap-3 text-xs font-mono text-slate-500 border-t sm:border-t-0 pt-3 sm:pt-0 border-slate-100">
              <div className="flex items-center space-x-1.5 bg-slate-50 px-3 py-1.5 rounded-lg border border-slate-100">
                <User className="h-3.5 w-3.5 text-indigo-600" />
                <span>User: {detail.room.external_user_id || 'anonymous'}</span>
              </div>

              <div className="flex items-center space-x-1.5 bg-slate-50 px-3 py-1.5 rounded-lg border border-slate-100">
                <Calendar className="h-3.5 w-3.5 text-slate-400" />
                <span>Created: {new Date(detail.room.created_at).toLocaleDateString()}</span>
              </div>
            </div>
          </div>

          {/* Per-Room Token & Cost Analytics Cards */}
          {detail.metrics && (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-3 border-t border-slate-100">
              <div className="bg-slate-50/80 p-3 rounded-lg border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
                    Est. Room Cost
                  </div>
                  <div className="text-base font-bold font-mono text-emerald-700 mt-0.5">
                    ${(detail.metrics.total_cost || 0).toFixed(6)}
                  </div>
                </div>
                <DollarSign className="h-5 w-5 text-emerald-600" />
              </div>

              <div className="bg-slate-50/80 p-3 rounded-lg border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
                    Avg Tokens / Turn
                  </div>
                  <div className="text-base font-bold font-mono text-slate-900 mt-0.5">
                    {Math.round(detail.metrics.avg_tokens_per_turn || 0)}
                  </div>
                </div>
                <Coins className="h-5 w-5 text-amber-500" />
              </div>

              <div className="bg-slate-50/80 p-3 rounded-lg border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
                    Total Room Tokens
                  </div>
                  <div className="text-base font-bold font-mono text-indigo-700 mt-0.5">
                    {detail.metrics.total_tokens || 0}
                  </div>
                </div>
                <Zap className="h-5 w-5 text-indigo-600" />
              </div>
            </div>
          )}
        </div>
      )}

      {/* Child Sub-Agents / Child Rooms Section */}
      {!loading && detail?.children && detail.children.length > 0 && (
        <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6 space-y-4">
          <div className="flex items-center space-x-2 text-slate-900">
            <Sparkles className="h-5 w-5 text-indigo-600" />
            <h2 className="text-xl font-bold tracking-tight">Child Sub-Agent Rooms ({detail.children.length})</h2>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {detail.children.map((child) => (
              <div key={child.room_id} className="border border-slate-200 rounded-xl p-4 bg-slate-50/60 space-y-2">
                <div className="flex items-center justify-between">
                  <Link
                    to={`/dashboard/rooms/${child.room_id}`}
                    className="font-mono font-bold text-indigo-600 hover:underline text-sm"
                  >
                    {child.room_id}
                  </Link>
                  <span className="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 bg-emerald-100 text-emerald-800 rounded-full">
                    {child.status}
                  </span>
                </div>
                {child.description && (
                  <p className="text-xs font-semibold text-slate-800 bg-white p-2 rounded border border-slate-200">
                    🎯 {child.description}
                  </p>
                )}
                {child.active_topic && (
                  <p className="text-xs text-slate-600">
                    <strong className="text-slate-500 uppercase text-[10px]">Active Topic:</strong> {child.active_topic}
                  </p>
                )}
                {child.summary && (
                  <p className="text-xs text-slate-600 italic bg-white p-2 rounded border border-slate-100">
                    "{child.summary}"
                  </p>
                )}
              </div>
            ))}
          </div>
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
              <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-8 flex flex-col items-center justify-center text-center space-y-4">
                <div className="w-16 h-16 bg-slate-50 rounded-2xl flex items-center justify-center border border-slate-100 text-slate-400">
                  <Archive className="h-8 w-8" />
                </div>
                <h3 className="text-lg font-bold text-slate-900">Room Archived in Cold Storage</h3>
                <p className="text-sm text-slate-500 max-w-sm leading-relaxed">
                  To minimize database costs, messages for this room have been zipped and stored in MinIO S3 object storage.
                </p>
                <div className="pt-2">
                  <button
                    onClick={handleDownloadArchive}
                    disabled={downloading}
                    className="inline-flex items-center justify-center space-x-2 px-5 py-2.5 text-sm font-semibold text-white bg-slate-900 hover:bg-slate-800 disabled:opacity-70 rounded-lg shadow-sm transition-all"
                  >
                    {downloading ? (
                      <span className="inline-block w-4 h-4 border-2 border-slate-400 border-t-white rounded-full animate-spin" />
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
        <div className="mt-8 bg-white rounded-xl border border-slate-200/80 shadow-soft p-6 space-y-4">
          <div className="flex items-center space-x-2 text-slate-900">
            <Sparkles className="h-5 w-5 text-indigo-600" />
            <h2 className="text-xl font-bold tracking-tight">Context Evaluations Log</h2>
          </div>
          <p className="text-sm text-slate-500">
            Out-of-band context evaluations measured dynamically.
          </p>

          <div className="space-y-4">
            {evaluations.map((ev) => (
              <div key={ev.id} className="border border-slate-100 rounded-lg p-5 bg-slate-50/50 space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center space-x-2">
                    <span className="text-xs font-mono font-bold text-slate-500">
                      {new Date(ev.created_at).toLocaleString()}
                    </span>
                    {ev.is_new_topic && (
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-indigo-50 text-indigo-700 border border-indigo-100">
                        Topic Shift
                      </span>
                    )}
                  </div>
                  <div className="flex gap-2 text-[11px] font-bold uppercase tracking-wider">
                    <span
                      className={`px-2.5 py-1 rounded-md ${
                        ev.relevance_score >= 0.8
                          ? 'bg-emerald-100 text-emerald-800'
                          : 'bg-amber-100 text-amber-800'
                      }`}
                    >
                      Relevance: {(ev.relevance_score * 100).toFixed(0)}%
                    </span>
                    <span
                      className={`px-2.5 py-1 rounded-md ${
                        ev.completeness_score >= 0.8
                          ? 'bg-emerald-100 text-emerald-800'
                          : 'bg-amber-100 text-amber-800'
                      }`}
                    >
                      Completeness: {(ev.completeness_score * 100).toFixed(0)}%
                    </span>
                    <span
                      className={`px-2.5 py-1 rounded-md ${
                        ev.token_efficiency_score >= 0.8
                          ? 'bg-emerald-100 text-emerald-800'
                          : 'bg-amber-100 text-amber-800'
                      }`}
                    >
                      Efficiency: {(ev.token_efficiency_score * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>

                <div>
                  <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                    Query
                  </span>
                  <p className="text-sm font-medium text-slate-900 bg-white p-3 border border-slate-200 rounded-lg">
                    {ev.query}
                  </p>
                </div>

                <div>
                  <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                    Reasoning
                  </span>
                  <p className="text-sm text-slate-700 bg-white p-3 border border-slate-200 rounded-lg italic leading-relaxed">
                    {ev.reasoning}
                  </p>
                </div>

                <details className="text-sm">
                  <summary className="cursor-pointer font-semibold text-indigo-600 hover:text-indigo-700 list-none flex items-center space-x-1">
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
