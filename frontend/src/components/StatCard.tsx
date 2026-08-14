import React from 'react';

interface StatCardProps {
  label: string;
  value: string | number;
  icon: React.ReactNode;
  subtext?: string;
  badge?: string;
  highlighted?: boolean;
  progress?: number;
  progressColor?: string;
}

export const StatCard: React.FC<StatCardProps> = ({
  label,
  value,
  icon,
  subtext,
  badge,
  highlighted = false,
  progress,
  progressColor,
}) => {
  const progressPercent =
    progress !== undefined
      ? Math.min(Math.max(progress > 1 ? progress : progress * 100, 0), 100)
      : null;

  if (highlighted) {
    return (
      <div className="bg-slate-900 text-white rounded-2xl p-5 border border-slate-800 shadow-md hover:shadow-lg transition-all duration-200 flex flex-col justify-between">
        <div>
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-bold uppercase tracking-wider text-slate-400">
              {label}
            </span>
            <div className="w-8 h-8 rounded-xl bg-white/10 text-white flex items-center justify-center shrink-0">
              {icon}
            </div>
          </div>
          <div className="text-3xl font-extrabold text-white tracking-tight my-1 font-mono">
            {value}
          </div>
          {progressPercent !== null && (
            <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden my-2">
              <div
                className={`${progressColor || 'bg-indigo-500'} h-full rounded-full transition-all duration-300`}
                style={{ width: `${progressPercent}%` }}
              />
            </div>
          )}
        </div>
        {(subtext || badge) && (
          <div className="mt-3 pt-2.5 border-t border-slate-800 flex items-center justify-between text-xs">
            {subtext && <span className="text-slate-400 font-medium">{subtext}</span>}
            {badge && (
              <span className="bg-emerald-500/20 text-emerald-300 font-bold px-2 py-0.5 rounded-full text-[10px] border border-emerald-500/30">
                {badge}
              </span>
            )}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl p-5 border border-slate-200/80 shadow-soft hover:shadow-md hover:-translate-y-0.5 transition-all duration-200 flex flex-col justify-between">
      <div>
        <div className="flex items-center justify-between mb-3">
          <span className="text-xs font-bold uppercase tracking-wider text-slate-400">
            {label}
          </span>
          <div className="w-8 h-8 rounded-xl bg-slate-50 text-slate-600 border border-slate-100 flex items-center justify-center shrink-0">
            {icon}
          </div>
        </div>
        <div className="text-3xl font-extrabold text-slate-900 tracking-tight my-1 font-mono">
          {value}
        </div>
        {progressPercent !== null && (
          <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden my-2">
            <div
              className={`${progressColor || 'bg-indigo-600'} h-full rounded-full transition-all duration-300`}
              style={{ width: `${progressPercent}%` }}
            />
          </div>
        )}
      </div>
      {(subtext || badge) && (
        <div className="mt-3 pt-2.5 border-t border-slate-100 flex items-center justify-between text-xs">
          {subtext && <span className="text-slate-500 font-medium">{subtext}</span>}
          {badge && (
            <span className="bg-slate-100 text-slate-700 font-semibold px-2 py-0.5 rounded-full text-[10px] border border-slate-200">
              {badge}
            </span>
          )}
        </div>
      )}
    </div>
  );
};
