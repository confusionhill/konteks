import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth, type Project } from '../context/AuthContext';
import { FolderPlus, Layers, Search, ArrowRight, Trash2, Calendar, ShieldCheck, Sparkles } from 'lucide-react';

export const Projects: React.FC = () => {
  const { projects, selectProject, createProject, deleteProject, fetchProjects, user, logout } = useAuth();
  const [search, setSearch] = useState('');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newProjectName, setNewProjectName] = useState('');
  const [newProjectDesc, setNewProjectDesc] = useState('');
  const [loading, setLoading] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleSelectProject = (proj: Project) => {
    selectProject(proj);
    navigate('/dashboard');
  };

  const handleCreateProject = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newProjectName.trim()) return;
    setError(null);
    setLoading(true);
    try {
      await createProject(newProjectName.trim(), newProjectDesc.trim());
      setIsModalOpen(false);
      setNewProjectName('');
      setNewProjectDesc('');
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.message || 'Failed to create project');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteProject = async (e: React.MouseEvent, projectId: string) => {
    e.stopPropagation();
    if (!window.confirm('Are you sure you want to soft delete this project?')) return;
    setDeletingId(projectId);
    try {
      await deleteProject(projectId);
    } catch (err: any) {
      alert(err.message || 'Failed to delete project');
    } finally {
      setDeletingId(null);
    }
  };

  const filteredProjects = projects.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.description.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 font-sans relative overflow-hidden">
      {/* Top Background Glow */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full max-w-7xl h-96 bg-indigo-600/10 rounded-full blur-3xl pointer-events-none" />

      {/* Header Bar */}
      <header className="border-b border-slate-800/80 bg-slate-900/60 backdrop-blur-xl sticky top-0 z-30">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-md shadow-indigo-500/20">
              <Layers className="h-5 w-5 text-white" />
            </div>
            <span className="text-xl font-bold tracking-tight text-white">konteks</span>
            <span className="bg-slate-800 text-slate-300 text-xs px-2.5 py-0.5 rounded-full font-medium border border-slate-700">
              Projects Hub
            </span>
          </div>

          <div className="flex items-center space-x-4">
            <div className="text-right hidden sm:block">
              <div className="text-sm font-semibold text-slate-200">{user?.name}</div>
              <div className="text-xs text-slate-400">{user?.email}</div>
            </div>
            <button
              onClick={logout}
              className="text-xs font-semibold text-slate-400 hover:text-slate-200 px-3 py-1.5 rounded-lg border border-slate-800 hover:border-slate-700 transition-all"
            >
              Sign Out
            </button>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10 relative z-10">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-8">
          <div>
            <h1 className="text-3xl font-extrabold text-white tracking-tight">Select a Project</h1>
            <p className="text-slate-400 text-sm mt-1">
              Rooms, context states, and API keys are strictly scoped to your project workspace.
            </p>
          </div>

          <button
            onClick={() => setIsModalOpen(true)}
            className="inline-flex items-center space-x-2 px-4 py-2.5 rounded-xl font-semibold text-sm text-white bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 transition-all shadow-lg shadow-indigo-600/25 shrink-0"
          >
            <FolderPlus className="h-4 w-4" />
            <span>Create New Project</span>
          </button>
        </div>

        {/* Search Bar */}
        <div className="relative mb-8 max-w-md">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search projects..."
            className="w-full pl-10 pr-4 py-2.5 bg-slate-900/80 border border-slate-800 rounded-xl text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
          />
        </div>

        {/* Projects Grid */}
        {filteredProjects.length === 0 ? (
          <div className="text-center py-16 bg-slate-900/40 border border-slate-800/80 rounded-2xl p-8 backdrop-blur-sm">
            <div className="w-16 h-16 rounded-2xl bg-indigo-950/60 border border-indigo-500/30 text-indigo-400 flex items-center justify-center mx-auto mb-4">
              <Sparkles className="h-8 w-8" />
            </div>
            <h3 className="text-lg font-bold text-slate-200">No Projects Found</h3>
            <p className="text-slate-400 text-sm max-w-md mx-auto mt-1 mb-6">
              You haven't created any projects yet. Create your first project to start creating AI context rooms and managing API keys.
            </p>
            <button
              onClick={() => setIsModalOpen(true)}
              className="inline-flex items-center space-x-2 px-5 py-2.5 rounded-xl font-semibold text-sm text-white bg-indigo-600 hover:bg-indigo-500 transition-all shadow-lg shadow-indigo-600/25"
            >
              <FolderPlus className="h-4 w-4" />
              <span>Create Project</span>
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredProjects.map((proj) => (
              <div
                key={proj.id}
                onClick={() => handleSelectProject(proj)}
                className="group relative bg-slate-900/70 hover:bg-slate-900 border border-slate-800 hover:border-indigo-500/50 rounded-2xl p-6 transition-all duration-200 hover:shadow-xl hover:shadow-indigo-500/10 cursor-pointer flex flex-col justify-between"
              >
                <div>
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center space-x-3">
                      <div className="h-10 w-10 rounded-xl bg-indigo-950/80 border border-indigo-500/30 text-indigo-400 flex items-center justify-center group-hover:scale-105 transition-transform">
                        <Layers className="h-5 w-5" />
                      </div>
                      <div>
                        <h3 className="font-bold text-slate-100 group-hover:text-indigo-300 transition-colors line-clamp-1">
                          {proj.name}
                        </h3>
                        <span className="text-[11px] font-mono text-slate-400 flex items-center gap-1">
                          <ShieldCheck className="h-3 w-3 text-indigo-400" />
                          {proj.id.slice(0, 8)}...
                        </span>
                      </div>
                    </div>

                    <button
                      onClick={(e) => handleDeleteProject(e, proj.id)}
                      disabled={deletingId === proj.id}
                      className="text-slate-500 hover:text-red-400 p-1.5 rounded-lg hover:bg-red-950/40 border border-transparent hover:border-red-800/40 transition-all"
                      title="Soft delete project"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>

                  <p className="text-xs text-slate-400 line-clamp-2 min-h-[32px] mb-4">
                    {proj.description || 'No description provided.'}
                  </p>
                </div>

                <div className="pt-4 border-t border-slate-800/60 flex items-center justify-between text-xs text-slate-400">
                  <span className="flex items-center space-x-1">
                    <Calendar className="h-3.5 w-3.5" />
                    <span>{new Date(proj.created_at).toLocaleDateString()}</span>
                  </span>

                  <span className="inline-flex items-center space-x-1 font-semibold text-indigo-400 group-hover:translate-x-1 transition-transform">
                    <span>Enter Project</span>
                    <ArrowRight className="h-3.5 w-3.5" />
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {/* Create Project Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-md w-full p-6 shadow-2xl relative">
            <h3 className="text-xl font-bold text-white mb-1">Create New Project</h3>
            <p className="text-xs text-slate-400 mb-6">
              Projects isolate context rooms, state schemas, and API key credentials.
            </p>

            {error && (
              <div className="mb-4 text-xs bg-red-950/80 border border-red-500/50 text-red-300 p-3 rounded-xl">
                {error}
              </div>
            )}

            <form onSubmit={handleCreateProject} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold uppercase text-slate-300 mb-1">
                  Project Name *
                </label>
                <input
                  type="text"
                  required
                  value={newProjectName}
                  onChange={(e) => setNewProjectName(e.target.value)}
                  placeholder="e.g. Customer Support AI Bot"
                  className="w-full px-3.5 py-2.5 bg-slate-950 border border-slate-800 rounded-xl text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase text-slate-300 mb-1">
                  Description
                </label>
                <textarea
                  value={newProjectDesc}
                  onChange={(e) => setNewProjectDesc(e.target.value)}
                  placeholder="Brief description of your project..."
                  rows={3}
                  className="w-full px-3.5 py-2.5 bg-slate-950 border border-slate-800 rounded-xl text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
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
                  disabled={loading}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 transition-all shadow-md shadow-indigo-600/20 disabled:opacity-50"
                >
                  {loading ? 'Creating...' : 'Create Project'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
