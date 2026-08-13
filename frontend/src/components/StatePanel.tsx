import React from 'react';
import type { RoomState } from '../api/dashboard';
import { Sliders, Tag, FileText, Link as LinkIcon, ArrowRight } from 'lucide-react';

interface StatePanelProps {
  state: RoomState | null;
}

export const StatePanel: React.FC<StatePanelProps> = ({ state }) => {
  if (!state) {
    return (
      <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 shadow-xl backdrop-blur-xl">
        <p className="text-slate-500 italic text-xs">No state recorded for this room.</p>
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
    <div className="bg-slate-900/70 border border-slate-800 rounded-2xl p-6 shadow-xl space-y-6 backdrop-blur-xl font-sans">
      <div className="flex items-center justify-between border-b border-slate-800 pb-4">
        <h3 className="text-base font-bold text-white flex items-center space-x-2">
          <Sliders className="h-4 w-4 text-indigo-400" />
          <span>Current Room State</span>
        </h3>
        <div className="flex items-center space-x-2">
          {state.strategy_used && (
            <span className="inline-flex items-center space-x-1 bg-slate-950 border border-slate-800 text-slate-300 font-mono font-semibold px-2.5 py-0.5 rounded-full text-[10px]">
              <span>{state.strategy_used}</span>
            </span>
          )}
          {state.active_topic && (
            <span className="inline-flex items-center space-x-1 bg-indigo-950/80 border border-indigo-500/30 text-indigo-300 font-semibold px-2.5 py-0.5 rounded-full text-[10px]">
              <Tag className="h-3 w-3" />
              <span>{state.active_topic}</span>
            </span>
          )}
        </div>
      </div>

      {/* Resolved References */}
      <div>
        <h4 className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-2 flex items-center space-x-1.5">
          <LinkIcon className="h-3.5 w-3.5 text-indigo-400" />
          <span>Resolved References (Pronouns)</span>
        </h4>
        {references.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {references.map(([src, tgt]) => (
              <span
                key={src}
                className="inline-flex items-center space-x-1.5 bg-slate-950 border border-slate-800 px-3 py-1 rounded-xl text-xs"
              >
                <span className="font-mono font-bold text-indigo-400">"{src}"</span>
                <ArrowRight className="h-3 w-3 text-slate-500" />
                <span className="font-semibold text-slate-200">{tgt}</span>
              </span>
            ))}
          </div>
        ) : (
          <p className="text-slate-500 italic text-xs">No pronoun references resolved yet.</p>
        )}
      </div>

      {/* Summary Section */}
      <div>
        <h4 className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-2 flex items-center space-x-1.5">
          <FileText className="h-3.5 w-3.5 text-slate-500" />
          <span>Conversation Summary</span>
        </h4>
        {state.summary ? (
          <blockquote className="border-l-2 border-indigo-500 pl-3 py-2 bg-slate-950/60 rounded-r-xl text-slate-300 text-xs italic leading-relaxed">
            "{state.summary}"
          </blockquote>
        ) : (
          <p className="text-slate-500 italic text-xs">No summary generated yet.</p>
        )}
      </div>

      {/* Structured Variables Table */}
      <div>
        <h4 className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-2">
          State Variables
        </h4>
        {stateVars.length > 0 ? (
          <div className="rounded-xl border border-slate-800 overflow-hidden bg-slate-950/60">
            <div className="divide-y divide-slate-800">
              {stateVars.map(([key, val]) => (
                <div key={key} className="flex items-center justify-between p-2.5 text-xs font-mono">
                  <span className="font-semibold text-slate-400">{key}</span>
                  <span className="bg-slate-900 px-2 py-0.5 rounded-lg border border-slate-800 text-indigo-300 font-medium max-w-[200px] truncate">
                    {val}
                  </span>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <p className="text-slate-500 italic text-xs">No variables extracted yet.</p>
        )}
      </div>
    </div>
  );
};
