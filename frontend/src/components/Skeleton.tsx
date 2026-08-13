import React from 'react';

export const StatCardSkeleton: React.FC = () => (
  <div className="bg-white rounded-xl p-5 border border-slate-200/80 shadow-soft animate-pulse">
    <div className="flex items-center space-x-3 mb-4">
      <div className="w-10 h-10 rounded-lg bg-slate-200" />
      <div className="h-4 bg-slate-200 rounded w-24" />
    </div>
    <div className="h-8 bg-slate-200 rounded w-20 mb-3" />
    <div className="h-1 bg-slate-200 rounded w-full" />
  </div>
);

export const TableSkeleton: React.FC = () => (
  <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft overflow-hidden animate-pulse">
    <div className="p-4 bg-slate-50 border-b border-slate-200 flex justify-between">
      <div className="h-4 bg-slate-200 rounded w-32" />
      <div className="h-4 bg-slate-200 rounded w-20" />
    </div>
    <div className="divide-y divide-slate-100">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="p-4 flex items-center justify-between">
          <div className="h-4 bg-slate-200 rounded w-48" />
          <div className="h-4 bg-slate-200 rounded w-28" />
          <div className="h-4 bg-slate-200 rounded w-16" />
        </div>
      ))}
    </div>
  </div>
);

export const DetailSkeleton: React.FC = () => (
  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 animate-pulse">
    <div className="bg-white rounded-xl p-6 border border-slate-200/80 shadow-soft h-64">
      <div className="h-6 bg-slate-200 rounded w-36 mb-4" />
      <div className="h-4 bg-slate-200 rounded w-full mb-2" />
      <div className="h-4 bg-slate-200 rounded w-3/4 mb-4" />
      <div className="h-20 bg-slate-200 rounded-lg w-full" />
    </div>
    <div className="bg-white rounded-xl p-6 border border-slate-200/80 shadow-soft h-64">
      <div className="h-6 bg-slate-200 rounded w-36 mb-4" />
      <div className="h-12 bg-slate-200 rounded-lg w-full mb-3" />
      <div className="h-12 bg-slate-200 rounded-lg w-full" />
    </div>
  </div>
);
