// FILE: frontend/src/components/SyncHistory.js
// Sync History & Rollback Component

import React, { useState, useEffect } from 'react';
import {
  getSyncHistory,
  rollbackSync,
  getSnapshotSummary,
  getOperationAuditLogs
} from '../services/api';
import {
  RotateCcw,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  ChevronDown,
  ChevronUp,
  History
} from 'lucide-react';
import '../styles/sync-history-glass.css';

const SyncHistory = ({ onSuccess, onError }) => {
  const [operations, setOperations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [expandedOp, setExpandedOp] = useState(null);
  const [snapshotSummaries, setSnapshotSummaries] = useState({});
  const [rollingBack, setRollingBack] = useState(null);

  useEffect(() => {
    loadSyncHistory();
  }, []);

  const loadSyncHistory = async () => {
    setLoading(true);
    try {
      const response = await getSyncHistory(15);
      setOperations(response.operations || response.data || []);
    } catch (error) {
      onError?.('Failed to load sync history: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleExpandOperation = async (operationId) => {
    if (expandedOp === operationId) {
      setExpandedOp(null);
      return;
    }

    setExpandedOp(operationId);

    // Load snapshot summary if not already loaded
    if (!snapshotSummaries[operationId]) {
      try {
        const response = await getSnapshotSummary(operationId);
        setSnapshotSummaries(prev => ({
          ...prev,
          [operationId]: response.summary
        }));
      } catch (error) {
        // Snapshot might not exist for this operation
        console.log('No snapshot available for operation', operationId);
      }
    }
  };

  const handleRollback = async (operationId) => {
    if (!window.confirm('Are you sure you want to rollback this operation? This will undo all changes made during the sync.')) {
      return;
    }

    setRollingBack(operationId);
    try {
      const response = await rollbackSync(operationId);

      if (response.success) {
        onSuccess?.('Operation rolled back successfully!');
        // Reload history
        await loadSyncHistory();
      } else {
        onError?.('Rollback completed with errors: ' + (response.result?.errors?.join(', ') || 'Unknown error'));
      }
    } catch (error) {
      onError?.('Failed to rollback operation: ' + error.message);
    } finally {
      setRollingBack(null);
    }
  };

  const canRollback = (operation) => {
    if (operation.status !== 'completed') return false;
    if (operation.status === 'rolled_back') return false;

    // Check if within 24 hours
    const operationTime = new Date(operation.created_at);
    const now = new Date();
    const hoursDiff = (now - operationTime) / (1000 * 60 * 60);

    return hoursDiff < 24;
  };

  const getStatusBadge = (status) => {
    const badges = {
      'completed': { class: 'status-badge-completed', icon: CheckCircle, text: 'Completed' },
      'failed': { class: 'status-badge-failed', icon: XCircle, text: 'Failed' },
      'rolled_back': { class: 'status-badge-rolled-back', icon: RotateCcw, text: 'Rolled Back' },
      'in_progress': { class: 'status-badge-in-progress', icon: Clock, text: 'In Progress' },
      'pending': { class: 'status-badge-in-progress', icon: Clock, text: 'Pending' }
    };

    const badge = badges[status] || badges['pending'];
    const Icon = badge.icon;

    return (
      <span className={badge.class}>
        <Icon className="w-3 h-3 inline mr-1" />
        {badge.text}
      </span>
    );
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins} min${diffMins > 1 ? 's' : ''} ago`;
    if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
    if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;

    return date.toLocaleString();
  };

  const getOperationType = (type) => {
    const types = {
      'asana_to_youtrack': 'Asana → YouTrack',
      'youtrack_to_asana': 'YouTrack → Asana',
      'bidirectional': 'Bidirectional',
      'rollback': 'Rollback'
    };
    return types[type] || type;
  };

  if (loading) {
    return (
      <div className="sync-history-container">
        <div className="text-center py-12">
          <div className="sync-loading-spinner mx-auto mb-4"></div>
          <p className="text-gray-600">Loading sync history...</p>
        </div>
      </div>
    );
  }

  if (operations.length === 0) {
    return (
      <div className="sync-history-container">
        <div className="empty-state">
          <div className="empty-state-icon">
            <History className="w-16 h-16 mx-auto text-gray-400" />
          </div>
          <p className="empty-state-text">No sync operations yet</p>
          <p className="text-sm text-gray-500 mt-2">Your sync history will appear here</p>
        </div>
      </div>
    );
  }

  return (
    <div className="sync-history-container">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900 flex items-center">
          <History className="w-6 h-6 mr-2" />
          Sync History
        </h2>
        <button
          onClick={loadSyncHistory}
          className="view-details-button"
        >
          <RotateCcw className="w-4 h-4 inline mr-2" />
          Refresh
        </button>
      </div>

      <div className="space-y-4">
        {operations.map((operation) => (
          <div key={operation.id} className="operation-card">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center space-x-3">
                <Clock className="w-5 h-5 text-gray-600" />
                <div>
                  <h3 className="font-semibold text-gray-900">
                    {getOperationType(operation.operation_type)}
                  </h3>
                  <p className="text-sm text-gray-600">
                    {formatDate(operation.created_at)}
                  </p>
                </div>
              </div>

              <div className="flex items-center space-x-3">
                {getStatusBadge(operation.status)}

                {canRollback(operation) && (
                  <button
                    onClick={() => handleRollback(operation.id)}
                    disabled={rollingBack === operation.id}
                    className="rollback-button"
                  >
                    {rollingBack === operation.id ? (
                      <>
                        <div className="sync-loading-spinner inline-block mr-2"></div>
                        Rolling back...
                      </>
                    ) : (
                      <>
                        <RotateCcw className="w-4 h-4 inline mr-2" />
                        Rollback
                      </>
                    )}
                  </button>
                )}

                <button
                  onClick={() => handleExpandOperation(operation.id)}
                  className="view-details-button"
                >
                  {expandedOp === operation.id ? (
                    <ChevronUp className="w-4 h-4" />
                  ) : (
                    <ChevronDown className="w-4 h-4" />
                  )}
                </button>
              </div>
            </div>

            {operation.error_message && (
              <div className="mt-3 p-3 bg-red-50 border border-red-200 rounded-lg">
                <div className="flex items-start">
                  <AlertCircle className="w-4 h-4 text-red-600 mt-0.5 mr-2" />
                  <p className="text-sm text-red-800">{operation.error_message}</p>
                </div>
              </div>
            )}

            {/* Expanded Details */}
            {expandedOp === operation.id && (
              <div className="mt-4 pt-4 border-t border-white border-opacity-30">
                {snapshotSummaries[operation.id] ? (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div className="summary-stat-card">
                      <div className="summary-stat-number text-green-600">
                        {snapshotSummaries[operation.id].tickets_created || 0}
                      </div>
                      <div className="summary-stat-label">Created</div>
                    </div>
                    <div className="summary-stat-card">
                      <div className="summary-stat-number text-blue-600">
                        {snapshotSummaries[operation.id].tickets_updated || 0}
                      </div>
                      <div className="summary-stat-label">Updated</div>
                    </div>
                    <div className="summary-stat-card">
                      <div className="summary-stat-number text-purple-600">
                        {snapshotSummaries[operation.id].mappings_changed || 0}
                      </div>
                      <div className="summary-stat-label">Mappings</div>
                    </div>
                    <div className="summary-stat-card">
                      <div className="summary-stat-number text-orange-600">
                        {snapshotSummaries[operation.id].total_changes || 0}
                      </div>
                      <div className="summary-stat-label">Total Changes</div>
                    </div>
                  </div>
                ) : (
                  <p className="text-sm text-gray-600">No detailed information available for this operation.</p>
                )}

                {operation.operation_data && (
                  <div className="mt-4">
                    <h4 className="text-sm font-semibold text-gray-700 mb-2">Operation Details:</h4>
                    <pre className="text-xs text-gray-600 bg-white bg-opacity-30 p-3 rounded-lg overflow-auto">
                      {JSON.stringify(operation.operation_data, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
      </div>

      {operations.length >= 15 && (
        <div className="mt-6 text-center">
          <p className="text-sm text-gray-600">
            Showing last 15 operations. Older operations are automatically removed.
          </p>
        </div>
      )}
    </div>
  );
};

export default SyncHistory;
