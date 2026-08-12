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
      className={`bg-surface rounded-xl p-6 border transition-all duration-200 hover:-translate-y-1 active:scale-[0.98] ${
        highlighted
          ? 'border-indigo-200 shadow-lifted ring-2 ring-indigo-500/20'
          : 'border-slate-100 shadow-soft hover:shadow-lifted'
      }`}
    >
      <div className="flex items-center gap-3 mb-3">
        <div className="w-10 h-10 rounded-lg bg-indigo-50 text-primary-600 flex items-center justify-center">
          {icon}
        </div>
        <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
          {label}
        </span>
      </div>

      <div className="text-4xl font-extrabold text-slate-900 tracking-tight my-2 font-sans">
        {value}
      </div>

      <div className="h-1 w-full bg-gradient-to-r from-primary-600 to-secondary-600 rounded-full mt-3 opacity-80" />

      {subtext && (
        <p className="text-xs font-medium text-emerald-600 mt-2 flex items-center gap-1">
          {subtext}
        </p>
      )}
    </div>
  );
};
