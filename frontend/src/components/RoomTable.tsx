import React from 'react';
import { useNavigate } from 'react-router-dom';
import type { RoomOverview } from '../api/dashboard';
import { MessageSquare, Clock, User, ChevronRight } from 'lucide-react';

interface RoomTableProps {
  rooms: RoomOverview[];
}

export const RoomTable: React.FC<RoomTableProps> = ({ rooms }) => {
  const navigate = useNavigate();

  const formatDate = (isoString: string) => {
    if (!isoString) return 'Just now';
    try {
      const date = new Date(isoString);
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch {
      return isoString;
    }
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl overflow-hidden backdrop-blur-xl">
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-slate-950/80 border-b border-slate-800 text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
              <th className="py-3.5 px-5">Room ID</th>
              <th className="py-3.5 px-5">Active Topic</th>
              <th className="py-3.5 px-5">User ID</th>
              <th className="py-3.5 px-5">Messages</th>
              <th className="py-3.5 px-5">Last Active</th>
              <th className="py-3.5 px-4 text-right">Inspect</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 text-xs">
            {rooms.map((room) => (
              <tr
                key={room.id}
                onClick={() => navigate(`/dashboard/rooms/${room.id}`)}
                className="hover:bg-slate-800/40 transition-colors duration-150 cursor-pointer group"
              >
                <td className="py-3.5 px-5 font-mono text-slate-300">
                  <span className="bg-slate-950 group-hover:bg-indigo-950/80 group-hover:text-indigo-300 px-2 py-1 rounded-lg border border-slate-800 transition-colors">
                    {(room.id || '').slice(0, 18)}...
                  </span>
                  {room.status && room.status !== 'active' && (
                    <span
                      className={`ml-2 px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider ${
                        room.status === 'archived'
                          ? 'bg-slate-800 text-slate-400'
                          : room.status === 'archiving'
                          ? 'bg-amber-950/80 text-amber-300 border border-amber-800/50'
                          : 'bg-blue-950/80 text-blue-300 border border-blue-800/50'
                      }`}
                    >
                      {room.status}
                    </span>
                  )}
                </td>
                <td className="py-3.5 px-5 font-medium text-slate-200">
                  {room.active_topic ? (
                    <span className="inline-flex items-center gap-1.5 bg-indigo-950/80 border border-indigo-500/30 text-indigo-300 font-semibold px-2.5 py-0.5 rounded-full text-[11px]">
                      {room.active_topic}
                    </span>
                  ) : (
                    <span className="text-slate-500 italic">Uncategorized</span>
                  )}
                </td>
                <td className="py-3.5 px-5 text-slate-400 font-mono">
                  <span className="inline-flex items-center gap-1.5">
                    <User className="h-3.5 w-3.5 text-slate-500" />
                    {room.external_user_id || 'anonymous'}
                  </span>
                </td>
                <td className="py-3.5 px-5 text-slate-300 font-medium">
                  <span className="inline-flex items-center gap-1.5">
                    <MessageSquare className="h-3.5 w-3.5 text-indigo-400" />
                    {room.message_count}
                  </span>
                </td>
                <td className="py-3.5 px-5 text-slate-400">
                  <span className="inline-flex items-center gap-1.5">
                    <Clock className="h-3.5 w-3.5 text-slate-500" />
                    {formatDate(room.last_active_at)}
                  </span>
                </td>
                <td className="py-3.5 px-4 text-right">
                  <span className="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-slate-800 group-hover:bg-indigo-600 group-hover:text-white text-slate-400 transition-colors">
                    <ChevronRight className="h-4 w-4" />
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
