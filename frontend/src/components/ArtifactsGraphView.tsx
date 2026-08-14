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

  // Group options
  const groups = useMemo(() => {
    const set = new Set<string>();
    graph?.nodes.forEach((n) => {
      if (n.group_name) set.add(n.group_name);
    });
    return Array.from(set);
  }, [graph]);

  // Compute 2D Positions for SVG Nodes using force layout positioning
  const nodePositions = useMemo(() => {
    if (!graph || graph.nodes.length === 0) return new Map();

    const positions = new Map<string, { x: number; y: number }>();
    const total = graph.nodes.length;
    const width = 800;
    const height = 500;
    const centerX = width / 2;
    const centerY = height / 2;
    const radius = Math.min(width, height) * 0.35;

    graph.nodes.forEach((node, idx) => {
      const angle = (idx / total) * 2 * Math.PI;
      const x = centerX + radius * Math.cos(angle);
      const y = centerY + radius * Math.sin(angle);
      positions.set(node.id, { x, y });
    });

    return positions;
  }, [graph]);

  if (!graph || graph.nodes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-12 bg-slate-950/40 rounded-xl border border-slate-800 text-center">
        <Network className="w-12 h-12 text-slate-600 mb-3" />
        <h3 className="text-base font-semibold text-slate-300">No Knowledge Graph Nodes Yet</h3>
        <p className="text-xs text-slate-500 max-w-sm mt-1">
          Upload Markdown files with links like <span className="font-mono text-indigo-400">[Link](docs/auth.md)</span> to automatically construct a memory reference graph.
        </p>
      </div>
    );
  }

  const filteredNodes = graph.nodes.filter((node) => {
    const matchesSearch =
      node.file_path.toLowerCase().includes(search.toLowerCase()) ||
      node.title.toLowerCase().includes(search.toLowerCase());
    const matchesGroup = selectedGroup === 'all' || node.group_name === selectedGroup;
    return matchesSearch && matchesGroup;
  });

  const filteredNodeIDs = new Set(filteredNodes.map((n) => n.id));

  return (
    <div className="bg-slate-950 border border-slate-800 rounded-xl overflow-hidden shadow-xl flex flex-col">
      {/* Controls Bar */}
      <div className="p-4 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3 bg-slate-900/60">
        <div className="flex items-center gap-3 flex-1 min-w-[240px]">
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search graph nodes..."
              className="w-full pl-9 pr-3 py-1.5 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-indigo-500"
            />
          </div>

          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-slate-400 shrink-0" />
            <select
              value={selectedGroup}
              onChange={(e) => setSelectedGroup(e.target.value)}
              className="bg-slate-950 border border-slate-800 text-slate-300 text-xs rounded-lg px-2.5 py-1.5 focus:outline-none focus:border-indigo-500"
            >
              <option value="all">All Groups ({graph.nodes.length})</option>
              {groups.map((g) => (
                <option key={g} value={g}>
                  Group: {g}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setZoom((z) => Math.min(z + 0.2, 2))}
            className="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-md transition-colors"
            title="Zoom In"
          >
            <ZoomIn className="w-4 h-4" />
          </button>
          <button
            onClick={() => setZoom((z) => Math.max(z - 0.2, 0.6))}
            className="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-md transition-colors"
            title="Zoom Out"
          >
            <ZoomOut className="w-4 h-4" />
          </button>
          <button
            onClick={() => setZoom(1)}
            className="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-md transition-colors"
            title="Reset Zoom"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* SVG Interactive Canvas */}
      <div className="relative w-full h-[520px] bg-slate-950/80 overflow-hidden flex items-center justify-center">
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
              <polygon points="0 0, 10 3.5, 0 7" fill="#6366f1" opacity="0.8" />
            </marker>
          </defs>

          {/* Render Graph Edges */}
          {graph.edges.map((edge, i) => {
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
                  stroke={isHighlighted ? '#6366f1' : '#334155'}
                  strokeWidth={isHighlighted ? 2 : 1}
                  strokeDasharray={edge.relationship === 'refers_to' ? 'none' : '4,4'}
                  markerEnd="url(#arrowhead)"
                  opacity={isHighlighted ? 0.85 : 0.3}
                />
              </g>
            );
          })}

          {/* Render Graph Nodes */}
          {graph.nodes.map((node) => {
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
                  isMatch ? 'opacity-100' : 'opacity-30'
                }`}
              >
                {/* Node Outer Ring */}
                <circle
                  r={22}
                  className="fill-slate-900 stroke-indigo-500/50 hover:stroke-indigo-400 hover:scale-110 transition-all"
                  strokeWidth={2}
                />
                <circle r={14} className="fill-indigo-600/20" />

                {/* Node Icon Accent */}
                <circle r={6} className="fill-indigo-400" />

                {/* Node File Path Text */}
                <text
                  y={34}
                  textAnchor="middle"
                  className="fill-slate-200 text-[10px] font-mono font-medium tracking-tight drop-shadow-md"
                >
                  {node.file_path}
                </text>
              </g>
            );
          })}
        </svg>

        {/* Floating Legend */}
        <div className="absolute bottom-4 left-4 bg-slate-900/90 border border-slate-800 px-3 py-2 rounded-lg backdrop-blur-md text-[11px] text-slate-400 flex items-center gap-4">
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-indigo-500 inline-block" />
            <span>Artifact Node</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-4 h-0.5 bg-indigo-500 inline-block" />
            <span>Refers To Link</span>
          </div>
          <div className="flex items-center gap-1">
            <LinkIcon className="w-3 h-3 text-indigo-400" />
            <span>Click node to view memory</span>
          </div>
        </div>
      </div>
    </div>
  );
};
