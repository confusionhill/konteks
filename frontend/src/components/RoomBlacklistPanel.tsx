import React, { useState, useEffect, useCallback } from 'react';
import { ShieldAlert, Plus, Trash2, AlertCircle } from 'lucide-react';
import type { RoomArtifactBlacklist } from '../api/artifacts';
import {
  fetchRoomBlacklists,
  addRoomBlacklist,
  removeRoomBlacklist,
} from '../api/artifacts';

interface RoomBlacklistPanelProps {
  roomId: string;
}

export const RoomBlacklistPanel: React.FC<RoomBlacklistPanelProps> = ({ roomId }) => {
  const [blacklists, setBlacklists] = useState<RoomArtifactBlacklist[]>([]);
  const [targetType, setTargetType] = useState<'path' | 'group'>('path');
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadBlacklists = useCallback(async () => {
    try {
      const data = await fetchRoomBlacklists(roomId);
      setBlacklists(data || []);
    } catch (err: any) {
      console.error('Failed to load room blacklists:', err);
    }
  }, [roomId]);

  useEffect(() => {
    loadBlacklists();
  }, [loadBlacklists]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim()) return;

    setLoading(true);
    setError(null);
    try {
      await addRoomBlacklist(roomId, {
        artifact_id: targetType === 'path' ? inputValue.trim() : undefined,
        group_name: targetType === 'group' ? inputValue.trim() : undefined,
      });
      setInputValue('');
      loadBlacklists();
    } catch (err: any) {
      setError(err.message || 'Failed to add blacklist entry.');
    } finally {
      setLoading(false);
    }
  };

  const handleRemove = async (id: string) => {
    try {
      await removeRoomBlacklist(roomId, id);
      loadBlacklists();
    } catch (err: any) {
      alert(err.message || 'Failed to remove blacklist entry.');
    }
  };

  return (
    <div className="bg-white rounded-2xl p-6 border border-slate-200/80 shadow-soft space-y-4">
      <div className="flex items-center gap-2">
        <div className="p-2 rounded-lg bg-rose-50 text-rose-600 border border-rose-100">
          <ShieldAlert className="w-5 h-5" />
        </div>
        <div>
          <h2 className="text-lg font-bold text-slate-900">Room Memory Blacklist Exclusion Mask</h2>
          <p className="text-xs text-slate-500">
            Exclude specific artifacts or group folders from being retrieved for this room (0 token cost)
          </p>
        </div>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 text-red-700 rounded-xl text-xs flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Add Form */}
      <form onSubmit={handleAdd} className="flex flex-wrap items-center gap-3">
        <select
          value={targetType}
          onChange={(e) => setTargetType(e.target.value as 'path' | 'group')}
          className="bg-slate-50 border border-slate-200 text-slate-700 text-xs rounded-xl px-3 py-2 focus:outline-none focus:border-indigo-500 font-medium"
        >
          <option value="path">Exclude Artifact ID</option>
          <option value="group">Exclude Group Folder Tag</option>
        </select>

        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          placeholder={targetType === 'path' ? 'Enter Artifact ID (e.g. art_123)' : 'Enter Group Name (e.g. security)'}
          className="flex-1 min-w-[220px] px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs text-slate-800 focus:outline-none focus:border-indigo-500 font-mono"
          required
        />

        <button
          type="submit"
          disabled={loading}
          className="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl text-xs font-semibold flex items-center gap-1.5 transition-colors disabled:opacity-50"
        >
          <Plus className="w-4 h-4" />
          <span>Blacklist Memory</span>
        </button>
      </form>

      {/* Blacklist Items Table */}
      {blacklists.length === 0 ? (
        <p className="text-xs text-slate-400 italic bg-slate-50/50 p-4 rounded-xl border border-slate-100 text-center">
          No blacklisted memory exclusions configured for this room. All project memories are accessible.
        </p>
      ) : (
        <div className="border border-slate-200 rounded-xl overflow-hidden">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-50 border-b border-slate-200 text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                <th className="p-3">Exclusion Target Type</th>
                <th className="p-3">Target ID / Group</th>
                <th className="p-3 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-xs text-slate-700 font-mono">
              {blacklists.map((bl) => (
                <tr key={bl.id} className="hover:bg-slate-50/60">
                  <td className="p-3 font-sans font-medium text-slate-600">
                    {bl.artifact_id ? 'Artifact ID Exclusion' : 'Group Folder Exclusion'}
                  </td>
                  <td className="p-3 font-semibold text-rose-600">
                    {bl.artifact_id || bl.group_name}
                  </td>
                  <td className="p-3 text-right">
                    <button
                      onClick={() => handleRemove(bl.id)}
                      className="p-1 text-slate-400 hover:text-rose-600 transition-colors"
                      title="Remove Exclusion"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
