import React, { useState, useMemo } from 'react';
import type { KnowledgeGraph, Artifact } from '../api/artifacts';
import { Network, Search, Layers, Link as LinkIcon, ZoomIn, ZoomOut, RefreshCw } from 'lucide-react';

interface ArtifactsGraphViewProps {
  graph: KnowledgeGraph | null;
  artifacts: Artifact[];
  onSelectArtifact: (artifact: Artifact) => void;
}

export const ArtifactsGraphView: React.FC<ArtifactsGraphViewProps> = ({
  graph,
  artifacts,
  onSelectArtifact,
}) => {
  const [search, setSearch] = useState('');
  const [selectedGroup, setSelectedGroup] = useState<string>('all');
  const [zoom, setZoom] = useState(1);

  const nodesList = useMemo(() => graph?.nodes || [], [graph]);
  const edgesList = useMemo(() => graph?.edges || [], [graph]);

  // Group options
  const groups = useMemo(() => {
    const set = new Set<string>();
    nodesList.forEach((n) => {
      if (n.group_name) set.add(n.group_name);
    });
    return Array.from(set);
  }, [nodesList]);

  // Compute 2D Positions for SVG Nodes using force layout positioning
  const nodePositions = useMemo(() => {
    if (nodesList.length === 0) return new Map();

    const positions = new Map<string, { x: number; y: number }>();
    const total = nodesList.length;
    const width = 800;
    const height = 500;
    const centerX = width / 2;
    const centerY = height / 2;
    const radius = Math.min(width, height) * 0.35;

    nodesList.forEach((node, idx) => {
      const angle = (idx / total) * 2 * Math.PI;
      const x = centerX + radius * Math.cos(angle);
      const y = centerY + radius * Math.sin(angle);
      positions.set(node.id, { x, y });
    });

    return positions;
  }, [nodesList]);

  if (!graph || nodesList.length === 0) {
    return (
      <div className="bg-white rounded-2xl p-12 border border-slate-200/80 shadow-soft text-center">
        <div className="w-12 h-12 rounded-2xl bg-indigo-50 text-indigo-600 border border-indigo-100 flex items-center justify-center mx-auto mb-3">
          <Network className="w-6 h-6" />
        </div>
        <h3 className="text-base font-bold text-slate-900">No Knowledge Graph Nodes Yet</h3>
        <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1">
          Upload Markdown files with links like <span className="font-mono text-indigo-600 font-semibold">[Link](docs/auth.md)</span> to automatically construct a memory reference graph.
        </p>
      </div>
    );
  }

  const filteredNodes = nodesList.filter((node) => {
    const matchesSearch =
      (node.file_path || '').toLowerCase().includes(search.toLowerCase()) ||
      (node.title || '').toLowerCase().includes(search.toLowerCase());
    const matchesGroup = selectedGroup === 'all' || node.group_name === selectedGroup;
    return matchesSearch && matchesGroup;
  });

  const filteredNodeIDs = new Set(filteredNodes.map((n) => n.id));

  return (
    <div className="bg-white border border-slate-200/80 rounded-2xl overflow-hidden shadow-soft flex flex-col">
      {/* Controls Bar */}
      <div className="p-4 border-b border-slate-100 flex flex-wrap items-center justify-between gap-3 bg-slate-50/50">
        <div className="flex items-center gap-3 flex-1 min-w-[240px]">
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search graph nodes..."
              className="w-full pl-9 pr-3 py-1.5 bg-white border border-slate-200 rounded-xl text-xs text-slate-900 focus:outline-none focus:border-indigo-500"
            />
          </div>

          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-slate-400 shrink-0" />
            <select
              value={selectedGroup}
              onChange={(e) => setSelectedGroup(e.target.value)}
              className="bg-white border border-slate-200 text-slate-700 text-xs font-medium rounded-xl px-2.5 py-1.5 focus:outline-none focus:border-indigo-500"
            >
              <option value="all">All Groups ({nodesList.length})</option>
              {groups.map((g) => (
                <option key={g} value={g}>
                  Group: {g}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={() => setZoom((z) => Math.min(z + 0.2, 2))}
            className="p-1.5 bg-white hover:bg-slate-100 text-slate-700 border border-slate-200 rounded-lg transition-colors"
            title="Zoom In"
          >
            <ZoomIn className="w-4 h-4" />
          </button>
          <button
            onClick={() => setZoom((z) => Math.max(z - 0.2, 0.6))}
            className="p-1.5 bg-white hover:bg-slate-100 text-slate-700 border border-slate-200 rounded-lg transition-colors"
            title="Zoom Out"
          >
            <ZoomOut className="w-4 h-4" />
          </button>
          <button
            onClick={() => setZoom(1)}
            className="p-1.5 bg-white hover:bg-slate-100 text-slate-700 border border-slate-200 rounded-lg transition-colors"
            title="Reset Zoom"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* SVG Interactive Canvas */}
      <div className="relative w-full h-[520px] bg-slate-50/40 overflow-hidden flex items-center justify-center">
        <svg
          className="w-full h-full cursor-grab active:cursor-grabbing transition-transform duration-200 ease-out"
          viewBox="0 0 800 500"
          style={{ transform: `scale(${zoom})` }}
        >
          <defs>
            <marker
              id="arrowhead"
              markerWidth="10"
              markerHeight="7"
              refX="16"
              refY="3.5"
              orient="auto"
            >
              <polygon points="0 0, 10 3.5, 0 7" fill="#4f46e5" opacity="0.8" />
            </marker>
          </defs>

          {/* Render Graph Edges */}
          {edgesList.map((edge, i) => {
            const srcPos = nodePositions.get(edge.source);
            const tgtPos = nodePositions.get(edge.target);
            if (!srcPos || !tgtPos) return null;

            const isHighlighted = filteredNodeIDs.has(edge.source) && filteredNodeIDs.has(edge.target);

            return (
              <g key={i}>
                <line
                  x1={srcPos.x}
                  y1={srcPos.y}
                  x2={tgtPos.x}
                  y2={tgtPos.y}
                  stroke={isHighlighted ? '#4f46e5' : '#cbd5e1'}
                  strokeWidth={isHighlighted ? 2.5 : 1.5}
                  strokeDasharray={edge.relationship === 'refers_to' ? 'none' : '4,4'}
                  markerEnd="url(#arrowhead)"
                  opacity={isHighlighted ? 0.9 : 0.5}
                />
              </g>
            );
          })}

          {/* Render Graph Nodes */}
          {nodesList.map((node) => {
            const pos = nodePositions.get(node.id);
            if (!pos) return null;

            const isMatch = filteredNodeIDs.has(node.id);
            const art = artifacts.find((a) => a.id === node.id);

            return (
              <g
                key={node.id}
                transform={`translate(${pos.x}, ${pos.y})`}
                onClick={() => art && onSelectArtifact(art)}
                className={`cursor-pointer transition-all duration-200 ${
                  isMatch ? 'opacity-100' : 'opacity-40'
                }`}
              >
                {/* Node Outer Ring */}
                <circle
                  r={22}
                  className="fill-white stroke-indigo-500 hover:stroke-indigo-600 hover:scale-110 transition-all shadow-md"
                  strokeWidth={2.5}
                />
                <circle r={14} className="fill-indigo-50" />

                {/* Node Icon Accent */}
                <circle r={6} className="fill-indigo-600" />

                {/* Node File Path Text */}
                <text
                  y={36}
                  textAnchor="middle"
                  className="fill-slate-900 text-[11px] font-mono font-bold tracking-tight"
                >
                  {node.file_path}
                </text>
              </g>
            );
          })}
        </svg>

        {/* Floating Legend */}
        <div className="absolute bottom-4 left-4 bg-white/90 border border-slate-200 px-3.5 py-2 rounded-xl backdrop-blur-md text-[11px] font-medium text-slate-600 flex items-center gap-4 shadow-soft">
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-indigo-600 inline-block" />
            <span>Artifact Node</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-4 h-0.5 bg-indigo-600 inline-block" />
            <span>Refers To Link</span>
          </div>
          <div className="flex items-center gap-1">
            <LinkIcon className="w-3.5 h-3.5 text-indigo-600" />
            <span>Click node to view</span>
          </div>
        </div>
      </div>
    </div>
  );
};
