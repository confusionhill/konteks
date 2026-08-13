import React, { useState, useRef, useEffect } from 'react';
import { Link, useLocation, useNavigate, Outlet } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import {
  Layers,
  ChevronDown,
  LayoutDashboard,
  MessageSquare,
  Key,
  Settings,
  FolderKanban,
  LogOut,
  User as UserIcon,
  Check,
  Plus,
} from 'lucide-react';

interface LayoutProps {
  children?: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, projects, activeProject, selectProject, logout } = useAuth();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const navTabs = [
    { name: 'Overview', path: '/dashboard', icon: LayoutDashboard },
    { name: 'Rooms', path: '/dashboard/rooms', icon: MessageSquare },
    { name: 'API Keys', path: '/dashboard/api-keys', icon: Key },
    { name: 'Settings', path: '/dashboard/settings', icon: Settings },
  ];

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 font-sans flex flex-col relative overflow-hidden">
      {/* Background Decorative Glow */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full max-w-7xl h-96 bg-indigo-600/10 rounded-full blur-3xl pointer-events-none" />

      {/* Top Header Bar */}
      <header className="border-b border-slate-800/80 bg-slate-900/80 backdrop-blur-xl sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-6">
            <Link to="/projects" className="flex items-center space-x-3 group">
              <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-md shadow-indigo-500/20 group-hover:scale-105 transition-transform">
                <Layers className="h-5 w-5 text-white" />
              </div>
              <span className="text-xl font-bold tracking-tight text-white group-hover:text-indigo-300 transition-colors">
                konteks
              </span>
            </Link>

            <span className="text-slate-700 h-5 w-px" />

            {/* Project Switcher Dropdown */}
            <div className="relative" ref={dropdownRef}>
              <button
                onClick={() => setDropdownOpen(!dropdownOpen)}
                className="flex items-center space-x-2 px-3 py-1.5 rounded-xl bg-slate-800/70 hover:bg-slate-800 border border-slate-700/80 text-xs font-semibold text-slate-200 transition-all"
              >
                <FolderKanban className="h-4 w-4 text-indigo-400 shrink-0" />
                <span className="max-w-[140px] sm:max-w-[200px] truncate">
                  {activeProject ? activeProject.name : 'Select Project'}
                </span>
                <ChevronDown className="h-3.5 w-3.5 text-slate-400 shrink-0" />
              </button>

              {dropdownOpen && (
                <div className="absolute left-0 mt-2 w-64 bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl py-2 z-50 animate-in fade-in zoom-in-95 duration-100">
                  <div className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400">
                    Switch Project
                  </div>

                  <div className="max-h-56 overflow-y-auto py-1">
                    {projects.map((p) => (
                      <button
                        key={p.id}
                        onClick={() => {
                          selectProject(p);
                          setDropdownOpen(false);
                          navigate('/dashboard');
                        }}
                        className={`w-full text-left px-3.5 py-2 text-xs flex items-center justify-between transition-colors ${
                          activeProject?.id === p.id
                            ? 'bg-indigo-600/20 text-indigo-300 font-semibold'
                            : 'text-slate-300 hover:bg-slate-800/60'
                        }`}
                      >
                        <span className="truncate">{p.name}</span>
                        {activeProject?.id === p.id && <Check className="h-3.5 w-3.5 text-indigo-400 shrink-0 ml-2" />}
                      </button>
                    ))}
                  </div>

                  <div className="border-t border-slate-800 mt-1 pt-1.5 px-2">
                    <button
                      onClick={() => {
                        setDropdownOpen(false);
                        navigate('/projects');
                      }}
                      className="w-full text-left px-2.5 py-1.5 rounded-xl text-xs font-semibold text-indigo-400 hover:bg-indigo-950/50 flex items-center space-x-2 transition-colors"
                    >
                      <Plus className="h-3.5 w-3.5" />
                      <span>All Projects Hub</span>
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* User Badge & Logout */}
          <div className="flex items-center space-x-4">
            <div className="hidden sm:flex items-center space-x-2 bg-slate-900 border border-slate-800 rounded-xl px-3 py-1 text-xs">
              <div className="h-6 w-6 rounded-lg bg-indigo-600/20 border border-indigo-500/30 text-indigo-400 flex items-center justify-center">
                <UserIcon className="h-3.5 w-3.5" />
              </div>
              <span className="font-semibold text-slate-200">{user?.name || user?.email}</span>
            </div>

            <button
              onClick={logout}
              className="text-slate-400 hover:text-slate-200 p-2 rounded-xl border border-slate-800 hover:border-slate-700 hover:bg-slate-900 transition-all"
              title="Sign Out"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Navigation Tabs Bar */}
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center space-x-1 overflow-x-auto no-scrollbar pt-1">
          {navTabs.map((tab) => {
            const Icon = tab.icon;
            const isActive =
              tab.path === '/dashboard'
                ? location.pathname === '/dashboard' || location.pathname === '/dashboard/'
                : location.pathname.startsWith(tab.path);

            return (
              <Link
                key={tab.path}
                to={tab.path}
                className={`flex items-center space-x-2 px-4 py-3 text-xs font-semibold border-b-2 transition-all shrink-0 ${
                  isActive
                    ? 'border-indigo-500 text-indigo-400 bg-indigo-500/5'
                    : 'border-transparent text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
                }`}
              >
                <Icon className="h-4 w-4" />
                <span>{tab.name}</span>
              </Link>
            );
          })}
        </div>
      </header>

      {/* Main Page Body */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 relative z-10">
        {children || <Outlet />}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/80 bg-slate-900/40 py-4 text-xs text-slate-500">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-2">
          <div className="flex items-center space-x-2">
            <Layers className="h-4 w-4 text-indigo-400" />
            <span>konteks CaaS Infrastructure &bull; Multi-Tenant Scoped Architecture</span>
          </div>
          <span>Active Project: <strong className="text-slate-400">{activeProject?.name || 'None'}</strong></span>
        </div>
      </footer>
    </div>
  );
};
