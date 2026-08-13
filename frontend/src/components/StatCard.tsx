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
      className={`bg-white rounded-xl p-5 border transition-all duration-200 hover:-translate-y-1 ${
        highlighted
          ? 'border-indigo-200 shadow-lifted ring-2 ring-indigo-500/20'
          : 'border-slate-200/80 shadow-soft hover:shadow-lifted'
      }`}
    >
      <div className="flex items-center space-x-3 mb-3">
        <div className="w-10 h-10 rounded-lg bg-indigo-50 text-indigo-600 flex items-center justify-center shrink-0">
          {icon}
        </div>
        <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 truncate">
          {label}
        </span>
      </div>

      <div className="text-3xl font-extrabold text-slate-900 tracking-tight my-1 font-sans">
        {value}
      </div>

      <div className="h-1 w-full bg-gradient-to-r from-indigo-600 to-violet-600 rounded-full mt-3 opacity-80" />

      {subtext && (
        <p className="text-xs font-medium text-indigo-600 mt-2 flex items-center gap-1">
          {subtext}
        </p>
      )}
    </div>
  );
};
