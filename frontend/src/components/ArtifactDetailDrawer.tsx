import React from 'react';
import { X, Edit3, Trash2, Link as LinkIcon, Database, HardDrive, Calendar } from 'lucide-react';
import type { Artifact, KnowledgeGraph } from '../api/artifacts';

interface ArtifactDetailDrawerProps {
  artifact: Artifact | null;
  graph: KnowledgeGraph | null;
  isOpen: boolean;
  onClose: () => void;
  onEdit: (artifact: Artifact) => void;
  onDelete: (artifactId: string) => void;
}

export const ArtifactDetailDrawer: React.FC<ArtifactDetailDrawerProps> = ({
  artifact,
  graph,
  isOpen,
  onClose,
  onEdit,
  onDelete,
}) => {
  if (!isOpen || !artifact) return null;

  // Find graph connections
  const outgoingLinks = graph?.edges.filter((e) => e.source === artifact.id) || [];
  const incomingLinks = graph?.edges.filter((e) => e.target === artifact.id) || [];

  return (
    <div className="fixed inset-y-0 right-0 z-50 w-full max-w-xl bg-slate-900 border-l border-slate-800 shadow-2xl flex flex-col">
      <div className="p-6 border-b border-slate-800 flex items-center justify-between">
        <div>
          <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            {artifact.group_name || 'General Group'}
          </span>
          <h2 className="text-xl font-bold text-slate-100 mt-2">{artifact.title}</h2>
          <p className="text-xs font-mono text-slate-400 mt-0.5">{artifact.file_path}</p>
        </div>
        <button onClick={onClose} className="text-slate-400 hover:text-slate-200">
          <X className="w-5 h-5" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Quick Actions */}
        <div className="flex items-center gap-3">
          <button
            onClick={() => onEdit(artifact)}
            className="flex-1 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-medium flex items-center justify-center gap-2 transition-colors"
          >
            <Edit3 className="w-4 h-4" />
            <span>Edit Artifact</span>
          </button>
          <button
            onClick={() => onDelete(artifact.id)}
            className="px-4 py-2 bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 rounded-lg text-xs font-medium flex items-center justify-center gap-2 transition-colors"
          >
            <Trash2 className="w-4 h-4" />
            <span>Delete</span>
          </button>
        </div>

        {/* Metadata Details */}
        <div className="bg-slate-950/60 rounded-xl p-4 border border-slate-800/80 grid grid-cols-2 gap-4 text-xs">
          <div className="flex items-center gap-2 text-slate-300">
            <Database className="w-4 h-4 text-indigo-400 shrink-0" />
            <div>
              <p className="text-slate-500">Indexed Chunks</p>
              <p className="font-semibold text-slate-200">{artifact.chunks_count || 0} chunks</p>
            </div>
          </div>

          <div className="flex items-center gap-2 text-slate-300">
            <HardDrive className="w-4 h-4 text-emerald-400 shrink-0" />
            <div>
              <p className="text-slate-500">S3 Key</p>
              <p className="font-mono text-slate-300 truncate max-w-[140px]">{artifact.s3_key || 'Stored'}</p>
            </div>
          </div>

          <div className="flex items-center gap-2 text-slate-300 col-span-2">
            <Calendar className="w-4 h-4 text-amber-400 shrink-0" />
            <div>
              <p className="text-slate-500">Last Updated</p>
              <p className="font-medium text-slate-300">{new Date(artifact.updated_at).toLocaleString()}</p>
            </div>
          </div>
        </div>

        {/* Knowledge Graph Connections */}
        <div>
          <h3 className="text-xs font-semibold text-slate-300 uppercase tracking-wider mb-3 flex items-center gap-2">
            <LinkIcon className="w-4 h-4 text-indigo-400" />
            <span>Knowledge Reference Links ({outgoingLinks.length + incomingLinks.length})</span>
          </h3>

          {outgoingLinks.length === 0 && incomingLinks.length === 0 ? (
            <p className="text-xs text-slate-500 italic bg-slate-950/40 p-3 rounded-lg border border-slate-800/50">
              No graph reference links detected. Add links like <span className="font-mono text-indigo-400">[Link](docs/target.md)</span> or <span className="font-mono text-indigo-400">[[target.md]]</span> in your markdown.
            </p>
          ) : (
            <div className="space-y-2">
              {outgoingLinks.map((edge, i) => {
                const targetNode = graph?.nodes.find((n) => n.id === edge.target);
                return (
                  <div key={i} className="flex items-center justify-between p-2.5 bg-slate-950 rounded-lg text-xs border border-slate-800">
                    <span className="text-slate-400">Refers to ➔</span>
                    <span className="font-mono text-indigo-300 font-medium">{targetNode?.file_path || edge.target}</span>
                    <span className="text-[10px] px-2 py-0.5 rounded bg-slate-800 text-slate-400">{edge.relationship}</span>
                  </div>
                );
              })}

              {incomingLinks.map((edge, i) => {
                const sourceNode = graph?.nodes.find((n) => n.id === edge.source);
                return (
                  <div key={i} className="flex items-center justify-between p-2.5 bg-slate-950 rounded-lg text-xs border border-slate-800">
                    <span className="text-slate-400">Referenced by ⬅</span>
                    <span className="font-mono text-indigo-300 font-medium">{sourceNode?.file_path || edge.source}</span>
                    <span className="text-[10px] px-2 py-0.5 rounded bg-slate-800 text-slate-400">{edge.relationship}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Content Preview */}
        <div>
          <h3 className="text-xs font-semibold text-slate-300 uppercase tracking-wider mb-2">Raw Markdown Content</h3>
          <pre className="p-4 bg-slate-950 rounded-xl text-xs font-mono text-slate-300 border border-slate-800 overflow-x-auto whitespace-pre-wrap max-h-96">
            {artifact.content || 'Content stored in S3.'}
          </pre>
        </div>
      </div>
    </div>
  );
};
