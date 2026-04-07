import React, { useState } from 'react';
import { X, RefreshCw, CheckCircle, AlertTriangle, Zap } from 'lucide-react';
import { syncPriorities } from '../services/api';

// Priority colour map — matches common YouTrack priority palettes
const PRIORITY_COLORS = {
  P0: { bg: '#7B0000', text: '#fff' },
  P1: { bg: '#E20F86', text: '#fff' },
  P2: { bg: '#FF8C00', text: '#fff' },
  P3: { bg: '#2196F3', text: '#fff' },
  A1: { bg: '#7B0000', text: '#fff' },
  A2: { bg: '#E20F86', text: '#fff' },
  A3: { bg: '#FF8C00', text: '#fff' },
};

const PriorityBadge = ({ value }) => {
  if (!value) return <span className="text-gray-400 text-xs italic">None</span>;
  const col = PRIORITY_COLORS[value.toUpperCase()] || { bg: '#6B7280', text: '#fff' };
  return (
    <span
      className="inline-block px-2 py-0.5 rounded text-xs font-bold"
      style={{ backgroundColor: col.bg, color: col.text }}
    >
      {value}
    </span>
  );
};

const PrioritySyncModal = ({ mismatches, onClose, onSynced }) => {
  const [syncing, setSyncing] = useState(false);
  const [syncingId, setSyncingId] = useState(null);
  const [done, setDone] = useState(new Set());
  const [errors, setErrors] = useState({});

  const syncOne = async (issueId, priority) => {
    setSyncingId(issueId);
    try {
      await syncPriorities([{ youtrack_issue_id: issueId, priority }]);
      setDone(prev => new Set([...prev, issueId]));
      setErrors(prev => { const e = { ...prev }; delete e[issueId]; return e; });
      if (onSynced) onSynced(issueId);
    } catch (err) {
      setErrors(prev => ({ ...prev, [issueId]: err.message }));
    } finally {
      setSyncingId(null);
    }
  };

  const syncAll = async () => {
    setSyncing(true);
    const pending = mismatches.filter(m => {
      const id = m.youtrack_issue?.id;
      return id && !done.has(id);
    });
    try {
      const items = pending.map(m => ({
        youtrack_issue_id: m.youtrack_issue.id,
        priority: m.asana_priority,
      }));
      const result = await syncPriorities(items);
      const newDone = new Set(done);
      const newErrors = { ...errors };
      (result.data?.results || []).forEach(r => {
        if (r.status === 'ok') {
          newDone.add(r.youtrack_issue_id);
          delete newErrors[r.youtrack_issue_id];
          if (onSynced) onSynced(r.youtrack_issue_id);
        } else {
          newErrors[r.youtrack_issue_id] = r.error || 'Failed';
        }
      });
      setDone(newDone);
      setErrors(newErrors);
    } catch (err) {
      alert('Sync all failed: ' + err.message);
    } finally {
      setSyncing(false);
    }
  };

  const pending = mismatches.filter(m => !done.has(m.youtrack_issue?.id));

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 flex items-center">
              <Zap className="w-5 h-5 mr-2 text-yellow-500" />
              Priority Sync
              <span className="ml-2 bg-yellow-100 text-yellow-800 text-sm font-medium px-2 py-0.5 rounded-full">
                {pending.length} pending
              </span>
            </h2>
            <p className="text-sm text-gray-500 mt-0.5">
              Asana title priorities → YouTrack Priority field
            </p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Sync All bar */}
        {pending.length > 0 && (
          <div className="px-6 py-3 bg-yellow-50 border-b border-yellow-100 flex items-center justify-between">
            <span className="text-sm text-yellow-800">
              {pending.length} ticket{pending.length !== 1 ? 's' : ''} need priority update
            </span>
            <button
              onClick={syncAll}
              disabled={syncing}
              className="flex items-center bg-yellow-500 text-white px-4 py-1.5 rounded-lg text-sm font-medium hover:bg-yellow-600 disabled:opacity-50 transition-colors"
            >
              {syncing ? <RefreshCw className="w-3.5 h-3.5 mr-1.5 animate-spin" /> : <Zap className="w-3.5 h-3.5 mr-1.5" />}
              Sync All
            </button>
          </div>
        )}

        {/* Table */}
        <div className="flex-1 overflow-y-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 sticky top-0">
              <tr>
                <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Ticket</th>
                <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Asana</th>
                <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">YouTrack</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {mismatches.map((m) => {
                const ytId = m.youtrack_issue?.id;
                const isDone = done.has(ytId);
                const isSyncing = syncingId === ytId;
                const hasError = errors[ytId];
                const title = m.asana_task?.name || ytId;

                return (
                  <tr key={ytId} className={isDone ? 'bg-green-50' : 'hover:bg-gray-50'}>
                    <td className="px-6 py-3">
                      <div className="font-medium text-gray-900 truncate max-w-xs" title={title}>{title}</div>
                      <div className="text-xs text-gray-400">{ytId}</div>
                      {hasError && (
                        <div className="text-xs text-red-500 mt-0.5 flex items-center">
                          <AlertTriangle className="w-3 h-3 mr-1" />{hasError}
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <PriorityBadge value={m.asana_priority} />
                    </td>
                    <td className="px-4 py-3 text-center">
                      <PriorityBadge value={m.yt_priority} />
                    </td>
                    <td className="px-4 py-3 text-right">
                      {isDone ? (
                        <CheckCircle className="w-4 h-4 text-green-500 inline" />
                      ) : (
                        <button
                          onClick={() => syncOne(ytId, m.asana_priority)}
                          disabled={isSyncing || syncing}
                          className="flex items-center bg-blue-600 text-white px-3 py-1 rounded text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors ml-auto"
                        >
                          {isSyncing ? <RefreshCw className="w-3 h-3 mr-1 animate-spin" /> : <Zap className="w-3 h-3 mr-1" />}
                          Sync
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          {mismatches.length === 0 && (
            <div className="text-center py-12 text-gray-400">
              <CheckCircle className="w-8 h-8 mx-auto mb-2 text-green-400" />
              All priorities are in sync
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-3 border-t border-gray-200 flex justify-between items-center bg-gray-50 rounded-b-xl">
          <span className="text-xs text-gray-400">
            {done.size} of {mismatches.length} synced
          </span>
          <button onClick={onClose} className="text-sm text-gray-600 hover:text-gray-800 transition-colors">
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default PrioritySyncModal;
