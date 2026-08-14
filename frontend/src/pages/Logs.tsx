import React, { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { StatCard } from '../components/StatCard';
import {
  fetchProjectLogs,
  fetchProjectEvaluations,
  fetchEvaluationSummary,
  type ProjectLogEntry,
  type ContextEvaluation,
  type EvaluationSummary,
} from '../api/dashboard';
import {
  Search,
  ChevronLeft,
  ChevronRight,
  User,
  Bot,
  Coins,
  Activity,
  CheckCircle2,
  Cpu,
  ArrowUpDown,
  ExternalLink,
  MessageSquare,
  FileText,
  RotateCw,
} from 'lucide-react';

export const LogsPage: React.FC = () => {
  const navigate = useNavigate();

  // Tab mode: 'messages' | 'evaluations'
  const [activeTab, setActiveTab] = useState<'messages' | 'evaluations'>('messages');

  // Top Evaluation Metrics Summary
  const [evalSummary, setEvalSummary] = useState<EvaluationSummary | null>(null);

  // Message Logs State
  const [logs, setLogs] = useState<ProjectLogEntry[]>([]);
  const [logsTotal, setLogsTotal] = useState(0);
  const [logsLoading, setLogsLoading] = useState(false);

  // Message Logs Filters
  const [search, setSearch] = useState('');
  const [roomIdFilter, setRoomIdFilter] = useState('');
  const [userIdFilter, setUserIdFilter] = useState('');
  const [roleFilter, setRoleFilter] = useState('');
  const [sortBy, setSortBy] = useState('created_at');
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');
  const [page, setPage] = useState(1);
  const limit = 15;

  // Context Evaluations Logs State
  const [evaluations, setEvaluations] = useState<ContextEvaluation[]>([]);
  const [evalsTotal, setEvalsTotal] = useState(0);
  const [evalsLoading, setEvalsLoading] = useState(false);
  const [evalPage, setEvalPage] = useState(1);
  const evalLimit = 15;

  // Load Evaluation Summary Metrics
  const loadEvalSummary = () => {
    fetchEvaluationSummary()
      .then((data) => setEvalSummary(data))
      .catch(() => {});
  };

  // Load Message Logs Stream
  const loadLogs = () => {
    setLogsLoading(true);
    fetchProjectLogs({
      roomId: roomIdFilter.trim(),
      externalUserId: userIdFilter.trim(),
      search: search.trim(),
      role: roleFilter,
      sortBy,
      sortOrder,
      page,
      limit,
    })
      .then((res) => {
        setLogs(res.logs || []);
        setLogsTotal(res.total || 0);
      })
      .catch((err) => console.error('Failed to load project logs:', err))
      .finally(() => setLogsLoading(false));
  };

  // Load Context Evaluations Log Stream
  const loadEvaluations = () => {
    setEvalsLoading(true);
    fetchProjectEvaluations({
      page: evalPage,
      limit: evalLimit,
    })
      .then((res) => {
        setEvaluations(res.evaluations || []);
        setEvalsTotal(res.total || 0);
      })
      .catch((err) => console.error('Failed to load evaluations:', err))
      .finally(() => setEvalsLoading(false));
  };

  useEffect(() => {
    loadEvalSummary();
  }, []);

  useEffect(() => {
    if (activeTab === 'messages') {
      loadLogs();
    } else {
      loadEvaluations();
    }
  }, [activeTab, page, evalPage, sortBy, sortOrder, roleFilter]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    loadLogs();
  };

  const handleResetFilters = () => {
    setSearch('');
    setRoomIdFilter('');
    setUserIdFilter('');
    setRoleFilter('');
    setSortBy('created_at');
    setSortOrder('desc');
    setPage(1);
  };

  const formatScore = (val?: number) => {
    if (val === undefined || val === null) return '0.0%';
    return (val * 100).toFixed(1) + '%';
  };

  const totalLogsPages = Math.ceil(logsTotal / limit) || 1;
  const totalEvalsPages = Math.ceil(evalsTotal / evalLimit) || 1;

  return (
    <div className="space-y-6 font-sans">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <span className="text-xs font-bold uppercase tracking-wider text-indigo-600 bg-indigo-50 px-3 py-1 rounded-full border border-indigo-100 inline-block mb-2">
            Project Intelligence
          </span>
          <h1 className="text-3xl font-extrabold text-slate-900 tracking-tight">Logs & Context Intelligence</h1>
          <p className="text-slate-500 text-sm mt-1">
            Real-time project log stream and context quality evaluations across all rooms.
          </p>
        </div>

        <button
          onClick={() => {
            loadEvalSummary();
            if (activeTab === 'messages') loadLogs();
            else loadEvaluations();
          }}
          className="inline-flex items-center space-x-2 px-4 py-2 bg-white border border-slate-200 rounded-lg text-xs font-semibold text-slate-700 hover:bg-slate-50 transition-all shadow-soft shrink-0"
        >
          <RotateCw className="h-3.5 w-3.5 text-indigo-600" />
          <span>Refresh Data</span>
        </button>
      </div>

      {/* TOP EVALUATION QUALITY METRIC CARDS */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard
          label="Avg Relevance"
          value={formatScore(evalSummary?.avg_relevance)}
          icon={<Activity className="h-5 w-5 text-emerald-600" />}
          subtext={`${evalSummary?.total_evaluations || 0} evaluations`}
          progress={evalSummary?.avg_relevance}
          progressColor="bg-emerald-500"
        />
        <StatCard
          label="Avg Completeness"
          value={formatScore(evalSummary?.avg_completeness)}
          icon={<CheckCircle2 className="h-5 w-5 text-indigo-600" />}
          subtext="Context fullness"
          progress={evalSummary?.avg_completeness}
          progressColor="bg-indigo-600"
        />
        <StatCard
          label="Token Efficiency"
          value={formatScore(evalSummary?.avg_token_efficiency)}
          icon={<Cpu className="h-5 w-5 text-violet-600" />}
          subtext="Signal vs Noise"
          progress={evalSummary?.avg_token_efficiency}
          progressColor="bg-violet-600"
        />
      </div>

      {/* VIEW MODE TABS & LOG STREAM CARD */}
      <div className="bg-white border border-slate-200/80 rounded-xl shadow-soft overflow-hidden">
        {/* Header & Filter Controls Container */}
        <div className="p-6 space-y-5">
          <div className="flex items-center justify-between border-b border-slate-100 pb-4">
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setActiveTab('messages')}
                className={`flex items-center space-x-2 px-4 py-2 rounded-lg text-xs font-bold transition-all ${
                  activeTab === 'messages'
                    ? 'bg-slate-900 text-white shadow-sm'
                    : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
                }`}
              >
                <MessageSquare className="h-4 w-4" />
                <span>Message Logs Stream</span>
                <span className="ml-1 bg-white/20 px-2 py-0.5 rounded-full text-[10px]">
                  {logsTotal}
                </span>
              </button>

              <button
                onClick={() => setActiveTab('evaluations')}
                className={`flex items-center space-x-2 px-4 py-2 rounded-lg text-xs font-bold transition-all ${
                  activeTab === 'evaluations'
                    ? 'bg-slate-900 text-white shadow-sm'
                    : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
                }`}
              >
                <FileText className="h-4 w-4" />
                <span>Context Evaluations Log</span>
                <span className="ml-1 bg-white/20 px-2 py-0.5 rounded-full text-[10px]">
                  {evalsTotal}
                </span>
              </button>
            </div>
          </div>

          {/* Filter & Search Toolbar (Only for messages tab) */}
          {activeTab === 'messages' && (
            <form onSubmit={handleSearchSubmit} className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
              {/* Content / General Search */}
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-slate-400" />
                <input
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search message text..."
                  className="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              {/* Room ID Filter */}
              <div>
                <input
                  type="text"
                  value={roomIdFilter}
                  onChange={(e) => setRoomIdFilter(e.target.value)}
                  placeholder="Filter by Room ID..."
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono"
                />
              </div>

              {/* User ID Filter */}
              <div>
                <input
                  type="text"
                  value={userIdFilter}
                  onChange={(e) => setUserIdFilter(e.target.value)}
                  placeholder="Filter by User ID..."
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              {/* Role Filter */}
              <div>
                <select
                  value={roleFilter}
                  onChange={(e) => {
                    setRoleFilter(e.target.value);
                    setPage(1);
                  }}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="">All Roles</option>
                  <option value="user">User</option>
                  <option value="assistant">Assistant</option>
                  <option value="system">System</option>
                </select>
              </div>

              {/* Sort By & Order Toggle */}
              <div className="flex items-center space-x-1.5">
                <select
                  value={sortBy}
                  onChange={(e) => {
                    setSortBy(e.target.value);
                    setPage(1);
                  }}
                  className="flex-1 px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="created_at">Sort by Timestamp</option>
                  <option value="tokens">Sort by Tokens</option>
                  <option value="cost_total">Sort by Cost</option>
                </select>

                <button
                  type="button"
                  onClick={() => setSortOrder(sortOrder === 'desc' ? 'asc' : 'desc')}
                  className="p-2 rounded-lg bg-slate-50 border border-slate-200 text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition-all shrink-0"
                  title={`Order: ${sortOrder.toUpperCase()}`}
                >
                  <ArrowUpDown className="h-3.5 w-3.5" />
                </button>

                <button
                  type="button"
                  onClick={handleResetFilters}
                  className="p-2 rounded-lg text-xs font-semibold text-slate-500 hover:text-slate-900 transition-colors"
                  title="Reset Filters"
                >
                  Reset
                </button>
              </div>
            </form>
          )}
        </div>

        {/* TAB 1: MESSAGE LOGS STREAM TABLE */}
        {activeTab === 'messages' && (
          <div>
            {logsLoading ? (
              <div className="flex items-center justify-center py-16 border-t border-slate-200/80">
                <div className="w-5 h-5 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin mr-2" />
                <span className="text-xs text-slate-500">Loading project message logs...</span>
              </div>
            ) : logs.length === 0 ? (
              <div className="text-center py-12 border-t border-slate-200/80 bg-slate-50/50">
                <p className="text-xs text-slate-400 italic">No message logs match your active search filters.</p>
              </div>
            ) : (
              <div>
                <div className="border-t border-slate-200/80 overflow-x-auto">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="bg-slate-50/80 border-b border-slate-200 text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                        <th className="py-3.5 px-6">Timestamp</th>
                        <th className="py-3.5 px-6">Role</th>
                        <th className="py-3.5 px-6">User ID</th>
                        <th className="py-3.5 px-6">Room ID</th>
                        <th className="py-3.5 px-6">Content</th>
                        <th className="py-3.5 px-6 text-right">Tokens & Cost</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 text-xs">
                      {logs.map((log) => {
                        const isUser = log.role === 'user';
                        return (
                          <tr key={log.id} className="hover:bg-slate-50/80 transition-colors">
                            <td className="py-3.5 px-6 font-mono text-slate-500 whitespace-nowrap">
                              {new Date(log.created_at).toLocaleString()}
                            </td>

                            <td className="py-3.5 px-6 whitespace-nowrap">
                              <span
                                className={`inline-flex items-center space-x-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase tracking-wider ${
                                  isUser
                                    ? 'bg-slate-900 text-white'
                                    : 'bg-indigo-100 text-indigo-700'
                                }`}
                              >
                                {isUser ? <User className="h-3 w-3" /> : <Bot className="h-3 w-3" />}
                                <span>{log.role}</span>
                              </span>
                            </td>

                            <td className="py-3.5 px-6 font-medium text-slate-900 whitespace-nowrap">
                              {log.external_user_id || '-'}
                            </td>

                            <td className="py-3.5 px-6 whitespace-nowrap">
                              <Link
                                to={`/dashboard/logs/${log.room_id}`}
                                className="inline-flex items-center space-x-1 text-indigo-600 font-mono hover:underline font-semibold"
                                title="Inspect Room Detail Page"
                              >
                                <span>{log.room_id.slice(0, 8)}...</span>
                                <ExternalLink className="h-3 w-3" />
                              </Link>
                            </td>

                            <td className="py-3.5 px-6 text-slate-800 max-w-md truncate">
                              {log.content || <span className="italic text-slate-400">(No text content)</span>}
                            </td>

                            <td className="py-3.5 px-6 text-right font-mono text-slate-600 whitespace-nowrap">
                              {log.tokens_input !== undefined || log.tokens_output !== undefined ? (
                                <span className="inline-flex items-center space-x-1 bg-slate-100 px-2 py-0.5 rounded border border-slate-200 text-[11px]">
                                  <Coins className="h-3 w-3 text-amber-500" />
                                  <span>
                                    in:{log.tokens_input || 0} out:{log.tokens_output || 0}
                                  </span>
                                </span>
                              ) : (
                                '-'
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>

                {/* Message Logs Pagination Bar */}
                <div className="px-6 py-3.5 bg-slate-50/80 border-t border-slate-200 flex items-center justify-between text-xs text-slate-500">
                  <span>
                    Showing {logs.length} of {logsTotal} message logs
                  </span>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => setPage((p) => Math.max(p - 1, 1))}
                      disabled={page === 1}
                      className="p-1.5 rounded-lg border border-slate-200 hover:bg-slate-100 disabled:opacity-40 transition-all"
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </button>

                    <span className="font-semibold text-slate-800">
                      Page {page} of {totalLogsPages}
                    </span>

                    <button
                      onClick={() => setPage((p) => Math.min(p + 1, totalLogsPages))}
                      disabled={page >= totalLogsPages}
                      className="p-1.5 rounded-lg border border-slate-200 hover:bg-slate-100 disabled:opacity-40 transition-all"
                    >
                      <ChevronRight className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* TAB 2: CONTEXT EVALUATIONS LOG STREAM */}
        {activeTab === 'evaluations' && (
          <div className="px-6 pb-6 pt-0 space-y-4">
            {evalsLoading ? (
              <div className="flex items-center justify-center py-16">
                <div className="w-5 h-5 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin mr-2" />
                <span className="text-xs text-slate-500">Loading context evaluations...</span>
              </div>
            ) : evaluations.length === 0 ? (
              <div className="text-center py-12 border border-dashed border-slate-200 rounded-xl">
                <p className="text-xs text-slate-400 italic">No context evaluations recorded yet.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {evaluations.map((ev) => (
                  <div
                    key={ev.id}
                    onClick={() => navigate(`/dashboard/evaluations/${ev.id}`)}
                    className="group bg-slate-50/60 border border-slate-200/80 rounded-xl p-4 hover:bg-white hover:shadow-soft hover:-translate-y-0.5 transition-all cursor-pointer flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3"
                  >
                    {/* Left: Room Badge & Timestamp */}
                    <div className="flex items-center space-x-3">
                      <div className="h-9 w-9 rounded-xl bg-indigo-50 border border-indigo-100 text-indigo-700 flex items-center justify-center font-mono font-bold text-xs shrink-0 group-hover:bg-indigo-600 group-hover:text-white transition-colors">
                        <Activity className="h-4 w-4" />
                      </div>

                      <div>
                        <div className="flex items-center space-x-2">
                          <span className="text-xs font-mono font-bold text-slate-900 group-hover:text-indigo-600 transition-colors">
                            Room: {ev.room_id.slice(0, 8)}...
                          </span>
                          <span className="text-[10px] font-mono text-slate-400">
                            ID: {ev.id.slice(0, 8)}
                          </span>
                        </div>
                        <p className="text-[11px] text-slate-400 font-mono mt-0.5">
                          {new Date(ev.created_at).toLocaleString()}
                        </p>
                      </div>
                    </div>

                    {/* Middle: Evaluation Scores Badges */}
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="bg-emerald-50 text-emerald-700 font-semibold px-2.5 py-1 rounded-full text-[11px] border border-emerald-100">
                        RELEVANCE: {formatScore(ev.relevance_score)}
                      </span>
                      <span className="bg-indigo-50 text-indigo-700 font-semibold px-2.5 py-1 rounded-full text-[11px] border border-indigo-100">
                        COMPLETENESS: {formatScore(ev.completeness_score)}
                      </span>
                      <span className="bg-violet-50 text-violet-700 font-semibold px-2.5 py-1 rounded-full text-[11px] border border-violet-100">
                        EFFICIENCY: {formatScore(ev.token_efficiency_score)}
                      </span>
                    </div>

                    {/* Right: Open Dedicated Detail Page CTA */}
                    <div className="flex items-center space-x-1 text-xs font-semibold text-indigo-600 group-hover:translate-x-1 transition-transform shrink-0">
                      <span>View Details</span>
                      <ExternalLink className="h-3.5 w-3.5" />
                    </div>
                  </div>
                ))}

                {/* Evaluations Pagination Bar */}
                <div className="px-4 py-3 bg-slate-50/80 border border-slate-200/80 rounded-xl flex items-center justify-between text-xs text-slate-500">
                  <span>
                    Showing {evaluations.length} of {evalsTotal} evaluations
                  </span>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => setEvalPage((p) => Math.max(p - 1, 1))}
                      disabled={evalPage === 1}
                      className="p-1.5 rounded-lg border border-slate-200 hover:bg-slate-100 disabled:opacity-40 transition-all"
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </button>

                    <span className="font-semibold text-slate-800">
                      Page {evalPage} of {totalEvalsPages}
                    </span>

                    <button
                      onClick={() => setEvalPage((p) => Math.min(p + 1, totalEvalsPages))}
                      disabled={evalPage >= totalEvalsPages}
                      className="p-1.5 rounded-lg border border-slate-200 hover:bg-slate-100 disabled:opacity-40 transition-all"
                    >
                      <ChevronRight className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
