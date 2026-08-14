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
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4 bg-slate-900/60 p-4 rounded-xl border border-slate-800">
        <div className="relative flex-1">
          <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search extracted fact memories..."
            className="w-full pl-9 pr-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-sm text-slate-200 focus:outline-none focus:border-indigo-500"
          />
        </div>
        <span className="text-xs text-slate-400 font-mono bg-slate-950 px-3 py-1.5 rounded-lg border border-slate-800">
          Total Facts: {filteredMemories.length}
        </span>
      </div>

      {loading ? (
        <div className="p-8 text-center text-sm text-slate-400">Loading fact memories...</div>
      ) : filteredMemories.length === 0 ? (
        <div className="p-12 text-center bg-slate-950/40 rounded-xl border border-slate-800">
          <Brain className="w-10 h-10 text-slate-600 mx-auto mb-2" />
          <p className="text-sm font-medium text-slate-300">No Extracted Fact Memories Found</p>
          <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1">
            Short-term facts extracted automatically from room conversations will appear here.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {filteredMemories.map((mem) => (
            <div
              key={mem.id}
              className="p-4 bg-slate-900/70 border border-slate-800 hover:border-slate-700 rounded-xl transition-colors flex flex-col justify-between"
            >
              <div className="flex items-start gap-3">
                <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shrink-0 mt-0.5">
                  <Brain className="w-4 h-4" />
                </div>
                <p className="text-xs text-slate-200 font-medium leading-relaxed">{mem.content}</p>
              </div>

              <div className="flex items-center justify-between mt-3 pt-3 border-t border-slate-800/80 text-[11px] text-slate-500">
                {mem.room_id && (
                  <span className="flex items-center gap-1 font-mono">
                    <Hash className="w-3 h-3 text-slate-400" />
                    Room: {mem.room_id.substring(0, 8)}...
                  </span>
                )}
                {mem.created_at && (
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
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
