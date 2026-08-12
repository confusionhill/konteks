import React, { useState } from 'react';

export default function QuickStart() {
  const [activeLang, setActiveLang] = useState<'python' | 'ts' | 'curl'>('python');
  const [copied, setCopied] = useState(false);

  const codeExamples = {
    python: `# 1. Push user message to room
konteks.add_message(
    room_id="room_123",
    role="user",
    content="Find me a hotel in Bali with a pool under $150."
)

# 2. Get optimal context package for LLM
context = konteks.get_context(room_id="room_123")

# 3. Pass clean context directly into OpenAI / Claude
response = openai.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": f"Context: {context['summary']}\\nState: {context['state']}"},
        {"role": "user", "content": "Does it have a pool?"}
    ]
)`,
    ts: `// 1. Push user message
await caas.pushMessage({
  roomId: 'room_123',
  role: 'user',
  content: 'Find me a hotel in Bali with a pool under $150.',
});

// 2. Retrieve compiled context package
const { context } = await caas.getContext({ roomId: 'room_123' });

// 3. Send clean context to your LLM agent
const response = await openai.chat.completions.create({
  model: 'gpt-4o',
  messages: [
    { role: 'system', content: \`Summary: \${context.summary}\` },
    { role: 'user', content: 'Does it have a pool?' },
  ],
});`,
    curl: `# 1. Create a persistent Room
curl -X POST https://api.konteks.dev/v1/rooms \\
  -H "Authorization: Bearer $KONTEKS_API_KEY"

# 2. Push User Message
curl -X POST https://api.konteks.dev/v1/rooms/room_123/messages \\
  -H "Authorization: Bearer $KONTEKS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"role": "user", "content": "Find me a hotel in Bali."}'

# 3. Get Curated Context Package
curl -X POST https://api.konteks.dev/v1/rooms/room_123/context \\
  -H "Authorization: Bearer $KONTEKS_API_KEY"`
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(codeExamples[activeLang]);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section id="quickstart" className="py-20 md:py-28 bg-white border-t border-slate-200/60">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        
        {/* Section Header */}
        <div className="text-center max-w-3xl mx-auto space-y-4 mb-12">
          <span className="text-xs font-bold uppercase tracking-wider text-indigo-600 px-3 py-1 rounded-full bg-indigo-50 border border-indigo-200/60">
            Developer First
          </span>
          <h2 className="text-3xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
            Integrate in 5 minutes
          </h2>
          <p className="text-slate-600 text-base sm:text-lg">
            No complex SDK setup required. Works with standard REST calls or simple client helpers in your favorite language.
          </p>
        </div>

        {/* Code Window Container */}
        <div className="max-w-3xl mx-auto rounded-xl bg-slate-900 border border-slate-800 shadow-2xl overflow-hidden">
          {/* Header Controls */}
          <div className="px-4 py-3 bg-slate-950/80 border-b border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <button
                onClick={() => setActiveLang('python')}
                className={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${
                  activeLang === 'python' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Python
              </button>
              <button
                onClick={() => setActiveLang('ts')}
                className={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${
                  activeLang === 'ts' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                TypeScript
              </button>
              <button
                onClick={() => setActiveLang('curl')}
                className={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${
                  activeLang === 'curl' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                cURL API
              </button>
            </div>

            <button
              onClick={handleCopy}
              className="text-xs font-medium text-slate-400 hover:text-white px-2.5 py-1 rounded border border-slate-700/60 bg-slate-900 transition-colors"
            >
              {copied ? '✓ Copied' : 'Copy Code'}
            </button>
          </div>

          {/* Code Area */}
          <div className="p-6 font-mono text-xs leading-relaxed overflow-x-auto text-indigo-200">
            <pre><code>{codeExamples[activeLang]}</code></pre>
          </div>
        </div>

      </div>
    </section>
  );
}
