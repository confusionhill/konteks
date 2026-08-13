import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { ShieldCheck, Copy, Check, Trash2, AlertTriangle } from 'lucide-react';

export const Settings: React.FC = () => {
  const { activeProject, deleteProject } = useAuth();
  const [copied, setCopied] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const navigate = useNavigate();

  const handleCopyProjectID = () => {
    if (activeProject?.id) {
      navigator.clipboard.writeText(activeProject.id);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
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
    return <div className="text-slate-400 text-sm">No active project selected.</div>;
  }

  return (
    <div className="max-w-4xl space-y-8 font-sans">
      <div>
        <h1 className="text-2xl font-bold text-slate-100">Project Settings</h1>
        <p className="text-xs text-slate-400 mt-1">
          Manage configuration and scoping settings for project <strong className="text-indigo-300">{activeProject.name}</strong>.
        </p>
      </div>

      {/* Project ID Card */}
      <div className="bg-slate-900/60 backdrop-blur-xl border border-slate-800 rounded-2xl p-6 shadow-xl space-y-4">
        <h2 className="text-sm font-bold text-slate-200">Project Identification</h2>
        
        <div>
          <label className="block text-xs font-semibold uppercase text-slate-400 mb-1.5">
            Project UUID
          </label>
          <div className="flex items-center space-x-3">
            <div className="bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs font-mono text-indigo-300 flex-1 flex items-center space-x-2">
              <ShieldCheck className="h-4 w-4 text-indigo-400 shrink-0" />
              <span>{activeProject.id}</span>
            </div>
            <button
              onClick={handleCopyProjectID}
              className="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-slate-200 border border-slate-700 transition-all flex items-center space-x-1.5 shrink-0"
            >
              {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
              <span>{copied ? 'Copied' : 'Copy ID'}</span>
            </button>
          </div>
          <p className="text-[11px] text-slate-500 mt-1.5">
            Pass this ID in your API header <code className="text-indigo-300">X-Project-ID</code> when authenticating via JWT.
          </p>
        </div>
      </div>

      {/* Safe Project Deletion Card */}
      <div className="bg-red-950/20 border border-red-500/30 rounded-2xl p-6 shadow-xl space-y-4">
        <div className="flex items-center space-x-3 text-red-400">
          <AlertTriangle className="h-5 w-5" />
          <h2 className="text-sm font-bold">Safe Project Deletion</h2>
        </div>

        <p className="text-xs text-slate-400 leading-relaxed">
          Soft deleting a project hides it from your project hub and revokes all associated API key access.
          Associated data remains safely preserved in PostgreSQL under a soft-delete timestamp.
        </p>

        <div className="pt-2">
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="inline-flex items-center space-x-2 px-4 py-2.5 rounded-xl text-xs font-semibold text-white bg-red-600 hover:bg-red-500 transition-all shadow-lg shadow-red-600/20 disabled:opacity-50"
          >
            <Trash2 className="h-4 w-4" />
            <span>{deleting ? 'Deleting Project...' : 'Safe Delete Project'}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
