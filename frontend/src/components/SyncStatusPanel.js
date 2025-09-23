import React, { useState, useEffect } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import { getSyncHistory, rollbackSync } from '../services/api';
import { 
  Activity, 
  CheckCircle, 
  AlertTriangle, 
  RefreshCw, 
  Clock, 
  Undo,
  Wifi,
  WifiOff,
  X,
  History,
  Zap
} from 'lucide-react';

const SyncStatusPanel = ({ isVisible, onClose }) => {
  const { 
    connected, 
    connecting, 
    messages, 
    getMessagesByType, 
    clearMessagesByType 
  } = useWebSocket();

  const [syncHistory, setSyncHistory] = useState([]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [activeOperations, setActiveOperations] = useState(new Map());
  const [showHistory, setShowHistory] = useState(false);

  // Load sync history on mount
  useEffect(() => {
    if (isVisible) {
      loadSyncHistory();
    }
  }, [isVisible]);

  // Update active operations from WebSocket messages
  useEffect(() => {
    const syncMessages = getMessagesByType('sync_start')
      .concat(getMessagesByType('sync_progress'))
      .concat(getMessagesByType('sync_complete'))
      .concat(getMessagesByType('sync_error'));

    const operations = new Map();
    
    syncMessages.forEach(msg => {
      if (msg.data?.operation_id) {
        const operationId = msg.data.operation_id;
        const existing = operations.get(operationId) || { id: operationId };
        
        switch (msg.type) {
          case 'sync_start':
            operations.set(operationId, {
              ...existing,
              status: 'in_progress',
              type: msg.data.type,
              startedAt: msg.timestamp,
              progress: 0
            });
            break;
          case 'sync_progress':
            operations.set(operationId, {
              ...existing,
              progress: msg.data.progress,
              message: msg.data.message
            });
            break;
          case 'sync_complete':
            operations.set(operationId, {
              ...existing,
              status: 'completed',
              completedAt: msg.timestamp,
              progress: 100,
              result: msg.data.result
            });
            break;
          case 'sync_error':
            operations.set(operationId, {
              ...existing,
              status: 'failed',
              completedAt: msg.timestamp,
              error: msg.data.error
            });
            break;
        }
      }
    });

    setActiveOperations(operations);
  }, [getMessagesByType]);

  const loadSyncHistory = async () => {
    setLoadingHistory(true);
    try {
      const response = await getSyncHistory(10);
      setSyncHistory(response.data || response);
    } catch (err) {
      console.error('Failed to load sync history:', err);
    } finally {
      setLoadingHistory(false);
    }
  };

  const handleRollback = async (operationId) => {
    if (!window.confirm('Are you sure you want to rollback this sync operation? This action cannot be undone.')) {
      return;
    }

    try {
      await rollbackSync(operationId);
      loadSyncHistory(); // Refresh history
    } catch (err) {
      alert('Rollback failed: ' + err.message);
    }
  };

  const clearNotifications = (type) => {
    clearMessagesByType(type);
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-green-600" />;
      case 'failed':
        return <AlertTriangle className="w-4 h-4 text-red-600" />;
      case 'in_progress':
        return <RefreshCw className="w-4 h-4 text-blue-600 animate-spin" />;
      default:
        return <Clock className="w-4 h-4 text-gray-600" />;
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed':
        return 'bg-green-100 border-green-200 text-green-800';
      case 'failed':
        return 'bg-red-100 border-red-200 text-red-800';
      case 'in_progress':
        return 'bg-blue-100 border-blue-200 text-blue-800';
      default:
        return 'bg-gray-100 border-gray-200 text-gray-800';
    }
  };

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return '';
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  };

  if (!isVisible) return null;

  return (
    <div className="fixed inset-y-0 right-0 w-96 bg-white shadow-xl border-l border-gray-200 z-50 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200">
        <div className="flex items-center space-x-3">
          <Activity className="w-5 h-5 text-blue-600" />
          <h2 className="text-lg font-semibold text-gray-900">Sync Status</h2>
          <div className="flex items-center space-x-1">
            {connected ? (
              <Wifi className="w-4 h-4 text-green-600" title="Connected" />
            ) : (
              <WifiOff className="w-4 h-4 text-red-600" title="Disconnected" />
            )}
            {connecting && (
              <RefreshCw className="w-3 h-3 text-blue-600 animate-spin" title="Connecting..." />
            )}
          </div>
        </div>
        <button 
          onClick={onClose}
          className="p-1 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <X className="w-5 h-5 text-gray-500" />
        </button>
      </div>

      {/* Tab Navigation */}
      <div className="flex border-b border-gray-200">
        <button
          onClick={() => setShowHistory(false)}
          className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
            !showHistory 
              ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50'
              : 'text-gray-700 hover:text-blue-600'
          }`}
        >
          <Zap className="w-4 h-4 inline-block mr-1" />
          Live Status
        </button>
        <button
          onClick={() => setShowHistory(true)}
          className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
            showHistory 
              ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50'
              : 'text-gray-700 hover:text-blue-600'
          }`}
        >
          <History className="w-4 h-4 inline-block mr-1" />
          History
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {!showHistory ? (
          // Live Status Tab
          <div className="p-4 space-y-4">
            {/* Connection Status */}
            <div className={`p-3 rounded-lg border ${
              connected 
                ? 'bg-green-50 border-green-200'
                : 'bg-red-50 border-red-200'
            }`}>
              <div className="flex items-center space-x-2">
                {connected ? (
                  <CheckCircle className="w-4 h-4 text-green-600" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-red-600" />
                )}
                <span className={`text-sm font-medium ${
                  connected ? 'text-green-800' : 'text-red-800'
                }`}>
                  {connected ? 'Real-time updates active' : 'Disconnected from updates'}
                </span>
              </div>
            </div>

            {/* Active Operations */}
            {activeOperations.size > 0 ? (
              <div className="space-y-3">
                <h3 className="text-sm font-medium text-gray-900">Active Operations</h3>
                {Array.from(activeOperations.values()).map(operation => (
                  <div 
                    key={operation.id} 
                    className={`p-3 rounded-lg border ${getStatusColor(operation.status)}`}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center space-x-2">
                        {getStatusIcon(operation.status)}
                        <span className="text-sm font-medium">
                          {operation.type || 'Sync Operation'} #{operation.id}
                        </span>
                      </div>
                      {operation.startedAt && (
                        <span className="text-xs opacity-75">
                          {formatTimestamp(operation.startedAt)}
                        </span>
                      )}
                    </div>
                    
                    {operation.progress !== undefined && operation.status === 'in_progress' && (
                      <div className="mb-2">
                        <div className="flex justify-between text-xs mb-1">
                          <span>{operation.message || 'Processing...'}</span>
                          <span>{operation.progress}%</span>
                        </div>
                        <div className="w-full bg-white bg-opacity-50 rounded-full h-2">
                          <div 
                            className="bg-current h-2 rounded-full transition-all duration-300"
                            style={{ width: `${operation.progress}%` }}
                          />
                        </div>
                      </div>
                    )}
                    
                    {operation.error && (
                      <p className="text-xs text-red-700 mt-1">{operation.error}</p>
                    )}
                    
                    {operation.result && (
                      <div className="text-xs mt-1">
                        <p>Synced: {operation.result.synced_items || 0} items</p>
                        {operation.result.created_items > 0 && (
                          <p>Created: {operation.result.created_items}</p>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <Activity className="w-12 h-12 text-gray-300 mx-auto mb-3" />
                <p className="text-gray-500 text-sm">No active sync operations</p>
                <p className="text-gray-400 text-xs mt-1">
                  Start a sync to see real-time progress here
                </p>
              </div>
            )}
          </div>
        ) : (
          // History Tab
          <div className="p-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-medium text-gray-900">Recent Operations</h3>
              <button 
                onClick={loadSyncHistory}
                disabled={loadingHistory}
                className="p-1 hover:bg-gray-100 rounded transition-colors"
              >
                <RefreshCw className={`w-4 h-4 text-gray-500 ${loadingHistory ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {syncHistory.length > 0 ? (
              <div className="space-y-3">
                {syncHistory.map(operation => (
                  <div 
                    key={operation.id}
                    className="p-3 bg-gray-50 rounded-lg border border-gray-200"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center space-x-2">
                        {getStatusIcon(operation.status)}
                        <span className="text-sm font-medium">
                          {operation.operation_type} #{operation.id}
                        </span>
                      </div>
                      <span className="text-xs text-gray-500">
                        {new Date(operation.created_at).toLocaleDateString()}
                      </span>
                    </div>
                    
                    {operation.error_message && (
                      <p className="text-xs text-red-600 mb-2">{operation.error_message}</p>
                    )}
                    
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-gray-500">
                        {operation.completed_at 
                          ? `Completed ${formatTimestamp(operation.completed_at)}`
                          : `Started ${formatTimestamp(operation.created_at)}`
                        }
                      </span>
                      
                      {operation.status === 'completed' && (
                        <button
                          onClick={() => handleRollback(operation.id)}
                          className="flex items-center text-xs text-blue-600 hover:text-blue-800 transition-colors"
                        >
                          <Undo className="w-3 h-3 mr-1" />
                          Rollback
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <History className="w-12 h-12 text-gray-300 mx-auto mb-3" />
                <p className="text-gray-500 text-sm">No sync history</p>
                <p className="text-gray-400 text-xs mt-1">
                  Your completed operations will appear here
                </p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-gray-200 bg-gray-50">
        <div className="text-xs text-gray-500 text-center">
          {connected ? (
            <>
              <span className="inline-block w-2 h-2 bg-green-500 rounded-full mr-2"></span>
              Real-time updates enabled
            </>
          ) : (
            <>
              <span className="inline-block w-2 h-2 bg-red-500 rounded-full mr-2"></span>
              {connecting ? 'Reconnecting...' : 'Updates unavailable'}
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default SyncStatusPanel;