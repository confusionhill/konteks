import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { ShieldCheck, Copy, Check, Trash2, AlertTriangle, Activity, Sliders, Info, Save } from 'lucide-react';
import { fetchContextHealthPolicy, updateContextHealthPolicy } from '../api/context_health';
import type { ContextHealthPolicy } from '../api/context_health';

export const Settings: React.FC = () => {
  const { activeProject, deleteProject } = useAuth();
  const [copied, setCopied] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const navigate = useNavigate();

  // Context Health Policy State
  const [policy, setPolicy] = useState<ContextHealthPolicy>({
    enabled: true,
    mode: 'balanced',
    decay_half_life_days: 30,
    max_entropy_threshold: 4.8,
    turn_eval_interval: 50,
    deduplication_threshold: 0.95,
    e2ee_eval_location: 'server_distance_only',
  });
  const [loadingPolicy, setLoadingPolicy] = useState(false);
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  useEffect(() => {
    if (activeProject?.id) {
      setLoadingPolicy(true);
      fetchContextHealthPolicy(activeProject.id)
        .then((p) => setPolicy(p))
        .catch((err) => console.warn('Could not load policy:', err))
        .finally(() => setLoadingPolicy(false));
    }
  }, [activeProject?.id]);

  const handleCopyProjectID = () => {
    if (activeProject?.id) {
      navigator.clipboard.writeText(activeProject.id);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleSavePolicy = async () => {
    if (!activeProject?.id) return;
    setSavingPolicy(true);
    try {
      const updated = await updateContextHealthPolicy(activeProject.id, policy);
      setPolicy(updated);
      setSaveSuccess(true);
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (err: any) {
      alert(err.message || 'Failed to update context health policy');
    } finally {
      setSavingPolicy(false);
    }
  };

  const handleDelete = async () => {
    if (!activeProject) return;
    const confirmName = window.prompt(
      `Type "${activeProject.name}" to confirm safe soft deletion of this project:`
    );
    if (confirmName !== activeProject.name) {
      alert('Project name did not match. Deletion cancelled.');
      return;
    }

    setDeleting(true);
    try {
      await deleteProject(activeProject.id);
      navigate('/projects');
    } catch (err: any) {
      alert(err.message || 'Failed to delete project');
    } finally {
      setDeleting(false);
    }
  };

  if (!activeProject) {
    return <div className="text-slate-500 text-sm">No active project workspace selected.</div>;
  }

  return (
    <div className="max-w-4xl space-y-8 font-sans">
      <div>
        <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight">
          Project Settings
        </h1>
        <p className="text-xs text-slate-500 mt-1">
          Manage configuration and scoping settings for workspace <strong className="text-indigo-600">{activeProject.name}</strong>.
        </p>
      </div>

      {/* Project ID Card */}
      <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6 space-y-4">
        <h2 className="text-sm font-bold text-slate-900">Project Identification</h2>

        <div>
          <label className="block text-xs font-semibold uppercase text-slate-500 mb-1.5">
            Project UUID
          </label>
          <div className="flex items-center space-x-3">
            <div className="bg-slate-50 border border-slate-200 rounded-lg px-3.5 py-2 text-xs font-mono text-indigo-700 flex-1 flex items-center space-x-2">
              <ShieldCheck className="h-4 w-4 text-indigo-600 shrink-0" />
              <span>{activeProject.id}</span>
            </div>
            <button
              onClick={handleCopyProjectID}
              className="px-3.5 py-2 rounded-lg bg-white hover:bg-slate-50 text-xs font-semibold text-slate-700 border border-slate-200 transition-all flex items-center space-x-1.5 shrink-0 shadow-xs"
            >
              {copied ? <Check className="h-4 w-4 text-emerald-600" /> : <Copy className="h-4 w-4" />}
              <span>{copied ? 'Copied' : 'Copy ID'}</span>
            </button>
          </div>
          <p className="text-[11px] text-slate-500 mt-1.5">
            Pass this ID in your API header <code className="text-indigo-600 font-semibold bg-indigo-50 px-1 rounded">X-Project-ID</code> when authenticating via JWT.
          </p>
        </div>
      </div>

      {/* Context Health Engine Policy Card */}
      <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6 space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <Activity className="h-5 w-5 text-indigo-600" />
            <div>
              <h2 className="text-sm font-bold text-slate-900">Context Health Engine Policy</h2>
              <p className="text-xs text-slate-500">Configure memory noise filtering, fact supersedence DAGs, and time-decay settings.</p>
            </div>
          </div>

          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={policy.enabled}
              onChange={(e) => setPolicy({ ...policy, enabled: e.target.checked })}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-indigo-600"></div>
          </label>
        </div>

        {loadingPolicy ? (
          <div className="text-xs text-slate-500 py-4">Loading project policy...</div>
        ) : (
          <div className="space-y-6 pt-2 border-t border-slate-100">
            {/* Mode Selector */}
            <div>
              <label className="block text-xs font-semibold uppercase text-slate-500 mb-2">
                Operational Mode
              </label>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                {(['balanced', 'strict', 'cost_optimized', 'off'] as const).map((m) => (
                  <button
                    key={m}
                    type="button"
                    onClick={() => setPolicy({ ...policy, mode: m })}
                    className={`px-3 py-2.5 rounded-lg border text-xs font-semibold capitalize transition-all text-left ${
                      policy.mode === m
                        ? 'border-indigo-600 bg-indigo-50/50 text-indigo-700 ring-1 ring-indigo-600'
                        : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50'
                    }`}
                  >
                    <div>{m.replace('_', ' ')}</div>
                    <div className="text-[10px] font-normal text-slate-500 mt-0.5">
                      {m === 'balanced' && 'Micro-sweeps every 50 turns'}
                      {m === 'strict' && 'Aggressive sweeps every 25 turns'}
                      {m === 'cost_optimized' && '$0 LLM - Recency decay only'}
                      {m === 'off' && 'Health processing disabled'}
                    </div>
                  </button>
                ))}
              </div>
            </div>

            {/* Capability Warning Banner for Cost Optimized */}
            {policy.mode === 'cost_optimized' && (
              <div className="bg-amber-50 border border-amber-200 rounded-lg p-3.5 flex items-start space-x-2.5 text-xs text-amber-800">
                <Info className="h-4 w-4 text-amber-600 shrink-0 mt-0.5" />
                <div>
                  <strong className="font-semibold">Cost Optimized Capability Limitation:</strong> Background LLM contradiction workers are disabled under this mode. Fact supersedence DAGs will not be constructed. ContextKit will apply SQL-based recency decay only.
                </div>
              </div>
            )}

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
              {/* Decay Half-Life Slider */}
              <div>
                <div className="flex justify-between items-center mb-1.5">
                  <label className="text-xs font-semibold uppercase text-slate-500">
                    Recency Decay Half-Life
                  </label>
                  <span className="text-xs font-mono font-bold text-indigo-600">{policy.decay_half_life_days} days</span>
                </div>
                <input
                  type="range"
                  min="1"
                  max="180"
                  value={policy.decay_half_life_days}
                  onChange={(e) => setPolicy({ ...policy, decay_half_life_days: parseInt(e.target.value) || 30 })}
                  className="w-full h-1.5 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-indigo-600"
                />
                <p className="text-[11px] text-slate-500 mt-1">Half-life parameter (τ) used for SQL query-time recency weighting.</p>
              </div>

              {/* Turn Evaluation Interval */}
              <div>
                <label className="block text-xs font-semibold uppercase text-slate-500 mb-1.5">
                  Micro-Sweep Turn Interval
                </label>
                <select
                  value={policy.turn_eval_interval}
                  onChange={(e) => setPolicy({ ...policy, turn_eval_interval: parseInt(e.target.value) || 50 })}
                  className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-700 font-semibold focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value={25}>Every 25 turns (Strict)</option>
                  <option value={50}>Every 50 turns (Balanced Default)</option>
                  <option value={100}>Every 100 turns (Relaxed)</option>
                </select>
                <p className="text-[11px] text-slate-500 mt-1">Number of room turns before enqueuing a background contradiction sweep.</p>
              </div>
            </div>

            <div className="flex items-center justify-between pt-3 border-t border-slate-100">
              <div className="flex items-center space-x-2 text-xs text-slate-500">
                <Sliders className="h-4 w-4 text-slate-400" />
                <span>Max Entropy Sensitivity Threshold: <strong className="font-mono text-slate-700">{policy.max_entropy_threshold}</strong></span>
              </div>

              <button
                type="button"
                onClick={handleSavePolicy}
                disabled={savingPolicy}
                className="inline-flex items-center space-x-1.5 px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-700 text-xs font-semibold text-white transition-all shadow-xs disabled:opacity-50"
              >
                {saveSuccess ? <Check className="h-4 w-4" /> : <Save className="h-4 w-4" />}
                <span>{savingPolicy ? 'Saving...' : saveSuccess ? 'Saved!' : 'Save Policy Settings'}</span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Safe Project Deletion Card */}
      <div className="bg-rose-50/60 border border-rose-200 rounded-xl p-6 shadow-soft space-y-4">
        <div className="flex items-center space-x-3 text-rose-700">
          <AlertTriangle className="h-5 w-5" />
          <h2 className="text-sm font-bold">Safe Project Deletion</h2>
        </div>

        <p className="text-xs text-slate-600 leading-relaxed">
          Soft deleting a project hides it from your project hub and revokes all associated API key access.
          Associated data remains safely preserved in PostgreSQL under a soft-delete timestamp.
        </p>

        <div className="pt-2">
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="inline-flex items-center space-x-2 px-4 py-2.5 rounded-lg text-xs font-semibold text-white bg-rose-600 hover:bg-rose-700 transition-all shadow-sm disabled:opacity-50"
          >
            <Trash2 className="h-4 w-4" />
            <span>{deleting ? 'Deleting Project...' : 'Safe Delete Project'}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
