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
      <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 shadow-xl backdrop-blur-xl">
        <p className="text-slate-500 italic text-xs">No messages recorded in this room.</p>
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
    <div className="bg-slate-900/70 border border-slate-800 rounded-2xl shadow-xl overflow-hidden flex flex-col h-full backdrop-blur-xl font-sans">
      <div className="p-4 bg-slate-950/80 border-b border-slate-800 flex items-center justify-between shrink-0">
        <h3 className="text-xs font-bold text-white uppercase tracking-wider">
          Message Log ({messages.length})
        </h3>
        <span className="text-[10px] font-mono text-slate-400">Sequential History</span>
      </div>

      <div className="divide-y divide-slate-800/60 max-h-[700px] overflow-y-auto p-2 space-y-2">
        {hasMore && (
          <div className="flex justify-center p-2 mb-2">
            <button
              onClick={onLoadMore}
              disabled={loadingMore}
              className="text-xs font-semibold text-indigo-400 hover:text-indigo-300 bg-indigo-950/60 hover:bg-indigo-950 px-4 py-1.5 rounded-xl border border-indigo-500/30 transition-all disabled:opacity-50"
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
              className={`p-3.5 rounded-xl border transition-colors ${
                isUser
                  ? 'bg-slate-950/80 border-slate-800'
                  : 'bg-indigo-950/30 border-indigo-500/20'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center space-x-2">
                  <span
                    className={`inline-flex items-center space-x-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider ${
                      isUser
                        ? 'bg-slate-800 text-slate-200'
                        : 'bg-indigo-600 text-white'
                    }`}
                  >
                    {isUser ? <User className="h-3 w-3" /> : <Bot className="h-3 w-3" />}
                    <span>{msg.role}</span>
                  </span>

                  {hasTokens && (
                    <span className="inline-flex items-center space-x-1 text-[10px] font-mono text-slate-400 bg-slate-950 px-2 py-0.5 rounded-md border border-slate-800">
                      <Coins className="h-3 w-3 text-amber-400" />
                      <span>
                        {msg.tokens_input !== undefined ? `in: ${msg.tokens_input}` : ''}
                        {msg.tokens_output !== undefined ? ` | out: ${msg.tokens_output}` : ''}
                        {msg.cost_total !== undefined ? ` ($${msg.cost_total.toFixed(6)})` : ''}
                      </span>
                    </span>
                  )}
                </div>

                <span className="text-[10px] font-mono text-slate-500">
                  {formatTime(msg.created_at)}
                </span>
              </div>

              <div className="text-xs text-slate-200 leading-relaxed whitespace-pre-wrap pl-0.5 break-words">
                {textContent ? (
                  textContent
                ) : (
                  <span className="italic text-slate-500 text-xs">(No text content recorded)</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
