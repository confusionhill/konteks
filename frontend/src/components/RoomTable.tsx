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
    <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-slate-50/80 border-b border-slate-200 text-xs font-semibold text-slate-500 uppercase tracking-wider">
              <th className="py-4 px-6">Room ID</th>
              <th className="py-4 px-6">Active Topic</th>
              <th className="py-4 px-6">User ID</th>
              <th className="py-4 px-6">Messages</th>
              <th className="py-4 px-6">Last Active</th>
              <th className="py-4 px-4 text-right">Inspect</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 text-sm">
            {rooms.map((room) => (
              <tr
                key={room.id}
                onClick={() => navigate(`/dashboard/rooms/${room.id}`)}
                className="hover:bg-indigo-50/40 transition-colors duration-150 cursor-pointer group active:scale-[0.99]"
              >
                <td className="py-4 px-6 font-mono text-xs font-medium text-slate-900">
                  <span className="bg-slate-100 group-hover:bg-indigo-100 text-slate-800 group-hover:text-indigo-700 px-2.5 py-1 rounded-md border border-slate-200/60 transition-colors">
                    {(room.id || '').slice(0, 18)}...
                  </span>
                  {room.status && room.status !== 'active' && (
                    <span
                      className={`ml-2 px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider ${
                        room.status === 'archived'
                          ? 'bg-slate-100 text-slate-500 border border-slate-200'
                          : room.status === 'archiving'
                          ? 'bg-amber-50 text-amber-700 border border-amber-200'
                          : 'bg-blue-50 text-blue-700 border border-blue-200'
                      }`}
                    >
                      {room.status}
                    </span>
                  )}
                </td>
                <td className="py-4 px-6 font-medium text-slate-800">
                  {room.active_topic ? (
                    <span className="inline-flex items-center gap-1.5 bg-indigo-50 text-indigo-700 font-semibold px-3 py-0.5 rounded-full text-xs border border-indigo-100">
                      {room.active_topic}
                    </span>
                  ) : (
                    <span className="text-slate-400 italic">Uncategorized</span>
                  )}
                </td>
                <td className="py-4 px-6 text-slate-600 font-mono text-xs">
                  <span className="inline-flex items-center gap-1.5">
                    <User className="h-3.5 w-3.5 text-slate-400" />
                    {room.external_user_id || 'anonymous'}
                  </span>
                </td>
                <td className="py-4 px-6 text-slate-700 font-medium">
                  <span className="inline-flex items-center gap-1.5">
                    <MessageSquare className="h-4 w-4 text-indigo-600" />
                    {room.message_count}
                  </span>
                </td>
                <td className="py-4 px-6 text-slate-500 text-xs">
                  <span className="inline-flex items-center gap-1.5">
                    <Clock className="h-3.5 w-3.5 text-slate-400" />
                    {formatDate(room.last_active_at)}
                  </span>
                </td>
                <td className="py-4 px-4 text-right">
                  <span className="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-slate-100 group-hover:bg-indigo-600 group-hover:text-white text-slate-500 transition-colors">
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
