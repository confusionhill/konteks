import React, { useEffect, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';

const FRONTS = ['agent', 'bot', 'chat', 'assistant', 'copilot', 'support'];

interface Turn {
  role: 'user' | 'assistant';
  front?: number;
  text: string;
}

const THREAD: Turn[] = [
  { role: 'user', text: 'Find me a hotel in Bali under $150.' },
  { role: 'assistant', front: 0, text: 'Five options fit your budget. Want the one with a pool?' },
  { role: 'user', text: 'The one with a pool, yes.' },
  { role: 'assistant', front: 1, text: 'That\'s the Grand Hyatt. Book for Friday?' },
  { role: 'user', text: 'Book it.' },
  { role: 'assistant', front: 2, text: 'Booked. Check-in Friday, Grand Hyatt, Bali.' },
  { role: 'user', text: 'Wait, what time is check-in?' },
  { role: 'assistant', front: 3, text: '3 PM. Still in our shared context.' },
  { role: 'user', text: 'And what was my budget again?' },
  { role: 'assistant', front: 4, text: 'Under $150 a night.' },
  { role: 'user', text: 'Did I still want the pool?' },
  { role: 'assistant', front: 5, text: 'Pool, yes. We kept that too.' },
];

const MEMORY = ['Bali', 'under $150', 'pool', 'Grand Hyatt', 'Friday'];

export default function HeroPreview() {
  const [step, setStep] = useState(1);

  useEffect(() => {
    const id = setInterval(() => setStep((s) => (s >= THREAD.length ? 1 : s + 1)), 1200);
    return () => clearInterval(id);
  }, []);

  const visible = THREAD.slice(0, step);
  const lastFront = [...visible].reverse().find((t) => t.role === 'assistant');
  const frontLabel = FRONTS[lastFront?.front ?? 0];

  return (
    <div className="w-full max-w-xl mx-auto rounded-2xl bg-white border border-slate-200/80 shadow-soft-indigo overflow-hidden">
      {/* Title bar */}
      <div className="px-4 py-3 bg-slate-50 border-b border-slate-200/80 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-red-400" />
          <div className="w-3 h-3 rounded-full bg-amber-400" />
          <div className="w-3 h-3 rounded-full bg-emerald-400" />
          <span className="ml-1 text-xs font-mono text-slate-500 font-medium">room_9842</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          <span className="text-[11px] font-semibold text-emerald-700">memory on</span>
        </div>
      </div>

      {/* Conversation */}
      <div className="px-5 pt-5 pb-4 space-y-3 h-[300px] overflow-hidden flex flex-col justify-end">
        {visible.map((turn, i) => {
          const isUser = turn.role === 'user';
          return (
            <motion.div
              key={`${turn.text}-${i}`}
              initial={{ y: 8, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
              className={`flex ${isUser ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-[85%] px-3.5 py-2 rounded-xl text-xs leading-relaxed ${
                  isUser
                    ? 'bg-indigo-600 text-white rounded-br-sm'
                    : 'bg-slate-100 text-slate-700 border border-slate-200/70 rounded-bl-sm'
                }`}
              >
                {!isUser && (
                  <span className="block mb-1 text-[10px] font-bold tracking-wide text-indigo-600">
                    {FRONTS[turn.front ?? 0]}
                  </span>
                )}
                {turn.text}
              </div>
            </motion.div>
          );
        })}
      </div>

      {/* Still remembers strip */}
      <div className="mx-4 mb-4 rounded-xl bg-indigo-50/70 border border-indigo-100/80 px-4 py-3">
        <div className="flex items-center justify-between mb-2">
          <span className="text-[11px] font-bold text-indigo-700 uppercase tracking-wide">
            still remembers
          </span>
          <AnimatePresence mode="wait" initial={false}>
            <motion.span
              key={frontLabel}
              initial={{ y: 6, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{ y: -6, opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="text-[11px] font-semibold text-slate-500"
            >
              now on <span className="text-indigo-600">{frontLabel}</span>
            </motion.span>
          </AnimatePresence>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {MEMORY.map((m) => (
            <span
              key={m}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-white border border-indigo-200/70 text-[11px] font-medium text-slate-700"
            >
              <svg className="w-3 h-3 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d="M5 13l4 4L19 7" />
              </svg>
              {m}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}