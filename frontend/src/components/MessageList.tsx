import React from 'react';
import type { Message } from '../api/dashboard';
import { Robot, User, Coins } from '@phosphor-icons/react';

interface MessageListProps {
  messages: Message[];
  hasMore: boolean;
  onLoadMore: () => void;
  loadingMore: boolean;
}

export const MessageList: React.FC<MessageListProps> = ({ messages, hasMore, onLoadMore, loadingMore }) => {
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
    <div className="bg-surface rounded-xl border border-slate-100 shadow-soft overflow-hidden flex flex-col h-full">
      <div className="p-4 bg-slate-50/80 border-b border-slate-100 flex items-center justify-between shrink-0">
        <h3 className="text-sm font-bold text-slate-900 uppercase tracking-wider">
          Message Log ({messages.length})
        </h3>
        <span className="text-xs font-mono text-slate-400">Sequential History</span>
      </div>

      <div className="divide-y divide-slate-100 max-h-[700px] overflow-y-auto p-1">
        {hasMore && (
          <div className="flex justify-center p-2 mb-2 border-b border-slate-50">
            <button
              onClick={onLoadMore}
              disabled={loadingMore}
              className="text-xs font-bold text-primary-600 hover:text-primary-700 bg-indigo-50 hover:bg-indigo-100 px-4 py-1.5 rounded-lg border border-indigo-100 transition-colors disabled:opacity-50 shadow-soft"
            >
              {loadingMore ? 'Loading older...' : 'Load Older Messages'}
            </button>
          </div>
        )}
        {messages.map((msg, idx) => {
          const isUser = msg.role === 'user';
          const hasTokens = msg.tokens_input !== undefined || msg.tokens_output !== undefined;
          const textContent = msg.content ? msg.content.trim() : '';

          return (
            <div
              key={msg.id || idx}
              className={`p-4 transition-colors rounded-lg my-1 ${
                isUser ? 'bg-surface border border-slate-100 shadow-xs' : 'bg-slate-50/60'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
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

                  {/* Render token & cost badge for new tracked messages */}
                  {hasTokens && (
                    <span className="inline-flex items-center gap-1 text-[11px] font-mono text-slate-500 bg-slate-100 px-2 py-0.5 rounded border border-slate-200">
                      <Coins size={12} className="text-amber-500" />
                      {msg.tokens_input !== undefined ? `in: ${msg.tokens_input}` : ''}
                      {msg.tokens_output !== undefined ? ` | out: ${msg.tokens_output}` : ''}
                      {msg.cost_total !== undefined ? ` ($${msg.cost_total.toFixed(6)})` : ''}
                    </span>
                  )}
                </div>

                <span className="text-xs font-mono text-slate-400">
                  {formatTime(msg.created_at)}
                </span>
              </div>

              <div className="text-sm text-slate-800 font-sans leading-relaxed whitespace-pre-wrap pl-1 break-words">
                {textContent ? (
                  textContent
                ) : (
                  <span className="italic text-slate-400 text-xs">(No text content recorded)</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
