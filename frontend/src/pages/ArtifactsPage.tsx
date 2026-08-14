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
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header Bar */}
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div>
          <div className="flex items-center gap-2 text-xs font-semibold text-indigo-400 uppercase tracking-wider mb-1">
            <Brain className="w-4 h-4" />
            <span>Memory & Context Infrastructure</span>
          </div>
          <h1 className="text-2xl font-bold text-slate-100">Project Core Memory Artifacts</h1>
          <p className="text-xs text-slate-400 mt-1">
            Indexed Markdown documents (.md) powering durable long-form knowledge and graph retrieval
          </p>
        </div>

        <button
          onClick={() => setIsUploadOpen(true)}
          className="px-4 py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs rounded-xl transition-all shadow-lg shadow-indigo-600/20 flex items-center gap-2"
        >
          <Upload className="w-4 h-4" />
          <span>Upload Markdown Memory</span>
        </button>
      </div>

      {error && (
        <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-xl text-red-400 text-sm flex items-center gap-2">
          <AlertCircle className="w-5 h-5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Tabs */}
      <div className="flex items-center gap-2 border-b border-slate-800">
        <button
          onClick={() => setActiveTab('explorer')}
          className={`px-4 py-2.5 font-medium text-xs rounded-t-lg transition-colors flex items-center gap-2 border-b-2 ${
            activeTab === 'explorer'
              ? 'border-indigo-500 text-indigo-400 bg-slate-900/50'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <FileText className="w-4 h-4" />
          <span>File Explorer & List ({artifacts.length})</span>
        </button>

        <button
          onClick={() => setActiveTab('graph')}
          className={`px-4 py-2.5 font-medium text-xs rounded-t-lg transition-colors flex items-center gap-2 border-b-2 ${
            activeTab === 'graph'
              ? 'border-indigo-500 text-indigo-400 bg-slate-900/50'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Network className="w-4 h-4" />
          <span>Knowledge Graph Visualizer</span>
        </button>

        <button
          onClick={() => setActiveTab('facts')}
          className={`px-4 py-2.5 font-medium text-xs rounded-t-lg transition-colors flex items-center gap-2 border-b-2 ${
            activeTab === 'facts'
              ? 'border-indigo-500 text-indigo-400 bg-slate-900/50'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Brain className="w-4 h-4" />
          <span>Short Fact Memories</span>
        </button>
      </div>

      {/* Tab 1: File Explorer & Table */}
      {activeTab === 'explorer' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-4 bg-slate-900/60 p-4 rounded-xl border border-slate-800">
            <div className="relative flex-1">
              <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search file path or title..."
                className="w-full pl-9 pr-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-sm text-slate-200 focus:outline-none focus:border-indigo-500"
              />
            </div>
            <select
              value={selectedGroup}
              onChange={(e) => setSelectedGroup(e.target.value)}
              className="bg-slate-950 border border-slate-800 text-slate-300 text-xs rounded-lg px-3 py-2 focus:outline-none focus:border-indigo-500"
            >
              <option value="">All Group Folders</option>
              <option value="security">security</option>
              <option value="rules">rules</option>
            </select>
          </div>

          {loading ? (
            <div className="p-12 text-center text-sm text-slate-400">Loading memory artifacts...</div>
          ) : filteredArtifacts.length === 0 ? (
            <div className="p-12 text-center bg-slate-950/40 rounded-xl border border-slate-800">
              <Folder className="w-12 h-12 text-slate-600 mx-auto mb-2" />
              <h3 className="text-base font-semibold text-slate-300">No Markdown Artifacts Found</h3>
              <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-4">
                Upload your project rules, API specs, or system documentation to enable deep memory retrieval.
              </p>
              <button
                onClick={() => setIsUploadOpen(true)}
                className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-medium"
              >
                Upload First .md Artifact
              </button>
            </div>
          ) : (
            <div className="bg-slate-900/70 border border-slate-800 rounded-xl overflow-hidden shadow-xl">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-slate-800 bg-slate-950/60 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                    <th className="p-4">File Path</th>
                    <th className="p-4">Title</th>
                    <th className="p-4">Group</th>
                    <th className="p-4">Chunks</th>
                    <th className="p-4">Last Updated</th>
                    <th className="p-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60 text-xs text-slate-300">
                  {filteredArtifacts.map((art) => (
                    <tr
                      key={art.id}
                      className="hover:bg-slate-800/40 transition-colors cursor-pointer"
                      onClick={() => handleInspect(art)}
                    >
                      <td className="p-4 font-mono font-medium text-indigo-400 flex items-center gap-2">
                        <FileText className="w-4 h-4 text-indigo-500 shrink-0" />
                        <span>{art.file_path}</span>
                      </td>
                      <td className="p-4 font-semibold text-slate-200">{art.title}</td>
                      <td className="p-4">
                        <span className="px-2.5 py-0.5 rounded-full bg-slate-800 text-slate-300 text-[11px] border border-slate-700">
                          {art.group_name || 'General'}
                        </span>
                      </td>
                      <td className="p-4 font-mono text-slate-400">{art.chunks_count || 1} chunks</td>
                      <td className="p-4 text-slate-400">{new Date(art.updated_at).toLocaleDateString()}</td>
                      <td className="p-4 text-right" onClick={(e) => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => setEditingArtifact(art)}
                            className="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-md transition-colors"
                            title="Edit Artifact"
                          >
                            <Edit3 className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => handleDelete(art.id)}
                            className="p-1.5 bg-red-500/10 hover:bg-red-500/20 text-red-400 rounded-md transition-colors"
                            title="Delete Artifact"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
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
