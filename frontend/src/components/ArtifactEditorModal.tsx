import React, { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { X, Save, AlertCircle, Edit3 } from 'lucide-react';
import type { Artifact } from '../api/artifacts';
import { fetchArtifactDetail, updateArtifact } from '../api/artifacts';

interface ArtifactEditorModalProps {
  projectId: string;
  artifact: Artifact | null;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const ArtifactEditorModal: React.FC<ArtifactEditorModalProps> = ({
  projectId,
  artifact,
  isOpen,
  onClose,
  onSuccess,
}) => {
  const [title, setTitle] = useState('');
  const [groupName, setGroupName] = useState('');
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(false);
  const [fetchingContent, setFetchingContent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (artifact) {
      setTitle(artifact.title || '');
      setGroupName(artifact.group_name || '');

      if (artifact.content) {
        setContent(artifact.content);
      } else {
        setFetchingContent(true);
        fetchArtifactDetail(projectId, artifact.id)
          .then((fullArt) => {
            setTitle(fullArt.title || artifact.title || '');
            setGroupName(fullArt.group_name || artifact.group_name || '');
            setContent(fullArt.content || '');
          })
          .catch((err) => {
            setError('Failed to load artifact content: ' + (err.message || 'Unknown error'));
          })
          .finally(() => {
            setFetchingContent(false);
          });
      }
    }
  }, [artifact, projectId]);

  if (!isOpen || !artifact) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!content.trim()) {
      setError('Content cannot be empty.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await updateArtifact(projectId, artifact.id, {
        title: title.trim(),
        group_name: groupName.trim() || undefined,
        content: content,
      });
      onSuccess();
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to update artifact.');
    } finally {
      setLoading(false);
    }
  };

  return createPortal(
    <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4 overflow-y-auto font-sans">
      <div className="bg-white border border-slate-200/80 rounded-2xl max-w-4xl w-full p-6 shadow-lifted relative my-8">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-slate-400 hover:text-slate-600 transition-colors p-1"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-6">
          <div className="p-2.5 rounded-xl bg-indigo-50 text-indigo-600 border border-indigo-100">
            <Edit3 className="w-6 h-6" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-slate-900">Edit Markdown Artifact</h2>
            <p className="text-xs font-mono font-bold text-indigo-600 mt-0.5">{artifact.file_path}</p>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-3.5 rounded-xl bg-rose-50 border border-rose-200 text-rose-800 text-xs flex items-center gap-2 font-medium">
            <AlertCircle className="w-4 h-4 shrink-0 text-rose-600" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Title</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-medium text-slate-900 focus:outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Group / Folder Tag</label>
              <input
                type="text"
                value={groupName}
                onChange={(e) => setGroupName(e.target.value)}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-medium text-slate-900 focus:outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100"
              />
            </div>
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-xs font-semibold text-slate-700">Markdown Content</label>
              {fetchingContent && <span className="text-xs font-semibold text-indigo-600 animate-pulse">Loading content...</span>}
            </div>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              rows={14}
              placeholder={fetchingContent ? 'Loading markdown content...' : 'Markdown content...'}
              disabled={fetchingContent}
              className="w-full p-3 bg-slate-50 border border-slate-200 rounded-xl text-xs font-mono text-slate-900 focus:outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100 disabled:opacity-60"
              required
            />
          </div>

          <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-xl text-xs font-semibold transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-button transition-colors flex items-center gap-2 disabled:opacity-50"
            >
              {loading ? (
                <span>Re-indexing...</span>
              ) : (
                <>
                  <Save className="w-4 h-4" />
                  <span>Save & Re-index</span>
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>,
    document.body
  );
};
