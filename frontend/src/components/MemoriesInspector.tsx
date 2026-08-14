import React, { useState } from 'react';
import { Search, Brain, Clock, Hash } from 'lucide-react';

interface MemoryItem {
  id: string;
  room_id?: string;
  external_user_id?: string;
  content: string;
  created_at?: string;
}

interface MemoriesInspectorProps {
  memories: MemoryItem[];
  loading: boolean;
}

export const MemoriesInspector: React.FC<MemoriesInspectorProps> = ({ memories, loading }) => {
  const [search, setSearch] = useState('');

  const filteredMemories = memories.filter((m) =>
    m.content.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-4 font-sans">
      <div className="flex items-center justify-between gap-4 bg-white p-4 rounded-2xl border border-slate-200/80 shadow-soft">
        <div className="relative flex-1">
          <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search extracted fact memories..."
            className="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-medium text-slate-900 focus:outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100"
          />
        </div>
        <span className="text-xs text-slate-600 font-mono font-semibold bg-slate-100 px-3 py-2 rounded-xl border border-slate-200">
          Total Facts: {filteredMemories.length}
        </span>
      </div>

      {loading ? (
        <div className="p-12 text-center text-sm font-medium text-slate-500 bg-white rounded-2xl border border-slate-200/80 shadow-soft">
          Loading fact memories...
        </div>
      ) : filteredMemories.length === 0 ? (
        <div className="bg-white rounded-2xl border border-slate-200/80 shadow-soft p-12 text-center">
          <div className="w-12 h-12 rounded-2xl bg-indigo-50 text-indigo-600 border border-indigo-100 flex items-center justify-center mx-auto mb-3">
            <Brain className="w-6 h-6" />
          </div>
          <h3 className="text-base font-bold text-slate-900">No Extracted Fact Memories Found</h3>
          <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1">
            Short-term facts extracted automatically from room conversations will appear here.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {filteredMemories.map((mem) => (
            <div
              key={mem.id}
              className="p-4 bg-white border border-slate-200/80 hover:border-indigo-200 rounded-2xl transition-all shadow-soft flex flex-col justify-between"
            >
              <div className="flex items-start gap-3">
                <div className="p-2 rounded-xl bg-emerald-50 text-emerald-600 border border-emerald-100 shrink-0 mt-0.5">
                  <Brain className="w-4 h-4" />
                </div>
                <p className="text-xs text-slate-800 font-medium leading-relaxed">{mem.content}</p>
              </div>

              <div className="flex items-center justify-between mt-3 pt-3 border-t border-slate-100 text-[11px] text-slate-500">
                {mem.room_id && (
                  <span className="flex items-center gap-1 font-mono font-bold text-slate-600">
                    <Hash className="w-3.5 h-3.5 text-slate-400" />
                    Room: {mem.room_id.substring(0, 8)}...
                  </span>
                )}
                {mem.created_at && (
                  <span className="flex items-center gap-1 font-medium">
                    <Clock className="w-3.5 h-3.5" />
                    {new Date(mem.created_at).toLocaleDateString()}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
