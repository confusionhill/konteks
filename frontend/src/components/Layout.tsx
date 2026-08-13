import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Cpu, ChartBar, Key } from '@phosphor-icons/react';

interface LayoutProps {
  children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const location = useLocation();

  const handleApiKeyChange = () => {
    const current = localStorage.getItem('konteks_api_key') || 'dev_secret_key_123';
    const updated = prompt('Enter Konteks API Key (Bearer Token):', current);
    if (updated !== null) {
      localStorage.setItem('konteks_api_key', updated.trim());
      window.location.reload();
    }
  };

  return (
    <div className="min-h-[100dvh] bg-background text-slate-900 flex flex-col relative overflow-hidden">
      {/* Soft Background Gradient Orbs (Corporate Trust Aesthetic) */}
      <div className="absolute -top-32 -left-32 w-96 h-96 bg-primary-100 rounded-full blur-3xl opacity-40 pointer-events-none -z-10" />
      <div className="absolute top-1/3 -right-32 w-96 h-96 bg-secondary-100 rounded-full blur-3xl opacity-30 pointer-events-none -z-10" />

      {/* Navigation Header */}
      <header className="bg-surface/80 backdrop-blur-md border-b border-slate-200/80 sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-8">
            <Link to="/dashboard" className="flex items-center gap-2 group">
              <img src="/konteksicon.svg" alt="Konteks Logo" className="w-8 h-8 drop-shadow-sm" />
              <span className="font-extrabold text-xl tracking-tight text-slate-900 group-hover:text-primary-600 transition-colors">
                konteks
              </span>
              <span className="text-[10px] font-mono font-bold uppercase tracking-wider bg-indigo-50 text-indigo-700 px-2 py-0.5 rounded-full border border-indigo-100">
                Dashboard
              </span>
            </Link>

            <nav className="hidden md:flex items-center gap-1">
              <Link
                to="/dashboard"
                className={`px-3 py-1.5 rounded-lg text-sm font-semibold transition-colors ${
                  location.pathname === '/dashboard'
                    ? 'bg-indigo-50 text-primary-600'
                    : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'
                }`}
              >
                Overview
              </Link>
            </nav>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={handleApiKeyChange}
              className="inline-flex items-center gap-1.5 text-xs font-mono font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 px-3 py-1.5 rounded-lg transition-colors border border-slate-200"
              title="Set Bearer Token"
            >
              <Key size={14} className="text-primary-600" />
              API Key
            </button>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 py-8">
        {children}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-200/80 bg-surface/50 py-6">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 font-medium">
          <div className="flex items-center gap-2 mb-2 sm:mb-0">
            <ChartBar size={16} className="text-primary-600" />
            <span>Konteks CaaS Infrastructure</span>
          </div>
          <span>Context-as-a-Service Engine &bull; Phase 1 MVP</span>
        </div>
      </footer>
    </div>
  );
};
