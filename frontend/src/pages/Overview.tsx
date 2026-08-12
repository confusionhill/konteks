import React, { useEffect, useState } from 'react';
import { fetchSummary, fetchRooms } from '../api/dashboard';
import type { UsageSummary, RoomOverview } from '../api/dashboard';
import { StatCard } from '../components/StatCard';
import { RoomTable } from '../components/RoomTable';
import { StatCardSkeleton, TableSkeleton } from '../components/Skeleton';
import { EmptyState } from '../components/EmptyState';
import { Door, ChatCircleText, HardDrive, Lightning, Warning, CurrencyDollar, ArrowDownLeft, ArrowUpRight } from '@phosphor-icons/react';

export const OverviewPage: React.FC = () => {
  const [summary, setSummary] = useState<UsageSummary | null>(null);
  const [rooms, setRooms] = useState<RoomOverview[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadDashboard() {
      try {
        setLoading(true);
        setError(null);
        const [sumData, roomsData] = await Promise.all([fetchSummary(), fetchRooms()]);
        setSummary(sumData);
        setRooms(roomsData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load dashboard data');
      } finally {
        setLoading(false);
      }
    }
    loadDashboard();
  }, []);

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
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
        </div>
      ) : (
        summary && (
          <div className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <StatCard
                label="Active Rooms"
                value={summary.total_rooms}
                icon={<Door size={22} weight="duotone" />}
                highlighted={true}
                subtext="Persistent contexts"
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

            {/* Token & Cost Analytics Row (DeepSeek V4 Flash Pricing) */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <StatCard
                label="Total Cost (USD)"
                value={`$${(summary.total_cost_overall || 0).toFixed(6)}`}
                icon={<CurrencyDollar size={22} weight="duotone" />}
                subtext={`In: $${(summary.total_cost_input || 0).toFixed(6)} | Out: $${(summary.total_cost_output || 0).toFixed(6)}`}
              />
              <StatCard
                label="Avg Input Tokens"
                value={Math.round(summary.avg_tokens_input || 0)}
                icon={<ArrowDownLeft size={22} weight="duotone" />}
                subtext="DeepSeek $0.0679 / 1M"
              />
              <StatCard
                label="Avg Output Tokens"
                value={Math.round(summary.avg_tokens_output || 0)}
                icon={<ArrowUpRight size={22} weight="duotone" />}
                subtext="DeepSeek $0.168 / 1M"
              />
            </div>
          </div>
        )
      )}

      {/* Active Rooms Table Section */}
      <div className="space-y-4 pt-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-bold text-slate-900 tracking-tight">Active Rooms</h2>
          <span className="text-xs font-mono text-slate-500 font-medium">
            Showing top {rooms.length} recent rooms
          </span>
        </div>

        {loading ? (
          <TableSkeleton />
        ) : rooms.length > 0 ? (
          <RoomTable rooms={rooms} />
        ) : (
          <EmptyState />
        )}
      </div>
    </div>
  );
};
