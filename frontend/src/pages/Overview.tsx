import React, { useEffect, useState } from 'react';
import { fetchSummary, fetchRooms, fetchEvaluationSummary } from '../api/dashboard';
import type { UsageSummary, RoomOverview, EvaluationSummary } from '../api/dashboard';
import { StatCard } from '../components/StatCard';
import { RoomTable } from '../components/RoomTable';
import { StatCardSkeleton, TableSkeleton } from '../components/Skeleton';
import { EmptyState } from '../components/EmptyState';
import { useAuth } from '../context/AuthContext';
import {
  DoorOpen,
  MessageSquare,
  HardDrive,
  Zap,
  AlertTriangle,
  DollarSign,
  ArrowDownLeft,
  ArrowUpRight,
  Search,
  ArrowDownAZ,
  ArrowUpAZ,
  ChevronLeft,
  ChevronRight,
  Activity,
  Layers,
} from 'lucide-react';

export const OverviewPage: React.FC = () => {
  const { activeProject } = useAuth();
  const [summary, setSummary] = useState<UsageSummary | null>(null);
  const [evalSummary, setEvalSummary] = useState<EvaluationSummary | null>(null);
  const [rooms, setRooms] = useState<RoomOverview[]>([]);
  const [totalRooms, setTotalRooms] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // Filter/Sort/Pagination state
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [search, setSearch] = useState<string>('');
  const [status, setStatus] = useState<string>('');
  const [sortBy, setSortBy] = useState<string>('last_active_at');
  const [sortOrder, setSortOrder] = useState<string>('DESC');
  const [minMessages, setMinMessages] = useState<number>(0);
  const [page, setPage] = useState<number>(1);
  const limit = 10;

  useEffect(() => {
    async function loadDashboard() {
      if (!activeProject) return;
      try {
        setLoading(true);
        setError(null);
        const [sumData, roomsResp, evalSumData] = await Promise.all([
          fetchSummary(),
          fetchRooms({ search, status, sortBy, sortOrder, page, limit, minMessages }),
          fetchEvaluationSummary().catch((e) => {
            console.error('Eval summary not available:', e);
            return null;
          }),
        ]);
        setSummary(sumData);
        setRooms(roomsResp.rooms || []);
        setTotalRooms(roomsResp.total || 0);
        if (evalSumData) setEvalSummary(evalSumData);
      } catch (err: any) {
        setError(err instanceof Error ? err.message : 'Failed to load dashboard data');
      } finally {
        setLoading(false);
      }
    }
    loadDashboard();
  }, [activeProject?.id, search, status, sortBy, sortOrder, page, minMessages]);

  const handleReset = () => {
    setSearchQuery('');
    setSearch('');
    setStatus('');
    setSortBy('last_active_at');
    setSortOrder('DESC');
    setMinMessages(0);
    setPage(1);
  };

  return (
    <div className="space-y-8 font-sans">
      {/* Page Title Header */}
      <div>
        <div className="inline-flex items-center space-x-2 text-xs font-semibold text-indigo-400 bg-indigo-950/60 border border-indigo-500/30 px-3 py-1 rounded-full mb-3">
          <Layers className="h-3.5 w-3.5" />
          <span>Active Workspace: {activeProject?.name || 'Project'}</span>
        </div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">
          Project <span className="bg-gradient-to-r from-indigo-400 to-violet-400 bg-clip-text text-transparent">Overview</span>
        </h1>
        <p className="text-slate-400 text-sm mt-1">
          Monitor your active conversation rooms, vector memory storage, and API call usage metrics.
        </p>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="bg-red-950/80 border border-red-500/50 text-red-200 rounded-2xl p-4 flex items-center space-x-3 text-xs">
          <AlertTriangle className="h-5 w-5 text-red-400 shrink-0" />
          <div>
            <strong className="font-bold">Error loading metrics:</strong> {error}. Make sure the backend server is running.
          </div>
        </div>
      )}

      {/* Stat Cards Grid */}
      {loading && !summary ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
        </div>
      ) : (
        summary && (
          <div className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
              <StatCard
                label="Active Rooms"
                value={summary.active_rooms || 0}
                icon={<DoorOpen className="h-5 w-5" />}
                highlighted={true}
                subtext="Persistent active contexts"
              />
              <StatCard
                label="Archived Rooms"
                value={summary.archived_rooms || 0}
                icon={<DoorOpen className="h-5 w-5 opacity-60" />}
                subtext="S3 cold storage"
              />
              <StatCard
                label="Total Messages"
                value={summary.total_messages || 0}
                icon={<MessageSquare className="h-5 w-5" />}
                subtext="Logged in Postgres"
              />
              <StatCard
                label="Vector Memories"
                value={summary.total_memories || 0}
                icon={<HardDrive className="h-5 w-5" />}
                subtext="pgvector embeddings"
              />
              <StatCard
                label="API Calls"
                value={summary.total_api_calls || 0}
                icon={<Zap className="h-5 w-5" />}
                subtext="Tracked usage"
              />
            </div>

            {/* Token & Cost Analytics Row */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <StatCard
                label="Total Cost (USD)"
                value={`$${(summary.total_cost_overall || 0).toFixed(6)}`}
                icon={<DollarSign className="h-5 w-5 text-emerald-400" />}
                subtext={`Ops: $${((summary.total_api_calls || 0) * 0.00001).toFixed(6)} | Proc: $${(summary.total_cost_input || 0).toFixed(6)}`}
              />
              <StatCard
                label="Avg Tokens Processed"
                value={Math.round(summary.avg_tokens_input || 0)}
                icon={<ArrowDownLeft className="h-5 w-5 text-indigo-400" />}
                subtext="Scanned & ranked @ $0.07 / 1M"
              />
              <StatCard
                label="Avg Tokens Delivered"
                value={Math.round(summary.avg_tokens_output || 0)}
                icon={<ArrowUpRight className="h-5 w-5 text-violet-400" />}
                subtext="Delivered to Agent (Free)"
              />
            </div>

            {/* Context Observability Row */}
            {evalSummary && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <StatCard
                  label="Avg Relevance"
                  value={evalSummary.avg_relevance > 0 ? `${(evalSummary.avg_relevance * 100).toFixed(1)}%` : 'N/A'}
                  icon={<Activity className="h-5 w-5 text-amber-400" />}
                  subtext={`${evalSummary.total_evaluations} evaluations`}
                />
                <StatCard
                  label="Avg Completeness"
                  value={evalSummary.avg_completeness > 0 ? `${(evalSummary.avg_completeness * 100).toFixed(1)}%` : 'N/A'}
                  icon={<MessageSquare className="h-5 w-5 text-emerald-400" />}
                  subtext="Context fullness"
                />
                <StatCard
                  label="Token Efficiency"
                  value={evalSummary.avg_token_efficiency > 0 ? `${(evalSummary.avg_token_efficiency * 100).toFixed(1)}%` : 'N/A'}
                  icon={<HardDrive className="h-5 w-5 text-blue-400" />}
                  subtext="Signal vs Noise"
                />
              </div>
            )}
          </div>
        )
      )}

      {/* Active Rooms Table Section */}
      <div className="space-y-4 pt-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <h2 className="text-xl font-bold text-white tracking-tight">Conversation Rooms</h2>
          <span className="text-xs font-mono text-slate-400 font-medium">
            Found {totalRooms} rooms matching filters
          </span>
        </div>

        {/* Search, Sort, and Filter Controls */}
        <div className="bg-slate-900/70 backdrop-blur-xl rounded-2xl border border-slate-800 p-4 space-y-4 md:space-y-0 md:flex md:items-center md:gap-4 shadow-xl">
          {/* Search Input */}
          <div className="flex-1 relative">
            <input
              type="text"
              placeholder="Search by Room ID, User ID, or Topic..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && setSearch(searchQuery)}
              className="w-full pl-10 pr-4 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {/* Min Messages */}
            <div className="flex items-center space-x-1.5 text-xs text-slate-400">
              <span className="font-semibold uppercase tracking-wider text-[10px]">Min Msgs:</span>
              <input
                type="number"
                min="0"
                value={minMessages === 0 ? '' : minMessages}
                placeholder="0"
                onChange={(e) => {
                  setMinMessages(Number(e.target.value) || 0);
                  setPage(1);
                }}
                className="w-16 px-2.5 py-1.5 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-100 text-center font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>

            {/* Status Filter */}
            <div className="flex items-center space-x-1.5 text-xs text-slate-400">
              <span className="font-semibold uppercase tracking-wider text-[10px]">Status:</span>
              <select
                value={status}
                onChange={(e) => {
                  setStatus(e.target.value);
                  setPage(1);
                }}
                className="bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-100 px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="">All</option>
                <option value="active">Active</option>
                <option value="archived">Archived</option>
              </select>
            </div>

            {/* Sort By */}
            <div className="flex items-center space-x-1.5 text-xs text-slate-400">
              <span className="font-semibold uppercase tracking-wider text-[10px]">Sort:</span>
              <select
                value={sortBy}
                onChange={(e) => {
                  setSortBy(e.target.value);
                  setPage(1);
                }}
                className="bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-100 px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="last_active_at">Last Active</option>
                <option value="id">Room ID</option>
                <option value="external_user_id">User ID</option>
                <option value="active_topic">Topic</option>
                <option value="message_count">Message Count</option>
              </select>
            </div>

            {/* Sort Order Toggle */}
            <button
              onClick={() => {
                setSortOrder(sortOrder === 'DESC' ? 'ASC' : 'DESC');
                setPage(1);
              }}
              className="p-2 border border-slate-800 rounded-xl text-slate-400 hover:text-slate-200 bg-slate-950 transition-colors"
              title={sortOrder === 'DESC' ? 'Descending' : 'Ascending'}
            >
              {sortOrder === 'DESC' ? <ArrowDownAZ className="h-4 w-4" /> : <ArrowUpAZ className="h-4 w-4" />}
            </button>

            {/* Apply & Reset Buttons */}
            <button
              onClick={() => {
                setSearch(searchQuery);
                setPage(1);
              }}
              className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl text-xs font-semibold transition-all shadow-md shadow-indigo-600/20"
            >
              Apply
            </button>
            <button
              onClick={handleReset}
              className="px-3 py-1.5 border border-slate-800 text-slate-400 hover:text-slate-200 rounded-xl text-xs font-semibold transition-colors bg-slate-950"
            >
              Reset
            </button>
          </div>
        </div>

        {loading ? (
          <TableSkeleton />
        ) : rooms.length > 0 ? (
          <div className="space-y-4">
            <RoomTable rooms={rooms} />

            {/* Pagination Controls */}
            {totalRooms > limit && (
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between border-t border-slate-800 pt-4 gap-3 text-xs text-slate-400">
                <div>
                  Showing <span className="font-semibold text-slate-200">{Math.min((page - 1) * limit + 1, totalRooms)}</span> to{' '}
                  <span className="font-semibold text-slate-200">{Math.min(page * limit, totalRooms)}</span> of{' '}
                  <span className="font-semibold text-slate-200">{totalRooms}</span> rooms
                </div>
                <div className="flex items-center space-x-3">
                  <button
                    onClick={() => setPage((p) => Math.max(p - 1, 1))}
                    disabled={page === 1}
                    className="p-1.5 border border-slate-800 rounded-lg hover:bg-slate-800 text-slate-400 transition-colors disabled:opacity-40 bg-slate-950"
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </button>
                  <span className="font-semibold text-slate-200">
                    Page {page} of {Math.ceil(totalRooms / limit)}
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.min(p + 1, Math.ceil(totalRooms / limit)))}
                    disabled={page === Math.ceil(totalRooms / limit)}
                    className="p-1.5 border border-slate-800 rounded-lg hover:bg-slate-800 text-slate-400 transition-colors disabled:opacity-40 bg-slate-950"
                  >
                    <ChevronRight className="h-4 w-4" />
                  </button>
                </div>
              </div>
            )}
          </div>
        ) : (
          <EmptyState />
        )}
      </div>
    </div>
  );
};
