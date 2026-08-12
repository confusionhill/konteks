import React from 'react';
import type { RoomState } from '../api/dashboard';
import { Tag, FileText, Faders, Link as LinkIcon, ArrowRight, Sliders } from '@phosphor-icons/react';

interface StatePanelProps {
  state: RoomState | null;
}

export const StatePanel: React.FC<StatePanelProps> = ({ state }) => {
  if (!state) {
    return (
      <div className="bg-surface rounded-xl p-6 border border-slate-100 shadow-soft">
        <p className="text-slate-400 italic text-sm">No state recorded for this room.</p>
      </div>
    );
  }

  const parseStateVars = (stateJson: Record<string, unknown> | null): Array<[string, string]> => {
    if (!stateJson) return [];
    try {
      return Object.entries(stateJson).map(([k, v]) => [
        k,
        typeof v === 'object' ? JSON.stringify(v) : String(v),
      ]);
    } catch {
      return [];
    }
  };

  const stateVars = parseStateVars(state.state);
  const references = state.references ? Object.entries(state.references) : [];

  return (
    <div className="bg-surface rounded-xl p-6 border border-slate-100 shadow-soft space-y-6">
      <div className="flex items-center justify-between border-b border-slate-100 pb-4">
        <h3 className="text-lg font-bold text-slate-900 flex items-center gap-2">
          <Faders size={20} className="text-primary-600" />
          Current Room State
        </h3>
        <div className="flex items-center gap-2">
          {state.strategy_used && (
            <span className="inline-flex items-center gap-1 bg-slate-100 text-slate-700 font-mono font-semibold px-2.5 py-0.5 rounded-full text-xs border border-slate-200">
              <Sliders size={12} weight="bold" />
              {state.strategy_used}
            </span>
          )}
          {state.active_topic && (
            <span className="inline-flex items-center gap-1.5 bg-indigo-50 text-indigo-700 font-semibold px-3 py-1 rounded-full text-xs border border-indigo-100">
              <Tag size={12} weight="bold" />
              {state.active_topic}
            </span>
          )}
        </div>
      </div>

      {/* Phase 2 Resolved References Section */}
      <div>
        <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 flex items-center gap-1.5">
          <LinkIcon size={14} className="text-primary-600" />
          Resolved References (Pronouns)
        </h4>
        {references.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {references.map(([src, tgt]) => (
              <span
                key={src}
                className="inline-flex items-center gap-1.5 bg-indigo-50/80 text-xs text-slate-800 px-3 py-1 rounded-lg border border-indigo-100 font-medium"
              >
                <span className="font-mono font-bold text-primary-600">"{src}"</span>
                <ArrowRight size={12} className="text-slate-400" />
                <span className="font-semibold text-slate-900">{tgt}</span>
              </span>
            ))}
          </div>
        ) : (
          <p className="text-slate-400 italic text-sm">No pronoun references resolved yet.</p>
        )}
      </div>

      {/* Summary Section */}
      <div>
        <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 flex items-center gap-1.5">
          <FileText size={14} className="text-slate-400" />
          Conversation Summary
        </h4>
        {state.summary ? (
          <blockquote className="border-l-4 border-primary-600 pl-4 py-2 bg-slate-50/80 rounded-r-lg text-slate-700 text-sm italic leading-relaxed">
            "{state.summary}"
          </blockquote>
        ) : (
          <p className="text-slate-400 italic text-sm">No summary generated yet.</p>
        )}
      </div>

      {/* Structured Variables Table */}
      <div>
        <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          State Variables
        </h4>
        {stateVars.length > 0 ? (
          <div className="rounded-lg border border-slate-100 overflow-hidden bg-slate-50/50">
            <div className="divide-y divide-slate-100">
              {stateVars.map(([key, val]) => (
                <div key={key} className="flex items-center justify-between p-3 text-xs font-mono">
                  <span className="font-semibold text-slate-600">{key}</span>
                  <span className="bg-surface px-2.5 py-1 rounded border border-slate-200 text-primary-700 font-medium max-w-[200px] truncate">
                    {val}
                  </span>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <p className="text-slate-400 italic text-sm">No variables extracted yet.</p>
        )}
      </div>
    </div>
  );
};
