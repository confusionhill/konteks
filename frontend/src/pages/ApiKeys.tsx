import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { type APIKey, fetchAPIKeys, createAPIKey, revokeAPIKey } from '../api/dashboard';
import { Key, Plus, Copy, Check, Trash2, ShieldAlert, AlertTriangle } from 'lucide-react';

export const ApiKeys: React.FC = () => {
  const { activeProject } = useAuth();
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [keyName, setKeyName] = useState('');
  const [expiryOption, setExpiryOption] = useState<string>('never');
  const [creating, setCreating] = useState(false);

  // Success Key Modal State
  const [newRawKey, setNewRawKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const loadKeys = async () => {
    if (!activeProject) return;
    setLoading(true);
    setError(null);
    try {
      const data = await fetchAPIKeys(activeProject.id);
      setKeys(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load API keys');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadKeys();
  }, [activeProject?.id]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeProject) return;
    setCreating(true);
    try {
      let days: number | undefined = undefined;
      if (expiryOption === '7') days = 7;
      if (expiryOption === '30') days = 30;
      if (expiryOption === '90') days = 90;

      const result = await createAPIKey(activeProject.id, keyName || 'API Key', days);
      setNewRawKey(result.raw_key);
      setIsModalOpen(false);
      setKeyName('');
      setExpiryOption('never');
      loadKeys();
    } catch (err: any) {
      alert(err.message || 'Failed to create API key');
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (keyId: string) => {
    if (!activeProject) return;
    if (!window.confirm('Are you sure you want to revoke this API Key? Any SDK / Agent using it will lose access immediately.')) return;

    try {
      await revokeAPIKey(activeProject.id, keyId);
      loadKeys();
    } catch (err: any) {
      alert(err.message || 'Failed to revoke API key');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-6 font-sans">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-100">API Keys</h1>
          <p className="text-xs text-slate-400 mt-1">
            Manage secret keys for Python/Node.js SDKs and AI Agents to communicate with project <code className="bg-slate-800 text-indigo-300 px-1.5 py-0.5 rounded">{activeProject?.name}</code>.
          </p>
        </div>

        <button
          onClick={() => setIsModalOpen(true)}
          className="inline-flex items-center space-x-2 px-4 py-2 rounded-xl text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 transition-all shadow-lg shadow-indigo-600/25 shrink-0"
        >
          <Plus className="h-4 w-4" />
          <span>Create New API Key</span>
        </button>
      </div>

      {/* Main Keys List Card */}
      <div className="bg-slate-900/60 backdrop-blur-xl border border-slate-800 rounded-2xl p-6 shadow-xl">
        {loading ? (
          <div className="py-12 text-center text-slate-400 text-sm">Loading API keys...</div>
        ) : error ? (
          <div className="p-4 bg-red-950/60 border border-red-500/40 text-red-300 text-xs rounded-xl flex items-center space-x-2">
            <AlertTriangle className="h-4 w-4 text-red-400" />
            <span>{error}</span>
          </div>
        ) : keys.length === 0 ? (
          <div className="text-center py-12">
            <Key className="h-10 w-10 text-slate-600 mx-auto mb-3" />
            <h3 className="text-sm font-semibold text-slate-300">No API Keys Created</h3>
            <p className="text-xs text-slate-500 mt-1 mb-4">
              Create an API key to allow your AI agents and SDKs to connect to this project.
            </p>
            <button
              onClick={() => setIsModalOpen(true)}
              className="inline-flex items-center space-x-1.5 px-3.5 py-2 rounded-xl text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 transition-all"
            >
              <Plus className="h-3.5 w-3.5" />
              <span>Create API Key</span>
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-slate-800 text-[11px] font-semibold uppercase tracking-wider text-slate-400">
                  <th className="pb-3 px-3">Name</th>
                  <th className="pb-3 px-3">Key Prefix</th>
                  <th className="pb-3 px-3">Created</th>
                  <th className="pb-3 px-3">Last Used</th>
                  <th className="pb-3 px-3">Status / Expiry</th>
                  <th className="pb-3 px-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 text-xs">
                {keys.map((k) => {
                  const isRevoked = !!k.revoked_at;
                  const isExpired = k.expires_at && new Date(k.expires_at) < new Date();

                  return (
                    <tr key={k.id} className="hover:bg-slate-800/30 transition-colors">
                      <td className="py-3.5 px-3 font-semibold text-slate-200">{k.name}</td>
                      <td className="py-3.5 px-3">
                        <code className="bg-slate-950 px-2 py-1 rounded text-slate-300 font-mono text-[11px] border border-slate-800">
                          {k.key_prefix}
                        </code>
                      </td>
                      <td className="py-3.5 px-3 text-slate-400">
                        {new Date(k.created_at).toLocaleDateString()}
                      </td>
                      <td className="py-3.5 px-3 text-slate-400">
                        {k.last_used_at ? new Date(k.last_used_at).toLocaleString() : 'Never'}
                      </td>
                      <td className="py-3.5 px-3">
                        {isRevoked ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold bg-red-950 text-red-400 border border-red-800/50">
                            Revoked
                          </span>
                        ) : isExpired ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-950 text-amber-400 border border-amber-800/50">
                            Expired
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-950 text-emerald-400 border border-emerald-800/50">
                            {k.expires_at ? `Expires ${new Date(k.expires_at).toLocaleDateString()}` : 'Active (No Expiry)'}
                          </span>
                        )}
                      </td>
                      <td className="py-3.5 px-3 text-right">
                        {!isRevoked && (
                          <button
                            onClick={() => handleRevoke(k.id)}
                            className="text-slate-400 hover:text-red-400 p-1.5 rounded-lg hover:bg-red-950/40 transition-colors"
                            title="Revoke key"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create API Key Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-md w-full p-6 shadow-2xl">
            <h3 className="text-lg font-bold text-white mb-1">Create Project API Key</h3>
            <p className="text-xs text-slate-400 mb-6">
              Generate a secure token for backend server integrations.
            </p>

            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold uppercase text-slate-300 mb-1">
                  Key Name
                </label>
                <input
                  type="text"
                  required
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  placeholder="e.g. Production Python SDK Agent"
                  className="w-full px-3.5 py-2 bg-slate-950 border border-slate-800 rounded-xl text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase text-slate-300 mb-1">
                  Expiration
                </label>
                <select
                  value={expiryOption}
                  onChange={(e) => setExpiryOption(e.target.value)}
                  className="w-full px-3.5 py-2 bg-slate-950 border border-slate-800 rounded-xl text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="never">Never (Lifetime until revoked)</option>
                  <option value="7">7 Days</option>
                  <option value="30">30 Days</option>
                  <option value="90">90 Days</option>
                </select>
              </div>

              <div className="flex items-center justify-end space-x-3 pt-4 border-t border-slate-800">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-400 hover:text-slate-200 border border-slate-800 hover:border-slate-700"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 transition-all shadow-md shadow-indigo-600/20 disabled:opacity-50"
                >
                  {creating ? 'Generating...' : 'Generate API Key'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Raw Key Display Modal (Displayed ONCE) */}
      {newRawKey && (
        <div className="fixed inset-0 z-50 bg-slate-950/85 backdrop-blur-md flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-indigo-500/40 rounded-2xl max-w-lg w-full p-6 shadow-2xl relative">
            <div className="flex items-center space-x-3 text-emerald-400 mb-3">
              <ShieldAlert className="h-6 w-6" />
              <h3 className="text-lg font-bold text-white">Save Your API Key Now</h3>
            </div>

            <p className="text-xs text-slate-300 mb-4 leading-relaxed">
              Please copy and store this API key in a secure secret manager. For security, <strong className="text-indigo-300">it will never be displayed again</strong>.
            </p>

            <div className="bg-slate-950 border border-slate-800 rounded-xl p-3.5 flex items-center justify-between font-mono text-xs text-indigo-300 mb-6 break-all">
              <span>{newRawKey}</span>
              <button
                onClick={() => copyToClipboard(newRawKey)}
                className="ml-3 p-2 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 border border-indigo-500/30 transition-all shrink-0"
                title="Copy Key"
              >
                {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
              </button>
            </div>

            <div className="flex justify-end">
              <button
                onClick={() => setNewRawKey(null)}
                className="px-5 py-2.5 rounded-xl text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 transition-all shadow-lg shadow-indigo-600/25"
              >
                I Have Saved My API Key
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
