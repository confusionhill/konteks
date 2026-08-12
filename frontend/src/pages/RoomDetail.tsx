import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { fetchRoomDetail } from '../api/dashboard';
import type { RoomDetail } from '../api/dashboard';
import { StatePanel } from '../components/StatePanel';
import { MessageList } from '../components/MessageList';
import { DetailSkeleton } from '../components/Skeleton';
import { ArrowLeft, User, Calendar, Warning } from '@phosphor-icons/react';

export const RoomDetailPage: React.FC = () => {
  const { roomId } = useParams<{ roomId: string }>();
  const [detail, setDetail] = useState<RoomDetail | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadRoom() {
      if (!roomId) return;
      try {
        setLoading(true);
        setError(null);
        const data = await fetchRoomDetail(roomId);
        setDetail(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load room details');
      } finally {
        setLoading(false);
      }
    }
    loadRoom();
  }, [roomId]);

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
        <div className="bg-surface rounded-xl p-6 border border-slate-100 shadow-soft">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <span className="text-xs font-mono font-bold uppercase tracking-wider text-primary-600 bg-indigo-50 px-2.5 py-1 rounded-full border border-indigo-100">
                Room Inspection
              </span>
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
        </div>
      )}

      {/* Loading Skeleton or Content */}
      {loading ? (
        <DetailSkeleton />
      ) : (
        detail && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <StatePanel state={detail.state} />
            <MessageList messages={detail.messages} />
          </div>
        )
      )}
    </div>
  );
};
