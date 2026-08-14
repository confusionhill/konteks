import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import {
  FileText,
  Upload,
  Network,
  Brain,
  Folder,
  Trash2,
  Edit3,
  Search,
  AlertCircle,
} from 'lucide-react';
import type { Artifact, KnowledgeGraph } from '../api/artifacts';
import {
  fetchArtifacts,
  fetchArtifactDetail,
  fetchKnowledgeGraph,
  deleteArtifact,
} from '../api/artifacts';
import { useAuth } from '../context/AuthContext';
import { UploadArtifactModal } from '../components/UploadArtifactModal';
import { ArtifactEditorModal } from '../components/ArtifactEditorModal';
import { ArtifactDetailDrawer } from '../components/ArtifactDetailDrawer';
import { ArtifactsGraphView } from '../components/ArtifactsGraphView';
import { MemoriesInspector } from '../components/MemoriesInspector';

export const ArtifactsPage: React.FC = () => {
  const { projectId } = useParams<{ projectId: string }>();
  const { activeProject } = useAuth();
  const activeProjectId = projectId || activeProject?.id || 'b0000000-0000-0000-0000-000000000001';

  const [activeTab, setActiveTab] = useState<'explorer' | 'graph' | 'facts'>('explorer');
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [graph, setGraph] = useState<KnowledgeGraph | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [selectedGroup, setSelectedGroup] = useState('');

  // Modals & Drawers
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const [editingArtifact, setEditingArtifact] = useState<Artifact | null>(null);
  const [inspectingArtifact, setInspectingArtifact] = useState<Artifact | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [artRes, graphRes] = await Promise.all([
        fetchArtifacts(activeProjectId, selectedGroup || undefined),
        fetchKnowledgeGraph(activeProjectId),
      ]);
      setArtifacts(artRes.artifacts || []);
      setGraph(graphRes);
    } catch (err: any) {
      setError(err.message || 'Failed to load artifacts.');
    } finally {
      setLoading(false);
    }
  }, [activeProjectId, selectedGroup]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleInspect = async (art: Artifact) => {
    try {
      const fullArt = await fetchArtifactDetail(activeProjectId, art.id);
      setInspectingArtifact(fullArt);
    } catch {
      setInspectingArtifact(art);
    }
  };

  const handleDelete = async (artifactId: string) => {
    if (!window.confirm('Are you sure you want to delete this memory artifact?')) return;
    try {
      await deleteArtifact(activeProjectId, artifactId);
      if (inspectingArtifact?.id === artifactId) setInspectingArtifact(null);
      loadData();
    } catch (err: any) {
      alert(err.message || 'Failed to delete artifact');
    }
  };

  const filteredArtifacts = artifacts.filter((a) =>
    a.file_path.toLowerCase().includes(search.toLowerCase()) ||
    a.title.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6 font-sans">
      {/* Header Bar */}
      <div className="bg-white rounded-2xl p-6 border border-slate-200/80 shadow-soft flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <span className="text-xs font-mono font-bold uppercase tracking-wider text-indigo-600 bg-indigo-50 px-2.5 py-1 rounded-full border border-indigo-100 inline-flex items-center gap-1.5">
            <Brain className="w-3.5 h-3.5" />
            <span>Memory & Context Infrastructure</span>
          </span>
          <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight font-sans mt-2">
            Project Core Memory Artifacts
          </h1>
          <p className="text-sm font-medium text-slate-500 mt-1">
            Indexed Markdown documents (.md) powering durable long-form knowledge and graph retrieval
          </p>
        </div>

        <button
          onClick={() => setIsUploadOpen(true)}
          className="px-4 py-2.5 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-700 hover:to-violet-700 text-white font-semibold text-xs rounded-xl shadow-button transition-all duration-150 flex items-center justify-center gap-2 shrink-0"
        >
          <Upload className="w-4 h-4" />
          <span>Upload Markdown Memory</span>
        </button>
      </div>

      {error && (
        <div className="bg-rose-50 border border-rose-200 text-rose-800 rounded-xl p-4 flex items-center space-x-3 text-sm">
          <AlertCircle className="w-5 h-5 text-rose-600 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Tabs Bar */}
      <div className="flex items-center space-x-2 border-b border-slate-200 pb-px">
        <button
          onClick={() => setActiveTab('explorer')}
          className={`px-4 py-2.5 font-semibold text-xs rounded-t-xl transition-all duration-150 flex items-center space-x-2 border-b-2 ${
            activeTab === 'explorer'
              ? 'border-indigo-600 text-indigo-600 bg-indigo-50/50'
              : 'border-transparent text-slate-500 hover:text-slate-800 hover:bg-slate-100/50'
          }`}
        >
          <FileText className="w-4 h-4" />
          <span>File Explorer & List ({artifacts.length})</span>
        </button>

        <button
          onClick={() => setActiveTab('graph')}
          className={`px-4 py-2.5 font-semibold text-xs rounded-t-xl transition-all duration-150 flex items-center space-x-2 border-b-2 ${
            activeTab === 'graph'
              ? 'border-indigo-600 text-indigo-600 bg-indigo-50/50'
              : 'border-transparent text-slate-500 hover:text-slate-800 hover:bg-slate-100/50'
          }`}
        >
          <Network className="w-4 h-4" />
          <span>Knowledge Graph Visualizer</span>
        </button>

        <button
          onClick={() => setActiveTab('facts')}
          className={`px-4 py-2.5 font-semibold text-xs rounded-t-xl transition-all duration-150 flex items-center space-x-2 border-b-2 ${
            activeTab === 'facts'
              ? 'border-indigo-600 text-indigo-600 bg-indigo-50/50'
              : 'border-transparent text-slate-500 hover:text-slate-800 hover:bg-slate-100/50'
          }`}
        >
          <Brain className="w-4 h-4" />
          <span>Short Fact Memories</span>
        </button>
      </div>

      {/* Tab 1: File Explorer & Table */}
      {activeTab === 'explorer' && (
        <div className="space-y-4">
          <div className="bg-white p-4 rounded-2xl border border-slate-200/80 shadow-soft flex items-center justify-between gap-4">
            <div className="relative flex-1">
              <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search file path or title..."
                className="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-medium text-slate-900 focus:outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100"
              />
            </div>
            <select
              value={selectedGroup}
              onChange={(e) => setSelectedGroup(e.target.value)}
              className="bg-slate-50 border border-slate-200 text-slate-700 text-xs font-medium rounded-xl px-3 py-2 focus:outline-none focus:border-indigo-500"
            >
              <option value="">All Group Folders</option>
              <option value="security">security</option>
              <option value="rules">rules</option>
            </select>
          </div>

          {loading ? (
            <div className="p-12 text-center text-sm font-medium text-slate-500 bg-white rounded-2xl border border-slate-200/80 shadow-soft">
              Loading memory artifacts...
            </div>
          ) : filteredArtifacts.length === 0 ? (
            <div className="bg-white rounded-2xl border border-slate-200/80 shadow-soft p-12 text-center">
              <div className="w-12 h-12 rounded-2xl bg-indigo-50 text-indigo-600 border border-indigo-100 flex items-center justify-center mx-auto mb-3">
                <Folder className="w-6 h-6" />
              </div>
              <h3 className="text-base font-bold text-slate-900">No Markdown Artifacts Found</h3>
              <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-5">
                Upload your project rules, API specs, or system documentation to enable deep memory retrieval.
              </p>
              <button
                onClick={() => setIsUploadOpen(true)}
                className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-button transition-colors"
              >
                Upload First .md Artifact
              </button>
            </div>
          ) : (
            <div className="bg-white border border-slate-200/80 rounded-2xl overflow-hidden shadow-soft">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 bg-slate-50 text-xs font-bold text-slate-500 uppercase tracking-wider">
                    <th className="p-4">File Path</th>
                    <th className="p-4">Title</th>
                    <th className="p-4">Group</th>
                    <th className="p-4">Chunks</th>
                    <th className="p-4">Last Updated</th>
                    <th className="p-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 text-xs text-slate-700">
                  {filteredArtifacts.map((art) => (
                    <tr
                      key={art.id}
                      className="hover:bg-slate-50/60 transition-colors cursor-pointer"
                      onClick={() => handleInspect(art)}
                    >
                      <td className="p-4 font-mono font-bold text-indigo-600 flex items-center gap-2">
                        <FileText className="w-4 h-4 text-indigo-500 shrink-0" />
                        <span>{art.file_path}</span>
                      </td>
                      <td className="p-4 font-semibold text-slate-900">{art.title}</td>
                      <td className="p-4">
                        <span className="px-2.5 py-0.5 rounded-full bg-slate-100 text-slate-700 text-[11px] font-medium border border-slate-200">
                          {art.group_name || 'General'}
                        </span>
                      </td>
                      <td className="p-4 font-mono text-slate-500">{art.chunks_count || 1} chunks</td>
                      <td className="p-4 text-slate-500">{new Date(art.updated_at).toLocaleDateString()}</td>
                      <td className="p-4 text-right" onClick={(e) => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-1.5">
                          <button
                            onClick={() => setEditingArtifact(art)}
                            className="p-1.5 bg-slate-100 hover:bg-indigo-50 text-slate-600 hover:text-indigo-600 rounded-lg transition-colors"
                            title="Edit Artifact"
                          >
                            <Edit3 className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleDelete(art.id)}
                            className="p-1.5 bg-rose-50 hover:bg-rose-100 text-rose-600 rounded-lg transition-colors"
                            title="Delete Artifact"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Tab 2: Knowledge Graph Visualizer */}
      {activeTab === 'graph' && (
        <ArtifactsGraphView
          graph={graph}
          artifacts={artifacts}
          onSelectArtifact={handleInspect}
        />
      )}

      {/* Tab 3: Short Fact Memories */}
      {activeTab === 'facts' && <MemoriesInspector memories={[]} loading={false} />}

      {/* Modals & Drawers */}
      <UploadArtifactModal
        projectId={activeProjectId}
        isOpen={isUploadOpen}
        onClose={() => setIsUploadOpen(false)}
        onSuccess={loadData}
      />

      <ArtifactEditorModal
        projectId={activeProjectId}
        artifact={editingArtifact}
        isOpen={!!editingArtifact}
        onClose={() => setEditingArtifact(null)}
        onSuccess={loadData}
      />

      <ArtifactDetailDrawer
        artifact={inspectingArtifact}
        graph={graph}
        isOpen={!!inspectingArtifact}
        onClose={() => setInspectingArtifact(null)}
        onEdit={(art) => {
          setInspectingArtifact(null);
          setEditingArtifact(art);
        }}
        onDelete={handleDelete}
      />
    </div>
  );
};
