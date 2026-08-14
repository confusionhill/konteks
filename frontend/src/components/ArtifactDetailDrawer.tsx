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
  const outgoingLinks = (graph?.edges || []).filter((e) => e.source === artifact.id);
  const incomingLinks = (graph?.edges || []).filter((e) => e.target === artifact.id);

  return (
    <div
      className="fixed inset-0 z-[100] flex justify-end bg-slate-900/40 backdrop-blur-sm font-sans transition-opacity"
      onClick={onClose}
    >
      <div
        className="w-full max-w-xl bg-white h-full border-l border-slate-200 shadow-2xl flex flex-col relative overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header Bar */}
        <div className="p-6 border-b border-slate-100 flex items-start justify-between bg-white shrink-0">
          <div>
            <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-indigo-50 text-indigo-700 border border-indigo-100 inline-block">
              {artifact.group_name || 'General Group'}
            </span>
            <h2 className="text-xl font-bold text-slate-900 mt-2.5">{artifact.title}</h2>
            <p className="text-xs font-mono font-bold text-indigo-600 mt-0.5">{artifact.file_path}</p>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg hover:bg-slate-100 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {/* Quick Actions */}
          <div className="flex items-center gap-3">
            <button
              onClick={() => onEdit(artifact)}
              className="flex-1 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-button flex items-center justify-center gap-2 transition-colors"
            >
              <Edit3 className="w-4 h-4" />
              <span>Edit Artifact</span>
            </button>
            <button
              onClick={() => onDelete(artifact.id)}
              className="px-4 py-2.5 bg-rose-50 hover:bg-rose-100 text-rose-600 border border-rose-200 rounded-xl text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              <span>Delete</span>
            </button>
          </div>

          {/* Metadata Details */}
          <div className="bg-slate-50 rounded-2xl p-4 border border-slate-200/80 grid grid-cols-2 gap-4 text-xs">
            <div className="flex items-center gap-2.5 text-slate-700">
              <div className="p-2 rounded-lg bg-indigo-50 text-indigo-600 border border-indigo-100 shrink-0">
                <Database className="w-4 h-4" />
              </div>
              <div>
                <p className="text-slate-500 font-medium">Indexed Chunks</p>
                <p className="font-bold text-slate-900">{artifact.chunks_count || 0} chunks</p>
              </div>
            </div>

            <div className="flex items-center gap-2.5 text-slate-700">
              <div className="p-2 rounded-lg bg-emerald-50 text-emerald-600 border border-emerald-100 shrink-0">
                <HardDrive className="w-4 h-4" />
              </div>
              <div className="overflow-hidden">
                <p className="text-slate-500 font-medium">S3 Key</p>
                <p className="font-mono text-slate-800 font-semibold truncate max-w-[140px]" title={artifact.s3_key}>
                  {artifact.s3_key || 'Stored'}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2.5 text-slate-700 col-span-2">
              <div className="p-2 rounded-lg bg-amber-50 text-amber-600 border border-amber-100 shrink-0">
                <Calendar className="w-4 h-4" />
              </div>
              <div>
                <p className="text-slate-500 font-medium">Last Updated</p>
                <p className="font-semibold text-slate-900">{new Date(artifact.updated_at).toLocaleString()}</p>
              </div>
            </div>
          </div>

          {/* Knowledge Graph Connections */}
          <div>
            <h3 className="text-xs font-bold text-slate-500 uppercase tracking-wider mb-3 flex items-center gap-2">
              <LinkIcon className="w-4 h-4 text-indigo-600" />
              <span>Knowledge Reference Links ({outgoingLinks.length + incomingLinks.length})</span>
            </h3>

            {outgoingLinks.length === 0 && incomingLinks.length === 0 ? (
              <p className="text-xs text-slate-500 italic bg-slate-50 p-3.5 rounded-xl border border-slate-200/80">
                No graph reference links detected. Add links like <span className="font-mono font-semibold text-indigo-600">[Link](docs/target.md)</span> or <span className="font-mono font-semibold text-indigo-600">[[target.md]]</span> in your markdown.
              </p>
            ) : (
              <div className="space-y-2">
                {outgoingLinks.map((edge, i) => {
                  const targetNode = graph?.nodes.find((n) => n.id === edge.target);
                  return (
                    <div key={i} className="flex items-center justify-between p-3 bg-slate-50 rounded-xl text-xs border border-slate-200/80">
                      <span className="text-slate-500 font-medium">Refers to ➔</span>
                      <span className="font-mono text-indigo-600 font-bold">{targetNode?.file_path || edge.target}</span>
                      <span className="text-[10px] font-bold px-2 py-0.5 rounded bg-slate-200 text-slate-700">{edge.relationship}</span>
                    </div>
                  );
                })}

                {incomingLinks.map((edge, i) => {
                  const sourceNode = graph?.nodes.find((n) => n.id === edge.source);
                  return (
                    <div key={i} className="flex items-center justify-between p-3 bg-slate-50 rounded-xl text-xs border border-slate-200/80">
                      <span className="text-slate-500 font-medium">Referenced by ⬅</span>
                      <span className="font-mono text-indigo-600 font-bold">{sourceNode?.file_path || edge.source}</span>
                      <span className="text-[10px] font-bold px-2 py-0.5 rounded bg-slate-200 text-slate-700">{edge.relationship}</span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Content Preview */}
          <div>
            <h3 className="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">Raw Markdown Content</h3>
            <pre className="p-4 bg-slate-50 rounded-2xl text-xs font-mono text-slate-800 border border-slate-200/80 overflow-x-auto whitespace-pre-wrap max-h-96">
              {artifact.content || 'Content stored in S3.'}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
};
