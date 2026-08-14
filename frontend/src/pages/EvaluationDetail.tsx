import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  fetchEvaluationDetail,
  type ContextEvaluation,
} from '../api/dashboard';
import {
  ArrowLeft,
  Activity,
  CheckCircle2,
  Cpu,
  Clock,
  ExternalLink,
  MessageSquare,
  FileCode,
  Sparkles,
} from 'lucide-react';

export const EvaluationDetailPage: React.FC = () => {
  const { evalId } = useParams<{ evalId: string }>();
  const navigate = useNavigate();

  const [evaluation, setEvaluation] = useState<ContextEvaluation | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!evalId) return;
    setLoading(true);
    setError(null);
    fetchEvaluationDetail(evalId)
      .then((data) => {
        setEvaluation(data);
      })
      .catch((err: any) => {
        setError(err.message || 'Failed to load evaluation details');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [evalId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="flex items-center space-x-3 bg-white p-4 rounded-xl border border-slate-200 shadow-soft">
          <div className="w-5 h-5 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin" />
          <span className="text-xs font-semibold text-slate-800">Loading evaluation inspection...</span>
        </div>
      </div>
    );
  }

  if (error || !evaluation) {
    return (
      <div className="bg-white rounded-xl border border-slate-200 shadow-soft p-8 text-center max-w-xl mx-auto">
        <div className="w-12 h-12 bg-rose-50 text-rose-600 rounded-full flex items-center justify-center mx-auto mb-3">
          <Activity className="h-6 w-6" />
        </div>
        <h3 className="text-lg font-bold text-slate-900 mb-1">Evaluation Not Found</h3>
        <p className="text-xs text-slate-500 mb-6">{error || 'The requested evaluation record does not exist.'}</p>
        <button
          onClick={() => navigate('/dashboard/logs')}
          className="inline-flex items-center space-x-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-xs rounded-lg transition-all"
        >
          <ArrowLeft className="h-4 w-4" />
          <span>Back to Logs</span>
        </button>
      </div>
    );
  }

  const formatScore = (val?: number) => {
    if (val === undefined || val === null) return '0.0%';
    return (val * 100).toFixed(1) + '%';
  };

  return (
    <div className="space-y-6 font-sans">
      {/* Header Bar & Back Button */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 bg-white border border-slate-200/80 rounded-xl p-6 shadow-soft">
        <div className="flex items-center space-x-4">
          <button
            onClick={() => navigate('/dashboard/logs')}
            className="p-2 rounded-lg border border-slate-200 text-slate-600 hover:text-slate-900 hover:bg-slate-50 transition-all shrink-0"
            title="Back to Logs"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>

          <div>
            <div className="flex items-center space-x-2.5">
              <h1 className="text-xl font-bold text-slate-900 tracking-tight">Context Evaluation Inspection</h1>
              <span className="bg-indigo-50 text-indigo-700 text-[11px] font-mono font-semibold px-2.5 py-0.5 rounded-full border border-indigo-100">
                ID: {evaluation.id.slice(0, 8)}...
              </span>
            </div>
            <p className="text-xs text-slate-500 mt-0.5 flex items-center space-x-3">
              <span className="flex items-center space-x-1">
                <Clock className="h-3.5 w-3.5 text-slate-400" />
                <span>{new Date(evaluation.created_at).toLocaleString()}</span>
              </span>
              <span>•</span>
              <span className="flex items-center space-x-1">
                <MessageSquare className="h-3.5 w-3.5 text-slate-400" />
                <span>Room: <code className="font-mono text-slate-700">{evaluation.room_id.slice(0, 8)}...</code></span>
              </span>
            </p>
          </div>
        </div>

        <Link
          to={`/dashboard/logs/${evaluation.room_id}`}
          className="inline-flex items-center space-x-2 px-4 py-2 rounded-lg text-xs font-semibold text-indigo-700 bg-indigo-50 border border-indigo-100 hover:bg-indigo-100 transition-all shrink-0"
        >
          <span>Inspect Room Logs</span>
          <ExternalLink className="h-3.5 w-3.5" />
        </Link>
      </div>

      {/* Top 3 Quality Metric Score Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white border border-slate-200/80 rounded-xl p-5 shadow-soft">
          <div className="flex items-center space-x-2 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
            <Activity className="h-4 w-4 text-emerald-600" />
            <span>Relevance Score</span>
          </div>
          <div className="text-3xl font-extrabold text-slate-900 tracking-tight mb-1">
            {formatScore(evaluation.relevance_score)}
          </div>
          <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
            <div
              className="bg-emerald-500 h-full rounded-full transition-all"
              style={{ width: `${(evaluation.relevance_score || 0) * 100}%` }}
            />
          </div>
        </div>

        <div className="bg-white border border-slate-200/80 rounded-xl p-5 shadow-soft">
          <div className="flex items-center space-x-2 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
            <CheckCircle2 className="h-4 w-4 text-indigo-600" />
            <span>Completeness Score</span>
          </div>
          <div className="text-3xl font-extrabold text-slate-900 tracking-tight mb-1">
            {formatScore(evaluation.completeness_score)}
          </div>
          <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
            <div
              className="bg-indigo-600 h-full rounded-full transition-all"
              style={{ width: `${(evaluation.completeness_score || 0) * 100}%` }}
            />
          </div>
        </div>

        <div className="bg-white border border-slate-200/80 rounded-xl p-5 shadow-soft">
          <div className="flex items-center space-x-2 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
            <Cpu className="h-4 w-4 text-violet-600" />
            <span>Token Efficiency</span>
          </div>
          <div className="text-3xl font-extrabold text-slate-900 tracking-tight mb-1">
            {formatScore(evaluation.token_efficiency_score)}
          </div>
          <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
            <div
              className="bg-violet-600 h-full rounded-full transition-all"
              style={{ width: `${(evaluation.token_efficiency_score || 0) * 100}%` }}
            />
          </div>
        </div>
      </div>

      {/* Query Card */}
      <div className="bg-white border border-slate-200/80 rounded-xl p-6 shadow-soft space-y-2">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center space-x-2">
          <span>Evaluated User Query</span>
        </h3>
        <div className="p-3.5 bg-slate-50 border border-slate-200 rounded-lg text-sm text-slate-900 font-mono font-medium">
          {evaluation.query || <span className="italic text-slate-400">(Empty query / Passive state compilation)</span>}
        </div>
      </div>

      {/* AI Model Reasoning Card */}
      <div className="bg-white border border-slate-200/80 rounded-xl p-6 shadow-soft space-y-2">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center space-x-2">
          <Sparkles className="h-4 w-4 text-indigo-600" />
          <span>Evaluation Reasoning Breakdown</span>
        </h3>
        <blockquote className="border-l-4 border-indigo-600 pl-4 py-3 bg-slate-50/80 rounded-r-lg text-slate-800 text-sm italic leading-relaxed">
          "{evaluation.reasoning || 'No evaluation reasoning provided.'}"
        </blockquote>
      </div>

      {/* Measured Context Snapshot Card */}
      <div className="bg-white border border-slate-200/80 rounded-xl p-6 shadow-soft space-y-3">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center space-x-2">
          <FileCode className="h-4 w-4 text-indigo-600" />
          <span>Measured Context Package Snapshot</span>
        </h3>
        <div className="bg-slate-900 text-slate-100 p-4 rounded-xl font-mono text-xs overflow-x-auto max-h-96">
          <pre>{JSON.stringify(evaluation.context_snapshot || {}, null, 2)}</pre>
        </div>
      </div>
    </div>
  );
};
