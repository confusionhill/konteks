import React from 'react';
import type { Message } from '../api/dashboard';
import { Robot, User } from '@phosphor-icons/react';

interface MessageListProps {
  messages: Message[];
}

export const MessageList: React.FC<MessageListProps> = ({ messages }) => {
  if (messages.length === 0) {
    return (
      <div className="bg-surface rounded-xl p-6 border border-slate-100 shadow-soft">
        <p className="text-slate-400 italic text-sm">No messages recorded in this room.</p>
      </div>
    );
  }

  const formatTime = (isoString: string) => {
    if (!isoString) return '';
    try {
      return new Date(isoString).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return isoString;
    }
  };

  return (
    <div className="bg-surface rounded-xl border border-slate-100 shadow-soft overflow-hidden">
      <div className="p-4 bg-slate-50/80 border-b border-slate-100 flex items-center justify-between">
        <h3 className="text-sm font-bold text-slate-900 uppercase tracking-wider">
          Message Log ({messages.length})
        </h3>
        <span className="text-xs font-mono text-slate-400">Sequential History</span>
      </div>

      <div className="divide-y divide-slate-100 max-h-[600px] overflow-y-auto">
        {messages.map((msg, idx) => {
          const isUser = msg.role === 'user';
          return (
            <div
              key={msg.id || idx}
              className={`p-4 transition-colors ${
                isUser ? 'bg-surface' : 'bg-slate-50/50'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <span
                  className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-bold uppercase tracking-wider ${
                    isUser
                      ? 'bg-slate-900 text-white'
                      : 'bg-primary-100 text-primary-700'
                  }`}
                >
                  {isUser ? <User size={12} weight="bold" /> : <Robot size={12} weight="bold" />}
                  {msg.role}
                </span>

                <span className="text-xs font-mono text-slate-400">
                  {formatTime(msg.created_at)}
                </span>
              </div>

              <p className="text-sm text-slate-800 font-sans leading-relaxed whitespace-pre-wrap pl-1">
                {msg.content}
              </p>
            </div>
          );
        })}
      </div>
    </div>
  );
};
