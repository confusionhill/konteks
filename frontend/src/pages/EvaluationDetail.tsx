import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { StatCard } from '../components/StatCard';
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

  const [decryptedQuery, setDecryptedQuery] = useState<string | null>(null);
  const [decryptedReasoning, setDecryptedReasoning] = useState<string | null>(null);
  const [decryptedSnapshot, setDecryptedSnapshot] = useState<any | null>(null);
  const [isDecrypting, setIsDecrypting] = useState(false);
  const [decryptionError, setDecryptionError] = useState<string | null>(null);

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

  const handleReveal = async () => {
    if (!evaluation) return;
    const storedKey = localStorage.getItem('konteks_master_key');
    let keyToUse = storedKey;

    if (!keyToUse) {
      keyToUse = prompt('Enter your Project Master Secret Key to decrypt evaluation:');
      if (!keyToUse) return;
      localStorage.setItem('konteks_master_key', keyToUse);
    }

    try {
      setIsDecrypting(true);
      setDecryptionError(null);

      let meta: any = {};
      if (evaluation.encryption_metadata) {
        if (typeof evaluation.encryption_metadata === 'string') {
          try {
            meta = JSON.parse(evaluation.encryption_metadata);
          } catch {
            meta = {};
          }
        } else {
          meta = evaluation.encryption_metadata;
        }
      }
      const nonce = meta.nonce || '';

      // Import decryptAESGCM helper
      const { decryptAESGCM } = await import('../utils/crypto');

      // 1. Decrypt Query if encrypted
      let decQ = evaluation.query;
      if (evaluation.is_encrypted && evaluation.query) {
        decQ = await decryptAESGCM(evaluation.query, nonce, keyToUse);
      }

      // 2. Decrypt Reasoning if encrypted
      let decR = evaluation.reasoning;
      if (evaluation.is_encrypted && evaluation.reasoning) {
        decR = await decryptAESGCM(evaluation.reasoning, nonce, keyToUse);
      }

      // 3. Decrypt Snapshot if encrypted string
      let decS = evaluation.context_snapshot;
      if (evaluation.is_encrypted && typeof evaluation.context_snapshot === 'string') {
        const rawDec = await decryptAESGCM(evaluation.context_snapshot, nonce, keyToUse);
        try {
          decS = JSON.parse(rawDec);
        } catch {
          decS = rawDec;
        }
      }

      setDecryptedQuery(decQ);
      setDecryptedReasoning(decR);
      setDecryptedSnapshot(decS);
    } catch (err: any) {
      console.error('Evaluation decryption error:', err);
      setDecryptionError('Failed to decrypt evaluation with provided secret key');
      localStorage.removeItem('konteks_master_key');
    } finally {
      setIsDecrypting(false);
    }
  };

  const isEncrypted = evaluation.is_encrypted;
  const showDecrypted = !isEncrypted || decryptedQuery !== null;

  return (
    <div className="space-y-6 font-sans">
      {/* Back Link */}
      <div>
        <Link
          to="/dashboard/logs"
          className="inline-flex items-center space-x-1.5 text-xs font-semibold text-indigo-600 hover:text-indigo-700 transition-colors group"
        >
          <ArrowLeft className="h-4 w-4 group-hover:-translate-x-1 transition-transform" />
          <span>Back to Logs</span>
        </Link>
      </div>

      {/* Header Info & Top Metrics Card */}
      <div className="bg-white rounded-2xl p-6 border border-slate-200/80 shadow-soft space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center space-x-2">
              <span className="text-xs font-mono font-bold uppercase tracking-wider text-indigo-600 bg-indigo-50 px-2.5 py-1 rounded-full border border-indigo-100">
                Evaluation Inspection
              </span>
              {isEncrypted && (
                <span className="bg-indigo-600 text-white text-[10px] font-mono font-bold px-2.5 py-1 rounded-full shadow-sm flex items-center space-x-1">
                  <span>🔒 ENCRYPTED EVALUATION</span>
                </span>
              )}
            </div>
            <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight font-mono mt-2">
              {evaluation.id}
            </h1>
          </div>

          <div className="flex flex-wrap items-center gap-3 text-xs font-mono text-slate-500 border-t sm:border-t-0 pt-3 sm:pt-0 border-slate-100">
            <div className="flex items-center space-x-1.5 bg-slate-50 px-3 py-1.5 rounded-lg border border-slate-100">
              <Clock className="h-3.5 w-3.5 text-slate-400" />
              <span>Created: {new Date(evaluation.created_at).toLocaleDateString()}</span>
            </div>

            <div className="flex items-center space-x-1.5 bg-slate-50 px-3 py-1.5 rounded-lg border border-slate-100">
              <MessageSquare className="h-3.5 w-3.5 text-slate-400" />
              <span>
                Room:{' '}
                <Link
                  to={`/dashboard/logs/${evaluation.room_id}`}
                  className="font-mono text-indigo-600 hover:underline"
                >
                  {evaluation.room_id.slice(0, 8)}...
                </Link>
              </span>
            </div>

            <Link
              to={`/dashboard/logs/${evaluation.room_id}`}
              className="inline-flex items-center space-x-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold text-indigo-700 bg-indigo-50 border border-indigo-100 hover:bg-indigo-100 transition-all shrink-0 font-sans"
            >
              <span>Inspect Room Logs</span>
              <ExternalLink className="h-3.5 w-3.5" />
            </Link>
          </div>
        </div>

        {/* Quality Metric Score Cards (Integrated) */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-4 border-t border-slate-100">
          <StatCard
            label="Relevance Score"
            value={formatScore(evaluation.relevance_score)}
            icon={<Activity className="h-5 w-5 text-emerald-600" />}
            progress={evaluation.relevance_score}
            progressColor="bg-emerald-500"
          />
          <StatCard
            label="Completeness Score"
            value={formatScore(evaluation.completeness_score)}
            icon={<CheckCircle2 className="h-5 w-5 text-indigo-600" />}
            progress={evaluation.completeness_score}
            progressColor="bg-indigo-600"
          />
          <StatCard
            label="Token Efficiency"
            value={formatScore(evaluation.token_efficiency_score)}
            icon={<Cpu className="h-5 w-5 text-violet-600" />}
            progress={evaluation.token_efficiency_score}
            progressColor="bg-violet-600"
          />
        </div>
      </div>

      {/* Privacy Banner if Encrypted and Not Decrypted */}
      {isEncrypted && !showDecrypted && (
        <div className="bg-indigo-50 border border-indigo-200 rounded-xl p-5 shadow-soft flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div className="flex items-center space-x-3">
            <div className="p-2.5 bg-indigo-600 text-white rounded-lg font-mono font-bold text-sm shrink-0">
              🔒
            </div>
            <div>
              <h4 className="text-sm font-bold text-slate-900">Zero-Knowledge Encrypted Evaluation Record</h4>
              <p className="text-xs text-slate-600 mt-0.5">
                The query, reasoning, and context snapshot are end-to-end encrypted. Enter your master secret key to decrypt in browser RAM.
              </p>
              {decryptionError && <p className="text-xs font-semibold text-rose-600 mt-1">{decryptionError}</p>}
            </div>
          </div>

          <button
            onClick={handleReveal}
            disabled={isDecrypting}
            className="inline-flex items-center justify-center space-x-2 px-5 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-bold rounded-lg shadow-button transition-all shrink-0 disabled:opacity-50"
          >
            {isDecrypting ? 'Decrypting...' : 'Reveal Evaluation Payload'}
          </button>
        </div>
      )}

      {/* Query Card */}
      <div className="bg-white border border-slate-200/80 rounded-xl p-6 shadow-soft space-y-2">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center space-x-2">
          <span>Evaluated User Query</span>
        </h3>
        <div className="p-3.5 bg-slate-50 border border-slate-200 rounded-lg text-sm text-slate-900 font-mono font-medium">
          {showDecrypted ? (
            decryptedQuery || evaluation.query || <span className="italic text-slate-400">(Empty query / Passive state compilation)</span>
          ) : (
            <span className="text-amber-700 font-semibold italic flex items-center space-x-1.5">
              <span>🔒 Encrypted Evaluation Query Payload</span>
            </span>
          )}
        </div>
      </div>

      {/* AI Model Reasoning Card */}
      <div className="bg-white border border-slate-200/80 rounded-xl p-6 shadow-soft space-y-2">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center space-x-2">
          <Sparkles className="h-4 w-4 text-indigo-600" />
          <span>Evaluation Reasoning Breakdown</span>
        </h3>
        <blockquote className="border-l-4 border-indigo-600 pl-4 py-3 bg-slate-50/80 rounded-r-lg text-slate-800 text-sm italic leading-relaxed">
          {showDecrypted ? (
            `"${decryptedReasoning || evaluation.reasoning || 'No evaluation reasoning provided.'}"`
          ) : (
            <span className="text-amber-700 font-semibold italic flex items-center space-x-1.5">
              <span>🔒 Encrypted Evaluation Reasoning Payload</span>
            </span>
          )}
        </blockquote>
      </div>

      {/* Measured Context Snapshot Card */}
      <div className="bg-white border border-slate-200/80 rounded-xl p-6 shadow-soft space-y-3">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center space-x-2">
          <FileCode className="h-4 w-4 text-indigo-600" />
          <span>Measured Context Package Snapshot</span>
        </h3>
        <div className="bg-slate-900 text-slate-100 p-4 rounded-xl font-mono text-xs overflow-x-auto max-h-96">
          <pre>
            {showDecrypted
              ? JSON.stringify(decryptedSnapshot || evaluation.context_snapshot || {}, null, 2)
              : '// 🔒 Context Snapshot is Encrypted'}
          </pre>
        </div>
      </div>
    </div>
  );
};

