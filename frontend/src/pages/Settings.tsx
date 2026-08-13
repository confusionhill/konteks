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
