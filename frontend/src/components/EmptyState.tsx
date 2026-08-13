import React from 'react';
import { Inbox } from 'lucide-react';

interface EmptyStateProps {
  title?: string;
  description?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  title = 'No rooms found',
  description = 'Create a conversation room via the POST /v1/rooms API endpoint to get started.',
}) => (
  <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-12 text-center my-6 backdrop-blur-xl shadow-xl">
    <div className="w-14 h-14 rounded-2xl bg-indigo-950/60 border border-indigo-500/30 text-indigo-400 flex items-center justify-center mx-auto mb-4">
      <Inbox className="h-7 w-7" />
    </div>
    <h3 className="text-lg font-bold text-slate-100 mb-1">{title}</h3>
    <p className="text-slate-400 text-xs max-w-md mx-auto leading-relaxed">{description}</p>
  </div>
);
