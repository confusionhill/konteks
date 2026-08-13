import React from 'react';

export const StatCardSkeleton: React.FC = () => (
  <div className="bg-slate-900/60 rounded-2xl p-5 border border-slate-800 animate-pulse">
    <div className="flex items-center space-x-3 mb-4">
      <div className="w-9 h-9 rounded-xl bg-slate-800" />
      <div className="h-3.5 bg-slate-800 rounded w-24" />
    </div>
    <div className="h-8 bg-slate-800 rounded w-20 mb-3" />
    <div className="h-1 bg-slate-800 rounded w-full" />
  </div>
);

export const TableSkeleton: React.FC = () => (
  <div className="bg-slate-900/60 rounded-2xl border border-slate-800 overflow-hidden animate-pulse">
    <div className="p-4 bg-slate-950/60 border-b border-slate-800 flex justify-between">
      <div className="h-4 bg-slate-800 rounded w-32" />
      <div className="h-4 bg-slate-800 rounded w-20" />
    </div>
    <div className="divide-y divide-slate-800">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="p-4 flex items-center justify-between">
          <div className="h-4 bg-slate-800 rounded w-48" />
          <div className="h-4 bg-slate-800 rounded w-28" />
          <div className="h-4 bg-slate-800 rounded w-16" />
        </div>
      ))}
    </div>
  </div>
);

export const DetailSkeleton: React.FC = () => (
  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 animate-pulse">
    <div className="bg-slate-900/60 rounded-2xl p-6 border border-slate-800 h-64">
      <div className="h-6 bg-slate-800 rounded w-36 mb-4" />
      <div className="h-4 bg-slate-800 rounded w-full mb-2" />
      <div className="h-4 bg-slate-800 rounded w-3/4 mb-4" />
      <div className="h-20 bg-slate-800 rounded-xl w-full" />
    </div>
    <div className="bg-slate-900/60 rounded-2xl p-6 border border-slate-800 h-64">
      <div className="h-6 bg-slate-800 rounded w-36 mb-4" />
      <div className="h-12 bg-slate-800 rounded-xl w-full mb-3" />
      <div className="h-12 bg-slate-800 rounded-xl w-full" />
    </div>
  </div>
);
