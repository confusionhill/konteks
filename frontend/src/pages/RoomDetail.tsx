import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { fetchRoomDetail, fetchMessages, getArchiveDownloadUrl } from '../api/dashboard';
import type { RoomDetail, Message } from '../api/dashboard';
import { StatePanel } from '../components/StatePanel';
import { MessageList } from '../components/MessageList';
import { DetailSkeleton } from '../components/Skeleton';
import { ArrowLeft, User, Calendar, Warning, CurrencyDollar, Coins, Lightning, DownloadSimple, Archive } from '@phosphor-icons/react';

export const RoomDetailPage: React.FC = () => {
  const { roomId } = useParams<{ roomId: string }>();
  const [detail, setDetail] = useState<RoomDetail | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
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
        const data = await fetchRoomDetail(roomId);
        setDetail(data);
        setMessages(data.messages || []);
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
    </div>
  );
};
