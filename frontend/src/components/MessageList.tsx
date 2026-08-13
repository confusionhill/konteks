import React from 'react';
import type { Message } from '../api/dashboard';
import { Bot, User, Coins } from 'lucide-react';

interface MessageListProps {
  messages: Message[];
  hasMore: boolean;
  onLoadMore: () => void;
  loadingMore: boolean;
}

export const MessageList: React.FC<MessageListProps> = ({
  messages,
  hasMore,
  onLoadMore,
  loadingMore,
}) => {
  if (messages.length === 0) {
    return (
      <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6">
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
    <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft overflow-hidden flex flex-col h-full font-sans">
      <div className="p-4 bg-slate-50/80 border-b border-slate-200 flex items-center justify-between shrink-0">
        <h3 className="text-sm font-bold text-slate-900 uppercase tracking-wider">
          Message Log ({messages.length})
        </h3>
        <span className="text-xs font-mono text-slate-400">Sequential History</span>
      </div>

      <div className="divide-y divide-slate-100 max-h-[700px] overflow-y-auto p-2 space-y-2">
        {hasMore && (
          <div className="flex justify-center p-2 mb-2 border-b border-slate-50">
            <button
              onClick={onLoadMore}
              disabled={loadingMore}
              className="text-xs font-bold text-indigo-600 hover:text-indigo-700 bg-indigo-50 hover:bg-indigo-100 px-4 py-1.5 rounded-lg border border-indigo-100 transition-colors disabled:opacity-50 shadow-soft"
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
              className={`p-4 rounded-lg border transition-colors ${
                isUser
                  ? 'bg-white border-slate-200 shadow-xs'
                  : 'bg-slate-50/80 border-slate-100'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center space-x-2">
                  <span
                    className={`inline-flex items-center space-x-1 px-2.5 py-0.5 rounded-full text-xs font-bold uppercase tracking-wider ${
                      isUser
                        ? 'bg-slate-900 text-white'
                        : 'bg-indigo-100 text-indigo-700'
                    }`}
                  >
                    {isUser ? <User className="h-3 w-3" /> : <Bot className="h-3 w-3" />}
                    <span>{msg.role}</span>
                  </span>

                  {hasTokens && (
                    <span className="inline-flex items-center space-x-1 text-[11px] font-mono text-slate-500 bg-slate-100 px-2 py-0.5 rounded border border-slate-200">
                      <Coins className="h-3 w-3 text-amber-500" />
                      <span>
                        {msg.tokens_input !== undefined ? `in: ${msg.tokens_input}` : ''}
                        {msg.tokens_output !== undefined ? ` | out: ${msg.tokens_output}` : ''}
                        {msg.cost_total !== undefined ? ` ($${msg.cost_total.toFixed(6)})` : ''}
                      </span>
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
