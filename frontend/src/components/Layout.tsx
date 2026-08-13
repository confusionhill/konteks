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
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans flex flex-col relative overflow-hidden">
      {/* Soft Background Gradient Orbs (Corporate Trust Aesthetic) */}
      <div className="absolute -top-32 -left-32 w-96 h-96 bg-indigo-100 rounded-full blur-3xl opacity-60 pointer-events-none -z-10" />
      <div className="absolute top-1/3 -right-32 w-96 h-96 bg-violet-100 rounded-full blur-3xl opacity-50 pointer-events-none -z-10" />

      {/* Top Header Bar */}
      <header className="bg-white/80 backdrop-blur-md border-b border-slate-200/80 sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-6">
            <Link to="/projects" className="flex items-center space-x-3 group">
              <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-600 flex items-center justify-center shadow-button group-hover:scale-105 transition-transform">
                <Layers className="h-5 w-5 text-white" />
              </div>
              <span className="text-xl font-bold tracking-tight text-slate-900 group-hover:text-indigo-600 transition-colors font-sans">
                konteks
              </span>
            </Link>

            <span className="text-slate-300 h-5 w-px" />

            {/* Project Switcher Dropdown */}
            <div className="relative" ref={dropdownRef}>
              <button
                onClick={() => setDropdownOpen(!dropdownOpen)}
                className="flex items-center space-x-2 px-3 py-1.5 rounded-lg bg-slate-100 hover:bg-slate-200/80 border border-slate-200 text-xs font-semibold text-slate-800 transition-all"
              >
                <FolderKanban className="h-4 w-4 text-indigo-600 shrink-0" />
                <span className="max-w-[140px] sm:max-w-[200px] truncate">
                  {activeProject ? activeProject.name : 'Select Project'}
                </span>
                <ChevronDown className="h-3.5 w-3.5 text-slate-500 shrink-0" />
              </button>

              {dropdownOpen && (
                <div className="absolute left-0 mt-2 w-64 bg-white border border-slate-200 rounded-xl shadow-lifted py-2 z-50 animate-in fade-in zoom-in-95 duration-100">
                  <div className="px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-500">
                    Switch Project Workspace
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
                            ? 'bg-indigo-50 text-indigo-600 font-semibold'
                            : 'text-slate-700 hover:bg-slate-50'
                        }`}
                      >
                        <span className="truncate">{p.name}</span>
                        {activeProject?.id === p.id && <Check className="h-3.5 w-3.5 text-indigo-600 shrink-0 ml-2" />}
                      </button>
                    ))}
                  </div>

                  <div className="border-t border-slate-100 mt-1 pt-1.5 px-2">
                    <button
                      onClick={() => {
                        setDropdownOpen(false);
                        navigate('/projects');
                      }}
                      className="w-full text-left px-2.5 py-1.5 rounded-lg text-xs font-semibold text-indigo-600 hover:bg-indigo-50 flex items-center space-x-2 transition-colors"
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
            <div className="hidden sm:flex items-center space-x-2 bg-slate-100 border border-slate-200 rounded-lg px-3 py-1 text-xs">
              <div className="h-6 w-6 rounded-md bg-indigo-100 text-indigo-600 flex items-center justify-center font-semibold">
                <UserIcon className="h-3.5 w-3.5" />
              </div>
              <span className="font-semibold text-slate-800">{user?.name || user?.email}</span>
            </div>

            <button
              onClick={logout}
              className="text-slate-500 hover:text-slate-800 p-2 rounded-lg border border-slate-200 hover:bg-slate-100 transition-all"
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
                    ? 'border-indigo-600 text-indigo-600 bg-indigo-50/50'
                    : 'border-transparent text-slate-600 hover:text-slate-900 hover:bg-slate-50'
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
      <footer className="border-t border-slate-200/80 bg-white/50 py-6 text-xs text-slate-500">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-2">
          <div className="flex items-center space-x-2 font-medium">
            <Layers className="h-4 w-4 text-indigo-600" />
            <span>konteks CaaS Infrastructure &bull; Context-as-a-Service Engine</span>
          </div>
          <span className="font-medium">Active Workspace: <strong className="text-slate-700">{activeProject?.name || 'None'}</strong></span>
        </div>
      </footer>
    </div>
  );
};
