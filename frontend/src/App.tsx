import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { Login } from './pages/Login';
import { Register } from './pages/Register';
import { Projects } from './pages/Projects';
import { Layout } from './components/Layout';
import { OverviewPage } from './pages/Overview';
import { LogsPage } from './pages/Logs';
import { RoomDetailPage } from './pages/RoomDetail';
import { EvaluationDetailPage } from './pages/EvaluationDetail';
import { ApiKeys } from './pages/ApiKeys';
import { Settings } from './pages/Settings';

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center text-slate-600">
        <div className="flex items-center space-x-3 bg-white p-4 rounded-xl border border-slate-200 shadow-soft">
          <div className="w-5 h-5 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin" />
          <span className="text-xs font-semibold text-slate-800">Loading authentication...</span>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

const DashboardRouteGuard: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { activeProject, projects, isLoading } = useAuth();

  if (isLoading) return null;

  // If user has no active project, force them to /projects hub
  if (!activeProject && projects.length === 0) {
    return <Navigate to="/projects" replace />;
  }

  return <>{children}</>;
};

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          {/* Public Auth Routes */}
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />

          {/* Intermediary Projects Hub Route */}
          <Route
            path="/projects"
            element={
              <ProtectedRoute>
                <Projects />
              </ProtectedRoute>
            }
          />

          {/* Protected Scoped Dashboard Routes */}
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <DashboardRouteGuard>
                  <Layout />
                </DashboardRouteGuard>
              </ProtectedRoute>
            }
          >
            <Route index element={<OverviewPage />} />
            <Route path="logs" element={<LogsPage />} />
            <Route path="logs/:roomId" element={<RoomDetailPage />} />
            <Route path="evaluations/:evalId" element={<EvaluationDetailPage />} />

            {/* Legacy Backward Compatibility Redirects */}
            <Route path="rooms" element={<Navigate to="/dashboard/logs" replace />} />
            <Route path="rooms/:roomId" element={<RoomDetailPage />} />

            <Route path="api-keys" element={<ApiKeys />} />
            <Route path="settings" element={<Settings />} />
          </Route>

          {/* Fallback Redirect */}
          <Route path="*" element={<Navigate to="/projects" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
