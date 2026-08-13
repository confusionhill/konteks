import React from 'react';

interface StatCardProps {
  label: string;
  value: string | number;
  icon: React.ReactNode;
  subtext?: string;
  highlighted?: boolean;
}

export const StatCard: React.FC<StatCardProps> = ({
  label,
  value,
  icon,
  subtext,
  highlighted = false,
}) => {
  return (
    <div
      className={`bg-slate-900/70 border rounded-2xl p-5 backdrop-blur-xl transition-all duration-200 hover:-translate-y-0.5 ${
        highlighted
          ? 'border-indigo-500/50 shadow-lg shadow-indigo-500/10 ring-1 ring-indigo-500/30'
          : 'border-slate-800 hover:border-slate-700 shadow-xl'
      }`}
    >
      <div className="flex items-center space-x-3 mb-3">
        <div className="w-9 h-9 rounded-xl bg-indigo-950/80 border border-indigo-500/30 text-indigo-400 flex items-center justify-center shrink-0">
          {icon}
        </div>
        <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400 truncate">
          {label}
        </span>
      </div>

      <div className="text-3xl font-extrabold text-white tracking-tight my-1">
        {value}
      </div>

      <div className="h-1 w-full bg-gradient-to-r from-indigo-500 to-violet-500 rounded-full mt-3 opacity-60" />

      {subtext && (
        <p className="text-[11px] font-medium text-indigo-300 mt-2 flex items-center gap-1">
          {subtext}
        </p>
      )}
    </div>
  );
};
