import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { Layers, ArrowRight, Lock, Mail, AlertCircle, CheckCircle2 } from 'lucide-react';

export const Login: React.FC = () => {
  const [email, setEmail] = useState('admin@admin.com');
  const [password, setPassword] = useState('admin123');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(email, password);
      navigate('/projects');
    } catch (err: any) {
      setError(err.message || 'Invalid credentials');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col justify-center py-12 sm:px-6 lg:px-8 relative overflow-hidden font-sans">
      {/* Background Decorative Gradients */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-indigo-600/15 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-10 right-10 w-[400px] h-[400px] bg-violet-600/10 rounded-full blur-3xl pointer-events-none" />

      <div className="sm:mx-auto sm:w-full sm:max-max-w-md z-10 text-center mb-8">
        <div className="inline-flex items-center justify-center space-x-3 mb-4">
          <div className="h-12 w-12 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-lg shadow-indigo-500/30">
            <Layers className="h-6 w-6 text-white" />
          </div>
          <span className="text-3xl font-bold text-white tracking-tight">konteks</span>
        </div>
        <h2 className="text-2xl font-bold text-slate-100 tracking-tight">Sign in to your account</h2>
        <p className="mt-2 text-sm text-slate-400">
          Manage multi-tenant context packages, memory vectors, and API keys.
        </p>
      </div>

      <div className="sm:mx-auto sm:w-full sm:max-w-md z-10">
        <div className="bg-slate-900/80 backdrop-blur-xl py-8 px-6 shadow-2xl shadow-black/50 border border-slate-800/80 rounded-2xl sm:px-10">
          
          {/* Quick Demo Pre-seed Badge */}
          <div className="mb-6 bg-indigo-950/60 border border-indigo-500/30 rounded-xl p-3.5 flex items-start space-x-3 text-xs text-indigo-200">
            <CheckCircle2 className="h-4 w-4 text-indigo-400 shrink-0 mt-0.5" />
            <div>
              <span className="font-semibold text-indigo-300">Quick Test Credentials Pre-filled:</span>
              <p className="text-indigo-300/80 mt-0.5">Email: <code className="bg-slate-950/60 px-1.5 py-0.5 rounded text-indigo-200">admin@admin.com</code> | Password: <code className="bg-slate-950/60 px-1.5 py-0.5 rounded text-indigo-200">admin123</code></p>
            </div>
          </div>

          {error && (
            <div className="mb-6 bg-red-950/80 border border-red-500/50 rounded-xl p-3.5 flex items-center space-x-3 text-xs text-red-300">
              <AlertCircle className="h-4 w-4 text-red-400 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <form className="space-y-5" onSubmit={handleSubmit}>
            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                Email address
              </label>
              <div className="relative rounded-xl shadow-sm">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                  <Mail className="h-4 w-4" />
                </div>
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="block w-full pl-10 pr-3.5 py-2.5 bg-slate-950/70 border border-slate-800 rounded-xl text-slate-100 placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                  placeholder="admin@admin.com"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                Password
              </label>
              <div className="relative rounded-xl shadow-sm">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                  <Lock className="h-4 w-4" />
                </div>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="block w-full pl-10 pr-3.5 py-2.5 bg-slate-950/70 border border-slate-800 rounded-xl text-slate-100 placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                  placeholder="••••••••"
                />
              </div>
            </div>

            <div>
              <button
                type="submit"
                disabled={loading}
                className="w-full flex justify-center items-center py-2.5 px-4 border border-transparent rounded-xl text-sm font-semibold text-white bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 focus:ring-offset-slate-900 transition-all shadow-lg shadow-indigo-600/25 disabled:opacity-50"
              >
                {loading ? (
                  <span className="inline-flex items-center space-x-2">
                    <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    <span>Signing in...</span>
                  </span>
                ) : (
                  <span className="inline-flex items-center space-x-2">
                    <span>Sign In</span>
                    <ArrowRight className="h-4 w-4" />
                  </span>
                )}
              </button>
            </div>
          </form>

          <div className="mt-6 text-center text-xs text-slate-400">
            Don't have an account?{' '}
            <Link to="/register" className="font-semibold text-indigo-400 hover:text-indigo-300">
              Create an account
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
};
