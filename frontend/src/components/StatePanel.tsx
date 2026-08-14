import React from 'react';
import type { RoomState } from '../api/dashboard';
import { Sliders, Tag, FileText } from 'lucide-react';

interface StatePanelProps {
  state: RoomState | null;
}

export const StatePanel: React.FC<StatePanelProps> = ({ state }) => {
  if (!state) {
    return (
      <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6">
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

  return (
    <div className="bg-white rounded-xl border border-slate-200/80 shadow-soft p-6 space-y-6 font-sans">
      <div className="flex items-center justify-between border-b border-slate-100 pb-4">
        <h3 className="text-lg font-bold text-slate-900 flex items-center space-x-2">
          <Sliders className="h-5 w-5 text-indigo-600" />
          <span>Current Room State</span>
        </h3>
        <div className="flex items-center space-x-2">
          {state.strategy_used && (
            <span className="inline-flex items-center space-x-1 bg-slate-100 border border-slate-200 text-slate-700 font-mono font-semibold px-2.5 py-0.5 rounded-full text-xs">
              <span>{state.strategy_used}</span>
            </span>
          )}
          {state.active_topic && (
            <span className="inline-flex items-center space-x-1 bg-indigo-50 border border-indigo-100 text-indigo-700 font-semibold px-3 py-1 rounded-full text-xs">
              <Tag className="h-3.5 w-3.5" />
              <span>{state.active_topic}</span>
            </span>
          )}
        </div>
      </div>



      {/* Summary Section */}
      <div>
        <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 flex items-center space-x-1.5">
          <FileText className="h-4 w-4 text-slate-400" />
          <span>Conversation Summary</span>
        </h4>
        {state.summary ? (
          <blockquote className="border-l-4 border-indigo-600 pl-4 py-2 bg-slate-50/80 rounded-r-lg text-slate-700 text-sm italic leading-relaxed">
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
          <div className="rounded-lg border border-slate-200/80 overflow-hidden bg-slate-50/50">
            <div className="divide-y divide-slate-100">
              {stateVars.map(([key, val]) => (
                <div key={key} className="flex items-center justify-between p-3 text-xs font-mono">
                  <span className="font-semibold text-slate-600">{key}</span>
                  <span className="bg-white px-2.5 py-1 rounded border border-slate-200 text-indigo-700 font-medium max-w-[240px] truncate">
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
