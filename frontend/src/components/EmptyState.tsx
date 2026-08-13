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
  <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-12 text-center my-6">
    <div className="w-16 h-16 rounded-2xl bg-indigo-50 text-indigo-600 flex items-center justify-center mx-auto mb-4">
      <Inbox className="h-8 w-8" />
    </div>
    <h3 className="text-xl font-bold text-slate-900 mb-2">{title}</h3>
    <p className="text-slate-500 text-sm max-w-md mx-auto leading-relaxed">{description}</p>
  </div>
);
