import React, { useEffect, useState } from 'react';
import { fetchSummary, fetchRooms, fetchEvaluationSummary } from '../api/dashboard';
import type { UsageSummary, RoomOverview, EvaluationSummary } from '../api/dashboard';
import { StatCard } from '../components/StatCard';
import { RoomTable } from '../components/RoomTable';
import { StatCardSkeleton, TableSkeleton } from '../components/Skeleton';
import { EmptyState } from '../components/EmptyState';
import { Door, ChatCircleText, HardDrive, Lightning, Warning, CurrencyDollar, ArrowDownLeft, ArrowUpRight, MagnifyingGlass, SortAscending, SortDescending, CaretLeft, CaretRight } from '@phosphor-icons/react';

export const OverviewPage: React.FC = () => {
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
      try {
        setLoading(true);
        setError(null);
        const [sumData, roomsResp, evalSumData] = await Promise.all([
          fetchSummary(),
          fetchRooms({ search, status, sortBy, sortOrder, page, limit, minMessages }),
          fetchEvaluationSummary().catch(e => { console.error("Eval summary not available:", e); return null; })
        ]);
        setSummary(sumData);
        setRooms(roomsResp.rooms);
        setTotalRooms(roomsResp.total);
        if (evalSumData) setEvalSummary(evalSumData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load dashboard data');
      } finally {
        setLoading(false);
      }
    }
    loadDashboard();
  }, [search, status, sortBy, sortOrder, page, minMessages]);

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
    <div className="space-y-8">
      {/* Page Title Header */}
      <div>
        <h1 className="text-3xl sm:text-4xl font-extrabold text-slate-900 tracking-tight font-sans">
          Project <span className="bg-gradient-to-r from-primary-600 to-secondary-600 bg-clip-text text-transparent">Overview</span>
        </h1>
        <p className="text-slate-500 text-sm mt-1 font-medium">
          Monitor your active conversation rooms, memory usage, and context API calls.
        </p>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="bg-rose-50 border border-rose-200 text-rose-800 rounded-xl p-4 flex items-center gap-3 text-sm">
          <Warning size={20} className="text-rose-600 shrink-0" />
          <div>
            <span className="font-bold">Error loading dashboard:</span> {error}. Ensure backend is running.
          </div>
        </div>
      )}

      {/* Stat Cards Grid (Asymmetric layout per SKILL.MD guidelines: 2fr 1fr 1fr 1fr) */}
      {loading && !summary ? (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
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
                icon={<Door size={22} weight="duotone" />}
                highlighted={true}
                subtext="Persistent contexts"
              />
              <StatCard
                label="Archived Rooms"
                value={summary.archived_rooms || 0}
                icon={<Door size={22} weight="duotone" className="text-slate-400" />}
                subtext="Cold storage"
              />
              <StatCard
                label="Total Messages"
                value={summary.total_messages}
                icon={<ChatCircleText size={22} weight="duotone" />}
                subtext="Logged in DB"
              />
              <StatCard
                label="Vector Memories"
                value={summary.total_memories}
                icon={<HardDrive size={22} weight="duotone" />}
                subtext="pgvector embeddings"
              />
              <StatCard
                label="API Calls"
                value={summary.total_api_calls}
                icon={<Lightning size={22} weight="duotone" />}
                subtext="Tracked usage"
              />
            </div>

            {/* Token & Cost Analytics Row */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <StatCard
                label="Total Cost (USD)"
                value={`$${(summary.total_cost_overall || 0).toFixed(6)}`}
                icon={<CurrencyDollar size={22} weight="duotone" />}
                subtext={`Ops: $${((summary.total_api_calls || 0) * 0.00001).toFixed(6)} | Proc: $${(summary.total_cost_input || 0).toFixed(6)}`}
              />
              <StatCard
                label="Avg Tokens Processed"
                value={Math.round(summary.avg_tokens_input || 0)}
                icon={<ArrowDownLeft size={22} weight="duotone" />}
                subtext="Scanned & ranked @ $0.07 / 1M"
              />
              <StatCard
                label="Avg Tokens Delivered"
                value={Math.round(summary.avg_tokens_output || 0)}
                icon={<ArrowUpRight size={22} weight="duotone" />}
                subtext="Delivered to Agent (Free)"
              />
            </div>
            
            {/* Context Observability Row */}
            {evalSummary && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <StatCard
                  label="Avg Relevance"
                  value={evalSummary.avg_relevance > 0 ? `${(evalSummary.avg_relevance * 100).toFixed(1)}%` : "N/A"}
                  icon={<Lightning size={22} weight="duotone" className="text-amber-500" />}
                  subtext={`${evalSummary.total_evaluations} evaluations`}
                />
                <StatCard
                  label="Avg Completeness"
                  value={evalSummary.avg_completeness > 0 ? `${(evalSummary.avg_completeness * 100).toFixed(1)}%` : "N/A"}
                  icon={<ChatCircleText size={22} weight="duotone" className="text-emerald-500" />}
                  subtext="Context fullness"
                />
                <StatCard
                  label="Token Efficiency"
                  value={evalSummary.avg_token_efficiency > 0 ? `${(evalSummary.avg_token_efficiency * 100).toFixed(1)}%` : "N/A"}
                  icon={<HardDrive size={22} weight="duotone" className="text-blue-500" />}
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
          <h2 className="text-xl font-bold text-slate-900 tracking-tight">Conversation Rooms</h2>
          <span className="text-xs font-mono text-slate-500 font-medium">
            Found {totalRooms} rooms matching filters
          </span>
        </div>

        {/* Search, Sort, and Filter Controls */}
        <div className="bg-surface rounded-xl border border-slate-100 shadow-soft p-4 space-y-4 md:space-y-0 md:flex md:items-center md:gap-4">
          {/* Search Input */}
          <div className="flex-1 relative">
            <input
              type="text"
              placeholder="Search by Room ID, User ID, or Topic..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && setSearch(searchQuery)}
              className="w-full pl-10 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 text-slate-800 bg-white"
            />
            <MagnifyingGlass size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {/* Min Messages */}
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Min Msgs:</span>
              <input
                type="number"
                min="0"
                value={minMessages === 0 ? '' : minMessages}
                placeholder="0"
                onChange={(e) => {
                  setMinMessages(Number(e.target.value) || 0);
                  setPage(1);
                }}
                className="w-16 px-2.5 py-1.5 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 text-slate-800 text-center font-mono bg-white"
              />
            </div>

            {/* Status Filter */}
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Status:</span>
              <select
                value={status}
                onChange={(e) => {
                  setStatus(e.target.value);
                  setPage(1);
                }}
                className="bg-white border border-slate-200 rounded-lg text-sm px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-primary-500 text-slate-800"
              >
                <option value="">All</option>
                <option value="active">Active</option>
                <option value="archived">Archived</option>
              </select>
            </div>

            {/* Sort By */}
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Sort:</span>
              <select
                value={sortBy}
                onChange={(e) => {
                  setSortBy(e.target.value);
                  setPage(1);
                }}
                className="bg-white border border-slate-200 rounded-lg text-sm px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-primary-500 text-slate-800"
              >
                <option value="last_active_at">Last Active</option>
                <option value="id">Room ID</option>
                <option value="external_user_id">User ID</option>
                <option value="active_topic">Topic</option>
                <option value="message_count">Message Count</option>
                <option value="total_cost">Total Cost</option>
              </select>
            </div>

            {/* Sort Order Toggle */}
            <button
              onClick={() => {
                setSortOrder(sortOrder === 'DESC' ? 'ASC' : 'DESC');
                setPage(1);
              }}
              className="p-2 border border-slate-200 rounded-lg text-slate-600 hover:bg-slate-50 transition-colors bg-white"
              title={sortOrder === 'DESC' ? 'Descending' : 'Ascending'}
            >
              {sortOrder === 'DESC' ? <SortDescending size={18} /> : <SortAscending size={18} />}
            </button>

            {/* Apply & Reset Buttons */}
            <button
              onClick={() => {
                setSearch(searchQuery);
                setPage(1);
              }}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg text-sm font-semibold transition-colors shadow-soft"
            >
              Apply
            </button>
            <button
              onClick={handleReset}
              className="px-3 py-2 border border-slate-200 text-slate-600 hover:bg-slate-50 rounded-lg text-sm font-semibold transition-colors bg-white"
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
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between border-t border-slate-100 pt-4 gap-3 text-sm text-slate-600">
                <div>
                  Showing <span className="font-semibold">{Math.min((page - 1) * limit + 1, totalRooms)}</span> to{' '}
                  <span className="font-semibold">{Math.min(page * limit, totalRooms)}</span> of{' '}
                  <span className="font-semibold">{totalRooms}</span> rooms
                </div>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => setPage((p) => Math.max(p - 1, 1))}
                    disabled={page === 1}
                    className="p-2 border border-slate-200 rounded-lg hover:bg-slate-50 transition-colors disabled:opacity-50 disabled:hover:bg-transparent bg-white"
                  >
                    <CaretLeft size={16} />
                  </button>
                  <span className="font-semibold text-slate-800">
                    Page {page} of {Math.ceil(totalRooms / limit)}
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.min(p + 1, Math.ceil(totalRooms / limit)))}
                    disabled={page === Math.ceil(totalRooms / limit)}
                    className="p-2 border border-slate-200 rounded-lg hover:bg-slate-50 transition-colors disabled:opacity-50 disabled:hover:bg-transparent bg-white"
                  >
                    <CaretRight size={16} />
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
