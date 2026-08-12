import React from 'react';
import { Tray } from '@phosphor-icons/react';

interface EmptyStateProps {
  title?: string;
  description?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  title = 'No rooms found',
  description = 'Create a conversation room via the POST /v1/rooms endpoint to get started.',
}) => (
  <div className="bg-surface rounded-xl p-12 border border-slate-100 shadow-soft text-center my-6">
    <div className="w-16 h-16 rounded-2xl bg-indigo-50 text-primary-600 flex items-center justify-center mx-auto mb-4">
      <Tray size={32} weight="duotone" />
    </div>
    <h3 className="text-xl font-bold text-slate-900 mb-2">{title}</h3>
    <p className="text-slate-500 text-sm max-w-md mx-auto leading-relaxed">{description}</p>
  </div>
);
