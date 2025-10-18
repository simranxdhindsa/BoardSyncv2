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
  const [showRollbackDialog, setShowRollbackDialog] = useState(false);
  const [rollbackOperationId, setRollbackOperationId] = useState(null);

  useEffect(() => {
    loadSyncHistory();
  }, []);

  const loadSyncHistory = async () => {
    setLoading(true);
    try {
      const response = await getSyncHistory(15);
      // Handle nested response format: response.data.operations
      const operations = response.operations || response.data?.operations || response.data || [];

      // Filter out rollback operations - they shouldn't appear in main history
      const filteredOperations = operations.filter(op =>
        op.operation_type !== 'rollback' && op.operation_type !== 'Rollback'
      );

      setOperations(filteredOperations);
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

  const handleRollbackClick = (operationId) => {
    setRollbackOperationId(operationId);
    setShowRollbackDialog(true);
  };

  const handleRollbackConfirm = async () => {
    const operationId = rollbackOperationId;
    setShowRollbackDialog(false);
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
      setRollbackOperationId(null);
    }
  };

  const handleRollbackCancel = () => {
    setShowRollbackDialog(false);
    setRollbackOperationId(null);
  };

  const canRollback = (operation) => {
    // Don't allow rollback of rollback operations
    if (operation.operation_type === 'rollback' || operation.operation_type === 'Rollback') return false;

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
      'rollback': 'Rollback',
      'Ticket Creation': 'Ticket Creation',
      'Ticket Sync': 'Ticket Sync',
      'create': 'Ticket Creation',
      'sync': 'Ticket Sync'
    };
    return types[type] || type;
  };

  const getOperationSummary = (operation) => {
    const data = operation.operation_data || {};

    // For Ticket Creation
    if (operation.operation_type === 'Ticket Creation' || operation.operation_type === 'create') {
      const created = data.created || 0;
      const skipped = data.skipped || 0;
      const failed = data.failed || 0;
      const column = data.column || 'all columns';

      // If we have actual data, show it
      if (created > 0 || skipped > 0 || failed > 0) {
        return `${column} - Created: ${created}, Skipped: ${skipped}, Failed: ${failed}`;
      }

      // Old operation without detailed data
      return `${column} - Operation completed (legacy record)`;
    }

    // For Ticket Sync
    if (operation.operation_type === 'Ticket Sync' || operation.operation_type === 'sync') {
      const synced = data.synced || 0;
      const failed = data.failed || 0;
      const total = data.total || 0;
      const ticketCount = data.ticket_count || 0;
      const column = data.column || 'all columns';

      // If we have actual data, show it
      if (synced > 0 || failed > 0 || total > 0) {
        return `${column} - Synced: ${synced}/${total}, Failed: ${failed}`;
      }

      // Old operation with ticket_count only
      if (ticketCount > 0) {
        return `${column} - ${ticketCount} ticket${ticketCount > 1 ? 's' : ''} synced (legacy record)`;
      }

      // Very old operation without detailed data
      return `${column} - Operation completed (legacy record)`;
    }

    // Fallback
    return data.column || data.action || 'Operation completed';
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
                    {getOperationSummary(operation)}
                  </p>
                  <p className="text-xs text-gray-500 mt-1">
                    {formatDate(operation.created_at)}
                  </p>
                </div>
              </div>

              <div className="flex items-center space-x-3">
                {getStatusBadge(operation.status)}

                {canRollback(operation) && (
                  <button
                    onClick={() => handleRollbackClick(operation.id)}
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
                {operation.operation_data && (
                  <div>
                    <h4 className="text-sm font-semibold text-gray-700 mb-3">Operation Summary:</h4>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                      {operation.operation_data.created !== undefined && (
                        <div className="summary-stat-card">
                          <div className="summary-stat-number text-green-600">
                            {operation.operation_data.created}
                          </div>
                          <div className="summary-stat-label">Created</div>
                        </div>
                      )}
                      {operation.operation_data.skipped !== undefined && (
                        <div className="summary-stat-card">
                          <div className="summary-stat-number text-yellow-600">
                            {operation.operation_data.skipped}
                          </div>
                          <div className="summary-stat-label">Skipped</div>
                        </div>
                      )}
                      {operation.operation_data.synced !== undefined && (
                        <div className="summary-stat-card">
                          <div className="summary-stat-number text-blue-600">
                            {operation.operation_data.synced}
                          </div>
                          <div className="summary-stat-label">Synced</div>
                        </div>
                      )}
                      {operation.operation_data.failed !== undefined && operation.operation_data.failed > 0 && (
                        <div className="summary-stat-card">
                          <div className="summary-stat-number text-red-600">
                            {operation.operation_data.failed}
                          </div>
                          <div className="summary-stat-label">Failed</div>
                        </div>
                      )}
                      {operation.operation_data.total !== undefined && (
                        <div className="summary-stat-card">
                          <div className="summary-stat-number text-purple-600">
                            {operation.operation_data.total}
                          </div>
                          <div className="summary-stat-label">Total</div>
                        </div>
                      )}
                    </div>
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

      {/* Rollback Confirmation Dialog */}
      {showRollbackDialog && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
          onClick={handleRollbackCancel}
          style={{ backdropFilter: 'blur(8px)' }}
        >
          <div
            className="glass-panel max-w-md w-full mx-4"
            style={{
              borderRadius: '20px',
              overflow: 'hidden'
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div
              className="p-6"
              style={{
                background: 'rgba(255, 255, 255, 0.3)',
                backdropFilter: 'blur(20px) saturate(150%)',
                WebkitBackdropFilter: 'blur(20px) saturate(150%)',
                borderBottom: '1px solid rgba(255, 255, 255, 0.3)'
              }}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="settings-profile-avatar" style={{ width: '3rem', height: '3rem', background: 'linear-gradient(135deg, #ff6b6b 0%, #ee5a6f 100%)' }}>
                    <AlertCircle className="w-5 h-5 text-white" />
                  </div>
                  <h2 className="text-2xl font-bold text-gray-900">
                    Confirm Rollback
                  </h2>
                </div>
              </div>
            </div>

            {/* Modal Body */}
            <div className="p-6">
              <div className="mb-4">
                <p className="text-gray-700 mb-3">
                  Are you sure you want to rollback this operation?
                </p>
                <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                  <div className="flex items-start">
                    <AlertCircle className="w-5 h-5 text-yellow-600 mt-0.5 mr-3 flex-shrink-0" />
                    <div className="text-sm text-yellow-800">
                      <p className="font-semibold mb-1">This action will:</p>
                      <ul className="list-disc list-inside space-y-1">
                        <li>Undo all changes made during this sync</li>
                        <li>Restore tickets to their previous state</li>
                        <li>Delete any tickets that were created</li>
                        <li>Revert status and field updates</li>
                      </ul>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Modal Footer */}
            <div
              className="p-6 flex justify-end space-x-3"
              style={{
                background: 'rgba(255, 255, 255, 0.2)',
                backdropFilter: 'blur(16px) saturate(150%)',
                WebkitBackdropFilter: 'blur(16px) saturate(150%)',
                borderTop: '1px solid rgba(255, 255, 255, 0.3)'
              }}
            >
              <button
                onClick={handleRollbackCancel}
                className="settings-button-secondary"
              >
                Cancel
              </button>
              <button
                onClick={handleRollbackConfirm}
                className="settings-button"
                style={{
                  background: 'linear-gradient(135deg, #ff6b6b 0%, #ee5a6f 100%)',
                  border: '1px solid rgba(255, 255, 255, 0.3)'
                }}
              >
                <RotateCcw className="w-4 h-4 mr-2" />
                Yes, Rollback
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SyncHistory;
