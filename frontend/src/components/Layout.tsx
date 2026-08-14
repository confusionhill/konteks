import React, { useState, useRef, useEffect } from 'react';
import { Link, useLocation, useNavigate, Outlet } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import {
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
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  Brain,
} from 'lucide-react';

interface LayoutProps {
  children?: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, projects, activeProject, selectProject, logout } = useAuth();

  // Collapsed Sidebar State (Persisted in localStorage)
  const [collapsed, setCollapsed] = useState<boolean>(() => {
    return localStorage.getItem('sidebar_collapsed') === 'true';
  });

  const toggleSidebar = () => {
    setCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem('sidebar_collapsed', String(next));
      return next;
    });
  };

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
    { name: 'Memory Artifacts', path: '/dashboard/artifacts', icon: Brain },
    { name: 'Logs', path: '/dashboard/logs', icon: MessageSquare },
    { name: 'API Keys', path: '/dashboard/api-keys', icon: Key },
    { name: 'Settings', path: '/dashboard/settings', icon: Settings },
  ];

  // Helper for Breadcrumbs
  const getBreadcrumbTitle = () => {
    if (location.pathname.startsWith('/dashboard/artifacts')) return 'Memory Artifacts';
    if (location.pathname.startsWith('/dashboard/evaluations/')) return 'Evaluation Detail';
    if (location.pathname.startsWith('/dashboard/logs/')) return 'Log Inspection';
    if (location.pathname.startsWith('/dashboard/logs')) return 'Logs';
    if (location.pathname.startsWith('/dashboard/rooms/')) return 'Log Inspection';
    if (location.pathname.startsWith('/dashboard/rooms')) return 'Logs';
    if (location.pathname.startsWith('/dashboard/api-keys')) return 'API Keys';
    if (location.pathname.startsWith('/dashboard/settings')) return 'Settings';
    return 'Overview';
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans flex relative overflow-x-hidden">
      {/* Background Decorative Gradient Orbs */}
      <div className="fixed -top-32 -left-32 w-96 h-96 bg-indigo-100 rounded-full blur-3xl opacity-60 pointer-events-none z-0" />
      <div className="fixed top-1/3 right-10 w-96 h-96 bg-violet-100 rounded-full blur-3xl opacity-50 pointer-events-none z-0" />

      {/* FIXED VERTICAL LEFT SIDEBAR */}
      <aside
        className={`fixed top-0 left-0 bottom-0 z-40 bg-white border-r border-slate-100 flex flex-col justify-between p-3 transition-all duration-300 ease-in-out ${
          collapsed ? 'w-16' : 'w-64'
        }`}
      >
        <div>
          {/* Sidebar Top Header (Brand Mark & Interactive Hover Expand Button) */}
          {collapsed ? (
            <div className="flex flex-col items-center justify-center my-1.5 relative group">
              <button
                onClick={toggleSidebar}
                className="relative h-10 w-10 rounded-xl bg-transparent hover:bg-slate-100 flex items-center justify-center shrink-0 hover:scale-105 transition-all overflow-hidden"
              >
                {/* Default Logo Mark (fades out on hover) */}
                <img
                  src="/konteksicon.svg"
                  alt="konteks logo"
                  className="h-7 w-7 transition-all duration-200 group-hover:opacity-0 group-hover:scale-75 absolute"
                />

                {/* Open Sidebar Toggle Icon (fades in on hover) */}
                <PanelLeftOpen className="h-5 w-5 text-indigo-600 transition-all duration-200 opacity-0 group-hover:opacity-100 group-hover:scale-100 absolute" />
              </button>

              {/* Floating Tooltip Pill "Open sidebar" */}
              <div className="absolute left-14 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 translate-x-1 group-hover:translate-x-0 z-50">
                <div className="bg-slate-900 text-white text-xs font-semibold px-3 py-1.5 rounded-full shadow-lifted whitespace-nowrap flex items-center space-x-1.5">
                  <PanelLeftOpen className="h-3.5 w-3.5 text-indigo-300" />
                  <span>Open sidebar</span>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-between pb-3 border-b border-slate-100 mb-4 h-12 px-1">
              <Link to="/projects" className="flex items-center space-x-3 overflow-hidden group">
                <img
                  src="/konteksicon.svg"
                  alt="konteks logo"
                  className="h-9 w-9 shrink-0 group-hover:scale-105 transition-transform"
                />
                <span className="text-xl font-bold tracking-tight text-slate-900 font-sans group-hover:text-indigo-600 transition-colors">
                  konteks
                </span>
              </Link>
              <button
                onClick={toggleSidebar}
                className="text-slate-400 hover:text-slate-700 p-1.5 rounded-lg hover:bg-slate-100 transition-all shrink-0"
                title="Collapse Sidebar"
              >
                <PanelLeftClose className="h-4 w-4" />
              </button>
            </div>
          )}

          {/* Project Switcher */}
          <div className="relative mb-3 mt-2" ref={dropdownRef}>
            {collapsed ? (
              <div className="relative group flex justify-center">
                <button
                  onClick={() => setDropdownOpen(!dropdownOpen)}
                  className="w-10 h-10 flex items-center justify-center rounded-xl bg-indigo-50 text-indigo-700 hover:bg-indigo-100 transition-all"
                >
                  <FolderKanban className="h-4 w-4 shrink-0" />
                </button>

                {/* Tooltip Pill */}
                <div className="absolute left-14 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 translate-x-1 group-hover:translate-x-0 z-50">
                  <div className="bg-slate-900 text-white text-xs font-semibold px-3 py-1.5 rounded-full shadow-lifted whitespace-nowrap">
                    Workspace: {activeProject?.name || 'Select Project'}
                  </div>
                </div>
              </div>
            ) : (
              <button
                onClick={() => setDropdownOpen(!dropdownOpen)}
                className="w-full flex items-center justify-between p-2.5 rounded-xl bg-slate-100 hover:bg-slate-200/80 text-xs font-semibold text-slate-800 transition-all"
              >
                <div className="flex items-center space-x-2 truncate">
                  <FolderKanban className="h-4 w-4 text-indigo-600 shrink-0" />
                  <span className="truncate">{activeProject ? activeProject.name : 'Select Project'}</span>
                </div>
                <ChevronDown className="h-3.5 w-3.5 text-slate-500 shrink-0 ml-1" />
              </button>
            )}

            {/* Dropdown Menu */}
            {dropdownOpen && (
              <div
                className={`absolute mt-2 w-64 bg-white border border-slate-200 rounded-xl shadow-lifted py-2 z-50 animate-in fade-in zoom-in-95 duration-100 ${
                  collapsed ? 'left-14 top-0' : 'left-0'
                }`}
              >
                <div className="px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-400">
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
                          ? 'bg-indigo-50 text-indigo-700 font-semibold'
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

          {/* Navigation Items Stack */}
          <nav className="space-y-1.5">
            {!collapsed && (
              <div className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400">
                Navigation
              </div>
            )}

            {navTabs.map((tab) => {
              const Icon = tab.icon;
              const isActive =
                tab.path === '/dashboard'
                  ? location.pathname === '/dashboard' || location.pathname === '/dashboard/'
                  : location.pathname.startsWith(tab.path);

              return (
                <div key={tab.path} className="relative group flex justify-center">
                  <Link
                    to={tab.path}
                    className={`flex items-center transition-all duration-150 ${
                      collapsed
                        ? 'w-10 h-10 justify-center rounded-xl'
                        : 'w-full space-x-3 px-3.5 py-2.5 rounded-xl text-xs font-semibold'
                    } ${
                      isActive
                        ? 'bg-gradient-to-r from-indigo-600 to-violet-600 text-white shadow-button'
                        : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
                    }`}
                  >
                    <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-white' : 'text-slate-500'}`} />
                    {!collapsed && <span className="truncate">{tab.name}</span>}
                  </Link>

                  {/* Floating Tooltip Pill for Collapsed State */}
                  {collapsed && (
                    <div className="absolute left-14 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 translate-x-1 group-hover:translate-x-0 z-50">
                      <div className="bg-slate-900 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow-lifted whitespace-nowrap">
                        {tab.name}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}

            {/* Link to Projects Hub */}
            <div className={`pt-2 ${collapsed ? 'mt-2' : 'mt-3 border-t border-slate-100'}`}>
              <div className="relative group flex justify-center">
                <Link
                  to="/projects"
                  className={`flex items-center transition-all duration-150 ${
                    collapsed
                      ? 'w-10 h-10 justify-center rounded-xl'
                      : 'w-full space-x-3 px-3.5 py-2.5 rounded-xl text-xs font-medium'
                  } text-slate-600 hover:text-slate-900 hover:bg-slate-100`}
                >
                  <FolderKanban className="h-4 w-4 shrink-0 text-slate-500" />
                  {!collapsed && <span className="truncate">Projects Hub</span>}
                </Link>

                {collapsed && (
                  <div className="absolute left-14 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 translate-x-1 group-hover:translate-x-0 z-50">
                    <div className="bg-slate-900 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow-lifted whitespace-nowrap">
                      Projects Hub
                    </div>
                  </div>
                )}
              </div>
            </div>
          </nav>
        </div>

        {/* Sidebar Footer (User Info & Sign Out) */}
        <div className="border-t border-slate-100 pt-3">
          {collapsed ? (
            <div className="flex flex-col items-center space-y-2">
              <div className="relative group flex justify-center">
                <div className="h-8 w-8 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center font-bold text-xs shrink-0">
                  <UserIcon className="h-4 w-4" />
                </div>
                <div className="absolute left-14 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 translate-x-1 group-hover:translate-x-0 z-50">
                  <div className="bg-slate-900 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow-lifted whitespace-nowrap">
                    {user?.name || user?.email}
                  </div>
                </div>
              </div>

              <div className="relative group flex justify-center">
                <button
                  onClick={logout}
                  className="text-slate-400 hover:text-slate-800 p-1.5 rounded-lg hover:bg-slate-100 transition-all shrink-0"
                >
                  <LogOut className="h-4 w-4" />
                </button>
                <div className="absolute left-14 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 translate-x-1 group-hover:translate-x-0 z-50">
                  <div className="bg-slate-900 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow-lifted whitespace-nowrap">
                    Sign Out
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center space-x-2.5 overflow-hidden">
                <div className="h-8 w-8 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center font-bold text-xs shrink-0">
                  <UserIcon className="h-4 w-4" />
                </div>
                <div className="overflow-hidden text-xs">
                  <div className="font-semibold text-slate-900 truncate">{user?.name || 'User'}</div>
                  <div className="text-[10px] text-slate-500 truncate">{user?.email}</div>
                </div>
              </div>

              <button
                onClick={logout}
                className="text-slate-400 hover:text-slate-800 p-2 rounded-lg hover:bg-slate-100 transition-all shrink-0"
                title="Sign Out"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </div>
          )}
        </div>
      </aside>

      {/* TOP NAVBAR HEADER BAR */}
      <header
        className={`fixed top-0 right-0 h-16 bg-white/80 backdrop-blur-md border-b border-slate-200/80 px-6 flex items-center justify-between z-30 transition-all duration-300 ease-in-out ${
          collapsed ? 'left-16' : 'left-64'
        }`}
      >
        {/* Left Side: Expand Sidebar Button (in Header when Collapsed) + Breadcrumbs */}
        <div className="flex items-center space-x-3 text-xs text-slate-500 font-medium">
          {collapsed && (
            <button
              onClick={toggleSidebar}
              className="p-1.5 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-600 hover:text-slate-900 transition-all border border-slate-200 flex items-center space-x-1.5"
              title="Open sidebar"
            >
              <PanelLeftOpen className="h-4 w-4" />
              <span className="text-[11px] font-semibold text-slate-700 hidden sm:inline">Open sidebar</span>
            </button>
          )}

          <span>Pages</span>
          <span>/</span>
          <span className="font-bold text-slate-900">{getBreadcrumbTitle()}</span>
          {activeProject && (
            <>
              <span>/</span>
              <span className="bg-indigo-50 text-indigo-700 font-semibold px-2.5 py-0.5 rounded-full text-[11px] border border-indigo-100 truncate max-w-[160px] sm:max-w-xs">
                {activeProject.name}
              </span>
            </>
          )}
        </div>

        {/* Search Bar / Quick Action */}
        <div className="flex items-center space-x-4">
          <div className="relative hidden sm:block">
            <input
              type="text"
              placeholder="Search features..."
              className="w-48 lg:w-64 pl-9 pr-3 py-1.5 bg-slate-100 border border-slate-200 rounded-xl text-xs text-slate-800 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-slate-400" />
          </div>
        </div>
      </header>

      {/* MAIN CONTENT VIEWPORT */}
      <main
        className={`flex-1 pt-16 min-h-screen transition-all duration-300 ease-in-out z-10 ${
          collapsed ? 'pl-16' : 'pl-64'
        }`}
      >
        <div className="max-w-7xl mx-auto p-6 sm:p-8 space-y-6">
          {children || <Outlet />}
        </div>
      </main>
    </div>
  );
};
