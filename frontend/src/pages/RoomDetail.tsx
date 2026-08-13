import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { fetchRoomDetail, fetchMessages, getArchiveDownloadUrl, fetchRoomEvaluations } from '../api/dashboard';
import type { RoomDetail, Message, ContextEvaluation } from '../api/dashboard';
import { StatePanel } from '../components/StatePanel';
import { MessageList } from '../components/MessageList';
import { DetailSkeleton } from '../components/Skeleton';
import { ArrowLeft, User, Calendar, Warning, CurrencyDollar, Coins, Lightning, DownloadSimple, Archive } from '@phosphor-icons/react';

// ---- Snapshot Viewer Component ----
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
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between px-3 py-2 bg-slate-100 hover:bg-slate-200 transition-colors text-left"
      >
        <span className="text-xs font-bold text-slate-600 uppercase tracking-wider">
          {label}{count !== undefined ? ` (${count})` : ''}
        </span>
        <span className={`text-slate-400 transition-transform text-xs ${open ? 'rotate-90' : ''}`}>▶</span>
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
      {/* Inline fields */}
      <div className="grid grid-cols-2 gap-2 text-xs">
        {s.summary && (
          <div className="col-span-2 bg-slate-50 border border-slate-200 rounded-lg p-2">
            <span className="font-bold text-slate-500 uppercase tracking-wider block mb-0.5">Summary</span>
            <p className="text-slate-700">{s.summary}</p>
          </div>
        )}
        {s.active_topic && (
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-2">
            <span className="font-bold text-slate-500 uppercase tracking-wider block mb-0.5">Active Topic</span>
            <p className="text-slate-700">{s.active_topic}</p>
          </div>
        )}
        {s.strategy_used && (
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-2">
            <span className="font-bold text-slate-500 uppercase tracking-wider block mb-0.5">Strategy</span>
            <p className="text-slate-700">{s.strategy_used}</p>
          </div>
        )}
      </div>

      {/* State */}
      {s.state && Object.keys(s.state).length > 0 && (
        <SnapshotSection label="State Variables" count={Object.keys(s.state).length}>
          <pre>{JSON.stringify(s.state, null, 2)}</pre>
        </SnapshotSection>
      )}

      {/* Relevant Messages */}
      {s.relevant_messages && s.relevant_messages.length > 0 && (
        <SnapshotSection label="Relevant Messages" count={s.relevant_messages.length}>
          <div className="space-y-2">
            {s.relevant_messages.map((m: any, i: number) => (
              <div key={i} className={`p-2 rounded ${m.role === 'user' ? 'bg-blue-50' : 'bg-slate-50'}`}>
                <span className="font-bold text-slate-500">{m.role}: </span>
                <span className="text-slate-700">{m.content}</span>
              </div>
            ))}
          </div>
        </SnapshotSection>
      )}

      {/* Memories */}
      {s.memories && s.memories.length > 0 && (
        <SnapshotSection label="Memories" count={s.memories.length}>
          <div className="space-y-2">
            {s.memories.map((m: any, i: number) => (
              <div key={i} className="p-2 rounded bg-amber-50 border border-amber-100">
                <p className="text-slate-700">{m.content}</p>
                {m.metadata && m.metadata !== '{}' && (
                  <p className="text-slate-400 mt-1 text-[10px]">{m.metadata}</p>
                )}
              </div>
            ))}
          </div>
        </SnapshotSection>
      )}

      {/* Entities */}
      {s.entities && s.entities.length > 0 && (
        <SnapshotSection label="Entities" count={s.entities.length}>
          <pre>{JSON.stringify(s.entities, null, 2)}</pre>
        </SnapshotSection>
      )}

      {/* References */}
      {s.references && Object.keys(s.references).length > 0 && (
        <SnapshotSection label="References" count={Object.keys(s.references).length}>
          <pre>{JSON.stringify(s.references, null, 2)}</pre>
        </SnapshotSection>
      )}
    </div>
  );
};
// ---- End Snapshot Viewer ----

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
          fetchRoomEvaluations(roomId).catch(e => { console.error("Evals not available", e); return { evaluations: [] }; })
        ]);
        setDetail(data);
        setMessages(data.messages || []);
        setEvaluations(evalsResp.evaluations || []);
        // The default server-side list returns up to 50 messages on detail page.
        // If it returns exactly 50, there might be more to paginate.
        setHasMore(data.messages ? data.messages.length >= 50 : false);
      } catch (err) {
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
      // Prepend older messages by passing the sequence of the currently oldest loaded message
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
      
      // Trigger download via temporary anchor
      const a = document.createElement('a');
      a.href = url;
      // We cannot force filename for cross-origin presigned URLs reliably, but we can set download attribute
      a.download = `${roomId}_archive.json.gz`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to download archive');
    } finally {
      setDownloading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Back Link */}
      <div>
        <Link
          to="/dashboard"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary-600 hover:text-primary-700 transition-colors group"
        >
          <ArrowLeft size={14} className="group-hover:-translate-x-1 transition-transform" />
          Back to Overview
        </Link>
      </div>

      {/* Error State */}
      {error && (
        <div className="bg-rose-50 border border-rose-200 text-rose-800 rounded-xl p-4 flex items-center gap-3 text-sm">
          <Warning size={20} className="text-rose-600 shrink-0" />
          <div>
            <span className="font-bold">Error:</span> {error}
          </div>
        </div>
      )}

      {/* Header Info */}
      {detail?.room && (
        <div className="bg-surface rounded-xl p-6 border border-slate-100 shadow-soft space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <span className="text-xs font-mono font-bold uppercase tracking-wider text-primary-600 bg-indigo-50 px-2.5 py-1 rounded-full border border-indigo-100">
                Room Inspection
              </span>
              {detail.room.status && detail.room.status !== 'active' && (
                <span className={`ml-2 text-xs font-mono font-bold uppercase tracking-wider px-2.5 py-1 rounded-full border ${
                  detail.room.status === 'archived' ? 'bg-slate-100 text-slate-500 border-slate-200' :
                  detail.room.status === 'archiving' ? 'bg-amber-100 text-amber-700 border-amber-200' :
                  'bg-blue-100 text-blue-700 border-blue-200'
                }`}>
                  {detail.room.status}
                </span>
              )}
              <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight font-sans mt-2">
                {detail.room.id}
              </h1>
            </div>

            <div className="flex flex-wrap gap-4 text-xs font-mono text-slate-500 border-t sm:border-t-0 pt-3 sm:pt-0 border-slate-100">
              <div className="flex items-center gap-1.5 bg-slate-50 px-3 py-1.5 rounded-lg border border-slate-100">
                <User size={14} className="text-primary-600" />
                <span>User: {detail.room.external_user_id || 'anonymous'}</span>
              </div>

              <div className="flex items-center gap-1.5 bg-slate-50 px-3 py-1.5 rounded-lg border border-slate-100">
                <Calendar size={14} className="text-slate-400" />
                <span>Created: {new Date(detail.room.created_at).toLocaleDateString()}</span>
              </div>
            </div>
          </div>

          {/* Per-Room Token & Cost Analytics Cards */}
          {detail.metrics && (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-2 border-t border-slate-100">
              <div className="bg-slate-50/80 p-3 rounded-lg border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider">Est. Room Cost</div>
                  <div className="text-base font-bold font-mono text-emerald-700 mt-0.5">
                    ${(detail.metrics.total_cost || 0).toFixed(6)}
                  </div>
                </div>
                <CurrencyDollar size={22} className="text-emerald-600" />
              </div>

              <div className="bg-slate-50/80 p-3 rounded-lg border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider">Avg Tokens / Turn</div>
                  <div className="text-base font-bold font-mono text-slate-900 mt-0.5">
                    {Math.round(detail.metrics.avg_tokens_per_turn || 0)}
                  </div>
                </div>
                <Coins size={22} className="text-amber-500" />
              </div>

              <div className="bg-slate-50/80 p-3 rounded-lg border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider">Total Room Tokens</div>
                  <div className="text-base font-bold font-mono text-primary-700 mt-0.5">
                    {detail.metrics.total_tokens || 0}
                  </div>
                </div>
                <Lightning size={22} className="text-primary-600" />
              </div>
            </div>
          )}
        </div>
      )}

      {/* Loading Skeleton or Content */}
      {loading ? (
        <DetailSkeleton />
      ) : (
        detail && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <StatePanel state={detail.state} />
            {detail.room.status === 'archived' ? (
              <div className="bg-surface rounded-xl border border-slate-100 shadow-soft p-8 flex flex-col items-center justify-center text-center space-y-4">
                <div className="w-16 h-16 bg-slate-50 rounded-full flex items-center justify-center mb-2 border border-slate-100">
                  <Archive size={32} className="text-slate-400" />
                </div>
                <h3 className="text-lg font-bold text-slate-900">Room Archived in Cold Storage</h3>
                <p className="text-sm text-slate-500 max-w-sm">
                  To minimize database costs, messages for this room have been zipped and moved to cold storage. 
                </p>
                <div className="pt-4">
                  <button
                    onClick={handleDownloadArchive}
                    disabled={downloading}
                    className="inline-flex items-center justify-center gap-2 px-5 py-2.5 text-sm font-semibold text-white bg-slate-900 hover:bg-slate-800 disabled:opacity-70 disabled:cursor-not-allowed rounded-lg shadow-sm transition-all focus:outline-none focus:ring-2 focus:ring-slate-900 focus:ring-offset-2"
                  >
                    {downloading ? (
                      <span className="inline-block w-4 h-4 border-2 border-slate-400 border-t-white rounded-full animate-spin"></span>
                    ) : (
                      <DownloadSimple size={18} weight="bold" />
                    )}
                    {downloading ? 'Preparing Download...' : 'Download .json.gz Archive'}
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
        <div className="mt-8 bg-surface rounded-xl border border-slate-100 shadow-soft p-6">
          <h2 className="text-xl font-bold text-slate-900 mb-2 tracking-tight">Context Evaluations Log</h2>
          <p className="text-sm text-slate-500 mb-4">
            Out-of-band context evaluations measured dynamically.
          </p>

          {/* Metric Legend for Onboarding */}
          <div className="mb-6 text-xs text-slate-600 bg-slate-50 p-4 rounded-lg border border-slate-100 leading-relaxed">
            <span className="font-bold text-slate-700 block mb-1">💡 Context Metrics Explanation:</span>
            <ul className="list-disc list-inside space-y-1">
              <li><strong>Relevance:</strong> Does the context contain the appropriate historical background, state, or entities needed to interpret/answer the query? (Defaults to 100% for brand-new topics).</li>
              <li><strong>Completeness:</strong> Are we missing any critical history or state variables discussed previously that are needed for this query? (Defaults to 100% for brand-new topics).</li>
              <li><strong>Token Efficiency:</strong> Is the retrieved context concise and high-signal, avoiding bloated or redundant variables? (100% means zero fluff).</li>
            </ul>
          </div>

          <div className="space-y-4">
            {evaluations.map((ev) => (
              <div key={ev.id} className="border border-slate-100 rounded-lg p-5 bg-slate-50/50">
                <div className="flex flex-wrap items-center justify-between mb-4 gap-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-mono font-bold text-slate-500">{new Date(ev.created_at).toLocaleString()}</span>
                    {ev.is_new_topic && (
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-primary-50 text-primary-700 border border-primary-100">
                        Topic Shift (New)
                      </span>
                    )}
                  </div>
                  <div className="flex gap-2 text-[11px] font-bold uppercase tracking-wider">
                    <span className={`px-2 py-1 rounded ${ev.relevance_score >= 0.8 ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}>
                      Relevance: {(ev.relevance_score * 100).toFixed(0)}%
                    </span>
                    <span className={`px-2 py-1 rounded ${ev.completeness_score >= 0.8 ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}>
                      Completeness: {(ev.completeness_score * 100).toFixed(0)}%
                    </span>
                    <span className={`px-2 py-1 rounded ${ev.token_efficiency_score >= 0.8 ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}>
                      Efficiency: {(ev.token_efficiency_score * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>
                <div className="mb-4">
                  <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">Query</span>
                  <p className="text-sm font-medium text-slate-900 bg-white p-3 border border-slate-200 rounded-lg">{ev.query}</p>
                </div>
                <div className="mb-4">
                  <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">Reasoning</span>
                  <p className="text-sm text-slate-700 bg-white p-3 border border-slate-200 rounded-lg italic leading-relaxed">{ev.reasoning}</p>
                </div>
                
                <details className="text-sm group">
                  <summary className="cursor-pointer font-semibold text-primary-600 hover:text-primary-700 list-none flex items-center gap-2">
                    <span className="group-open:rotate-90 transition-transform text-slate-400">▶</span>
                    View Measured Context Snapshot
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
