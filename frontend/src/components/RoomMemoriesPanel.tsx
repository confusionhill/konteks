import React, { useState, useEffect } from 'react';
import { Database, RefreshCw, RotateCcw, AlertTriangle, ShieldCheck } from 'lucide-react';
import { fetchRoomMemories, restoreMemory } from '../api/context_health';
import type { MemoryItem } from '../api/context_health';

interface RoomMemoriesPanelProps {
  roomId: string;
}

export const RoomMemoriesPanel: React.FC<RoomMemoriesPanelProps> = ({ roomId }) => {
  const [memories, setMemories] = useState<MemoryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [showSuperseded, setShowSuperseded] = useState(true);
  const [restoringId, setRestoringId] = useState<string | null>(null);
  const [cleaning, setCleaning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadMemories = async () => {
    if (!roomId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await fetchRoomMemories(roomId, showSuperseded);
      setMemories(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load room memories');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadMemories();
  }, [roomId, showSuperseded]);

  const handleRestore = async (memoryID: string) => {
    setRestoringId(memoryID);
    try {
      await restoreMemory(memoryID);
      await loadMemories();
    } catch (err: any) {
      alert(err.message || 'Failed to restore memory');
    } finally {
      setRestoringId(null);
    }
  };

  const handleTriggerClean = async () => {
    setCleaning(true);
    try {
      const token = localStorage.getItem('konteks_token');
      const activeProjStr = localStorage.getItem('konteks_active_project');
      let activeProjectID = '';
      if (activeProjStr) {
        try {
          activeProjectID = JSON.parse(activeProjStr)?.id || '';
        } catch {}
      }

      const response = await fetch(`/v1/rooms/${roomId}/context-health/clean`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
          ...(activeProjectID ? { 'X-Project-ID': activeProjectID } : {}),
        },
      });
      if (response.ok) {
        await loadMemories();
      }
    } catch (err: any) {
      console.warn('Manual cleanup error:', err);
    } finally {
      setCleaning(false);
    }
  };

  return (
    <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6 space-y-4 font-sans mt-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-100 pb-4">
        <div className="flex items-center space-x-3">
          <Database className="h-5 w-5 text-indigo-600" />
          <div>
            <h2 className="text-sm font-bold text-slate-900">Context Memory Audit & Supersedence DAG</h2>
            <p className="text-xs text-slate-500">Inspect stored facts, status decay, entropy scores, and restore superseded memories.</p>
          </div>
        </div>

        <div className="flex items-center space-x-3">
          <label className="flex items-center space-x-2 text-xs text-slate-600 cursor-pointer">
            <input
              type="checkbox"
              checked={showSuperseded}
              onChange={(e) => setShowSuperseded(e.target.checked)}
              className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500 h-4 w-4"
            />
            <span>Show Superseded Facts</span>
          </label>

          <button
            onClick={handleTriggerClean}
            disabled={cleaning}
            className="px-3 py-1.5 rounded-lg border border-slate-200 hover:bg-slate-50 text-xs font-semibold text-slate-700 transition-all flex items-center space-x-1.5"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${cleaning ? 'animate-spin' : ''}`} />
            <span>{cleaning ? 'Cleaning...' : 'Run Health Sweep'}</span>
          </button>
        </div>
      </div>

      {error && (
        <div className="text-xs text-rose-600 bg-rose-50 border border-rose-200 p-3 rounded-lg flex items-center space-x-2">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {loading ? (
        <div className="text-xs text-slate-500 py-6 text-center">Loading room context memories...</div>
      ) : memories.length === 0 ? (
        <div className="text-xs text-slate-400 py-6 text-center italic">No context memories stored for this room.</div>
      ) : (
        <div className="space-y-3">
          {memories.map((mem) => {
            const status = mem.status || 'active';
            return (
              <div
                key={mem.id}
                className={`p-4 rounded-xl border transition-all space-y-2.5 ${
                  status === 'active'
                    ? 'bg-slate-50/50 border-slate-200/80'
                    : status === 'superseded'
                    ? 'bg-amber-50/40 border-amber-200/80'
                    : 'bg-rose-50/40 border-rose-200/80'
                }`}
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center space-x-2">
                    <span
                      className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border ${
                        status === 'active'
                          ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                          : status === 'superseded'
                          ? 'bg-amber-100 text-amber-800 border-amber-300'
                          : 'bg-rose-100 text-rose-800 border-rose-300'
                      }`}
                    >
                      {status}
                    </span>

                    {mem.is_whitelisted && (
                      <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-indigo-50 text-indigo-700 border border-indigo-200">
                        <ShieldCheck className="h-3 w-3 text-indigo-600" />
                        <span>Whitelisted</span>
                      </span>
                    )}

                    <span className="text-[11px] font-mono text-slate-500">
                      Entropy: {mem.shannon_entropy ? mem.shannon_entropy.toFixed(2) : '0.00'}
                    </span>
                  </div>

                  {status === 'superseded' && (
                    <button
                      onClick={() => handleRestore(mem.id)}
                      disabled={restoringId === mem.id}
                      className="px-2.5 py-1 rounded bg-amber-600 hover:bg-amber-700 text-white text-[11px] font-semibold transition-all shadow-xs flex items-center space-x-1"
                    >
                      <RotateCcw className="h-3 w-3" />
                      <span>{restoringId === mem.id ? 'Restoring...' : 'Restore Fact'}</span>
                    </button>
                  )}
                </div>

                <p className="text-xs font-mono text-slate-800 bg-white p-3 rounded-lg border border-slate-200/80 leading-relaxed">
                  {mem.content}
                </p>

                {mem.superseded_by_id && (
                  <p className="text-[11px] text-amber-700 font-mono">
                    Superseded by memory ID: <span className="font-bold">{mem.superseded_by_id}</span>
                  </p>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
