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
          <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight">API Keys</h1>
          <p className="text-xs text-slate-500 mt-1">
            Manage secret keys for Python/Node.js SDKs and AI Agents to communicate with workspace <code className="bg-indigo-50 text-indigo-700 px-1.5 py-0.5 rounded font-semibold border border-indigo-100">{activeProject?.name}</code>.
          </p>
        </div>

        <button
          onClick={() => setIsModalOpen(true)}
          className="inline-flex items-center space-x-2 px-4 py-2 rounded-lg text-xs font-semibold text-white bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 transition-all shadow-button shrink-0"
        >
          <Plus className="h-4 w-4" />
          <span>Create New API Key</span>
        </button>
      </div>

      {/* Main Keys List Card */}
      <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6">
        {loading ? (
          <div className="py-12 text-center text-slate-400 text-sm">Loading API keys...</div>
        ) : error ? (
          <div className="p-4 bg-rose-50 border border-rose-200 text-rose-800 text-xs rounded-lg flex items-center space-x-2">
            <AlertTriangle className="h-4 w-4 text-rose-600" />
            <span>{error}</span>
          </div>
        ) : keys.length === 0 ? (
          <div className="text-center py-12">
            <div className="w-12 h-12 rounded-xl bg-indigo-50 text-indigo-600 flex items-center justify-center mx-auto mb-3">
              <Key className="h-6 w-6" />
            </div>
            <h3 className="text-sm font-bold text-slate-900">No API Keys Created</h3>
            <p className="text-xs text-slate-500 mt-1 mb-4">
              Create an API key to allow your AI agents and SDKs to connect to this workspace.
            </p>
            <button
              onClick={() => setIsModalOpen(true)}
              className="inline-flex items-center space-x-1.5 px-4 py-2 rounded-lg text-xs font-semibold text-white bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 transition-all shadow-button"
            >
              <Plus className="h-3.5 w-3.5" />
              <span>Create API Key</span>
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50/80 border-b border-slate-200 text-xs font-semibold uppercase tracking-wider text-slate-500">
                  <th className="py-3.5 px-4">Name</th>
                  <th className="py-3.5 px-4">Key Prefix</th>
                  <th className="py-3.5 px-4">Created</th>
                  <th className="py-3.5 px-4">Last Used</th>
                  <th className="py-3.5 px-4">Status / Expiry</th>
                  <th className="py-3.5 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-xs">
                {keys.map((k) => {
                  const isRevoked = !!k.revoked_at;
                  const isExpired = k.expires_at && new Date(k.expires_at) < new Date();

                  return (
                    <tr key={k.id} className="hover:bg-indigo-50/30 transition-colors">
                      <td className="py-3.5 px-4 font-semibold text-slate-900">{k.name}</td>
                      <td className="py-3.5 px-4">
                        <code className="bg-slate-100 px-2 py-1 rounded text-slate-800 font-mono text-[11px] border border-slate-200">
                          {k.key_prefix}
                        </code>
                      </td>
                      <td className="py-3.5 px-4 text-slate-500">
                        {new Date(k.created_at).toLocaleDateString()}
                      </td>
                      <td className="py-3.5 px-4 text-slate-500">
                        {k.last_used_at ? new Date(k.last_used_at).toLocaleString() : 'Never'}
                      </td>
                      <td className="py-3.5 px-4">
                        {isRevoked ? (
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-rose-50 text-rose-700 border border-rose-200">
                            Revoked
                          </span>
                        ) : isExpired ? (
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-amber-50 text-amber-700 border border-amber-200">
                            Expired
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-emerald-50 text-emerald-700 border border-emerald-200">
                            {k.expires_at ? `Expires ${new Date(k.expires_at).toLocaleDateString()}` : 'Active (No Expiry)'}
                          </span>
                        )}
                      </td>
                      <td className="py-3.5 px-4 text-right">
                        {!isRevoked && (
                          <button
                            onClick={() => handleRevoke(k.id)}
                            className="text-slate-400 hover:text-rose-600 p-1.5 rounded-lg hover:bg-rose-50 transition-colors"
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
        <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-white border border-slate-200 rounded-xl max-w-md w-full p-6 shadow-lifted">
            <h3 className="text-lg font-bold text-slate-900 mb-1">Create Project API Key</h3>
            <p className="text-xs text-slate-500 mb-6">
              Generate a secure token for backend server integrations.
            </p>

            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Key Name
                </label>
                <input
                  type="text"
                  required
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  placeholder="e.g. Production Python SDK Agent"
                  className="w-full px-3.5 py-2 bg-white border border-slate-200 rounded-lg text-sm text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Expiration
                </label>
                <select
                  value={expiryOption}
                  onChange={(e) => setExpiryOption(e.target.value)}
                  className="w-full px-3.5 py-2 bg-white border border-slate-200 rounded-lg text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="never">Never (Lifetime until revoked)</option>
                  <option value="7">7 Days</option>
                  <option value="30">30 Days</option>
                  <option value="90">90 Days</option>
                </select>
              </div>

              <div className="flex items-center justify-end space-x-3 pt-4 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-600 hover:text-slate-900 border border-slate-200 hover:bg-slate-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="px-4 py-2 rounded-lg text-xs font-semibold text-white bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 transition-all shadow-button disabled:opacity-50"
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
        <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-white border border-indigo-200 rounded-xl max-w-lg w-full p-6 shadow-lifted relative">
            <div className="flex items-center space-x-3 text-indigo-600 mb-3">
              <ShieldAlert className="h-6 w-6" />
              <h3 className="text-lg font-bold text-slate-900">Save Your API Key Now</h3>
            </div>

            <p className="text-xs text-slate-600 mb-4 leading-relaxed">
              Please copy and store this API key in a secure secret manager. For security, <strong className="text-indigo-600">it will never be displayed again</strong>.
            </p>

            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3.5 flex items-center justify-between font-mono text-xs text-indigo-700 mb-6 break-all">
              <span>{newRawKey}</span>
              <button
                onClick={() => copyToClipboard(newRawKey)}
                className="ml-3 p-2 rounded-lg bg-indigo-50 hover:bg-indigo-100 text-indigo-700 border border-indigo-200 transition-all shrink-0"
                title="Copy Key"
              >
                {copied ? <Check className="h-4 w-4 text-emerald-600" /> : <Copy className="h-4 w-4" />}
              </button>
            </div>

            <div className="flex justify-end">
              <button
                onClick={() => setNewRawKey(null)}
                className="px-5 py-2.5 rounded-lg text-xs font-semibold text-white bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 transition-all shadow-button"
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
