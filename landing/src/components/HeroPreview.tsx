import React, { useState } from 'react';

export default function HeroPreview() {
  const [activeTab, setActiveTab] = useState<'compiled' | 'raw'>('compiled');

  return (
    <div className="w-full max-w-xl mx-auto rounded-xl bg-white border border-slate-200/80 shadow-soft-indigo overflow-hidden transition-all duration-300 hover:shadow-soft-indigo-hover">
      {/* Window Titlebar */}
      <div className="px-4 py-3 bg-slate-50 border-b border-slate-200/80 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-red-400"></div>
          <div className="w-3 h-3 rounded-full bg-amber-400"></div>
          <div className="w-3 h-3 rounded-full bg-emerald-400"></div>
          <span className="ml-2 text-xs font-mono text-slate-500 font-medium">room_9842 / context_compiled</span>
        </div>
        <div className="flex items-center gap-1 bg-slate-200/70 p-0.5 rounded-lg text-xs font-medium text-slate-600">
          <button
            type="button"
            onClick={() => setActiveTab('compiled')}
            className={`px-2.5 py-1 rounded-md transition-all ${
              activeTab === 'compiled' ? 'bg-white text-indigo-600 shadow-sm font-semibold' : 'hover:text-slate-900'
            }`}
          >
            Konteks Lean Package
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('raw')}
            className={`px-2.5 py-1 rounded-md transition-all ${
              activeTab === 'raw' ? 'bg-white text-red-600 shadow-sm font-semibold' : 'hover:text-slate-900'
            }`}
          >
            Raw Unmanaged Prompt
          </button>
        </div>
      </div>

      {/* Content View */}
      <div className="p-5 font-mono text-xs">
        {activeTab === 'compiled' ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between pb-2 border-b border-slate-100">
              <span className="text-emerald-600 font-semibold flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                Context Ready for LLM
              </span>
              <span className="text-slate-400">Payload Size: <strong className="text-slate-700">142 tokens</strong></span>
            </div>

            <div className="space-y-2 text-slate-700">
              <div className="p-2.5 rounded-lg bg-indigo-50/60 border border-indigo-100/80">
                <span className="text-indigo-600 font-bold block mb-1">▶ SUMMARY:</span>
                User comparing Bali hotels. Preferred budget &lt;$150/night with pool.
              </div>

              <div className="p-2.5 rounded-lg bg-violet-50/60 border border-violet-100/80">
                <span className="text-violet-600 font-bold block mb-1">▶ ACTIVE STATE & RESOLVED ENTITIES:</span>
                {"{ location: 'Bali', entity: 'hotel_123 (Grand Hyatt)', date: 'Friday' }"}
              </div>

              <div className="p-2.5 rounded-lg bg-slate-50 border border-slate-200/80">
                <span className="text-slate-500 font-bold block mb-1">▶ RELEVANT TURN:</span>
                User: "Does it have a pool?"
              </div>
            </div>

            <div className="pt-2 flex items-center justify-between text-[11px] text-slate-500">
              <span>Token Savings: <strong className="text-emerald-600">96.4% lower cost</strong></span>
              <span>State Resolution: <strong className="text-indigo-600">Active & Compact</strong></span>
            </div>
          </div>
        ) : (
          <div className="space-y-3 text-slate-500">
            <div className="flex items-center justify-between pb-2 border-b border-slate-100">
              <span className="text-red-600 font-semibold flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-red-500"></span>
                Bloated Chat History Payload
              </span>
              <span className="text-slate-400">Payload Size: <strong className="text-red-600">4,280 tokens</strong></span>
            </div>

            <div className="p-3 bg-red-50/40 border border-red-100 rounded-lg text-slate-600 space-y-1.5 opacity-90 max-h-48 overflow-y-auto">
              <p className="text-slate-400">[Lines 1-45: System Prompt & Previous Hotel List...]</p>
              <p>User: "Find me a hotel in Bali."</p>
              <p>Assistant: "Here are 5 options: 1. Grand Hyatt ($140), 2. Padma Resort ($220)..."</p>
              <p>User: "Which one is cheaper?"</p>
              <p>Assistant: "Option 1 Grand Hyatt is cheaper..."</p>
              <p className="text-red-500 font-semibold">[Warning: Pronoun "it" ambiguous without state]</p>
              <p>User: "Does it have a pool?"</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
