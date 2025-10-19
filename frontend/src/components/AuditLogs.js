// FILE: frontend/src/components/AuditLogs.js
// Audit Logs Component with Filtering and CSV Export

import React, { useState, useEffect } from 'react';
import {
  getAuditLogs,
  exportAuditLogsCSV
} from '../services/api';
import {
  Filter,
  Download,
  Search,
  Calendar,
  X,
  FileText
} from 'lucide-react';
import '../styles/sync-history-glass.css';

const AuditLogs = ({ onSuccess, onError }) => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showFilters, setShowFilters] = useState(true);
  const [exporting, setExporting] = useState(false);

  const [filters, setFilters] = useState({
    ticketId: '',
    platform: '',
    actionType: '',
    startDate: '',
    endDate: '',
    limit: 50
  });

  useEffect(() => {
    loadAuditLogs();
  }, []);

  const loadAuditLogs = async () => {
    setLoading(true);
    try {
      const response = await getAuditLogs(filters);
      setLogs(response.logs || []);
    } catch (error) {
      onError?.('Failed to load audit logs: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (field, value) => {
    setFilters(prev => ({ ...prev, [field]: value }));
  };

  const handleApplyFilters = () => {
    loadAuditLogs();
  };

  const handleClearFilters = () => {
    setFilters({
      ticketId: '',
      platform: '',
      actionType: '',
      startDate: '',
      endDate: '',
      limit: 50
    });
  };

  const handleExportCSV = async () => {
    setExporting(true);
    try {
      const blob = await exportAuditLogsCSV(filters);

      // Create download link
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `audit_logs_${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      onSuccess?.('Audit logs exported successfully!');
    } catch (error) {
      onError?.('Failed to export audit logs: ' + error.message);
    } finally {
      setExporting(false);
    }
  };

  const getActionBadge = (actionType) => {
    const badges = {
      'created': 'action-badge action-badge-created',
      'updated': 'action-badge action-badge-updated',
      'deleted': 'action-badge action-badge-deleted',
      'status_changed': 'action-badge action-badge-status-changed',
      'ignored': 'action-badge action-badge-ignored',
      'rolled_back': 'action-badge action-badge-deleted',
      'mapping_added': 'action-badge action-badge-created'
    };

    return <span className={badges[actionType] || 'action-badge'}>{actionType}</span>;
  };

  const formatTimestamp = (timestamp) => {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  return (
    <div className="sync-history-container">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900 flex items-center">
          <FileText className="w-6 h-6 mr-2" />
          Audit Logs
        </h2>

        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowFilters(!showFilters)}
            className="view-details-button"
          >
            <Filter className="w-4 h-4 mr-2" />
            {showFilters ? 'Hide' : 'Show'} Filters
          </button>

          <button
            onClick={handleExportCSV}
            disabled={exporting || logs.length === 0}
            className="export-csv-button"
          >
            {exporting ? (
              <>
                <div className="sync-loading-spinner inline-block mr-2"></div>
                Exporting...
              </>
            ) : (
              <>
                <Download className="w-4 h-4 mr-2" />
                Export CSV
              </>
            )}
          </button>
        </div>
      </div>

      {/* Filters Panel */}
      {showFilters && (
        <div className="filter-panel">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
            {/* Ticket ID Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                <Search className="w-4 h-4 inline mr-1" />
                Ticket ID
              </label>
              <input
                type="text"
                className="filter-input"
                placeholder="ARD-123 or Asana GID"
                value={filters.ticketId}
                onChange={(e) => handleFilterChange('ticketId', e.target.value)}
              />
            </div>

            {/* Platform Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Platform
              </label>
              <select
                className="filter-select"
                value={filters.platform}
                onChange={(e) => handleFilterChange('platform', e.target.value)}
              >
                <option value="">All Platforms</option>
                <option value="youtrack">YouTrack</option>
                <option value="asana">Asana</option>
                <option value="mapping">Mapping</option>
                <option value="system">System</option>
              </select>
            </div>

            {/* Action Type Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Action Type
              </label>
              <select
                className="filter-select"
                value={filters.actionType}
                onChange={(e) => handleFilterChange('actionType', e.target.value)}
              >
                <option value="">All Actions</option>
                <option value="created">Created</option>
                <option value="updated">Updated</option>
                <option value="status_changed">Status Changed</option>
                <option value="deleted">Deleted</option>
                <option value="ignored">Ignored</option>
                <option value="rolled_back">Rolled Back</option>
                <option value="mapping_added">Mapping Added</option>
              </select>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Start Date */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                <Calendar className="w-4 h-4 inline mr-1" />
                Start Date
              </label>
              <input
                type="date"
                className="filter-input"
                value={filters.startDate}
                onChange={(e) => handleFilterChange('startDate', e.target.value)}
              />
            </div>

            {/* End Date */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                <Calendar className="w-4 h-4 inline mr-1" />
                End Date
              </label>
              <input
                type="date"
                className="filter-input"
                value={filters.endDate}
                onChange={(e) => handleFilterChange('endDate', e.target.value)}
              />
            </div>

            {/* Limit */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Limit
              </label>
              <select
                className="filter-select"
                value={filters.limit}
                onChange={(e) => handleFilterChange('limit', parseInt(e.target.value))}
              >
                <option value={50}>50 logs</option>
                <option value={100}>100 logs</option>
                <option value={200}>200 logs</option>
                <option value={500}>500 logs</option>
              </select>
            </div>
          </div>

          {/* Filter Actions */}
          <div className="flex items-center justify-end space-x-3 mt-4 pt-4 border-t border-white border-opacity-30">
            <button
              onClick={handleClearFilters}
              className="view-details-button"
            >
              <X className="w-4 h-4 mr-2" />
              Clear
            </button>
            <button
              onClick={handleApplyFilters}
              className="export-csv-button"
            >
              <Filter className="w-4 h-4 mr-2" />
              Apply Filters
            </button>
          </div>
        </div>
      )}

      {/* Audit Logs Table */}
      {loading ? (
        <div className="text-center py-12">
          <div className="sync-loading-spinner mx-auto mb-4"></div>
          <p className="text-gray-600">Loading audit logs...</p>
        </div>
      ) : logs.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">
            <FileText className="w-16 h-16 mx-auto text-gray-400" />
          </div>
          <p className="empty-state-text">No audit logs found</p>
          <p className="text-sm text-gray-500 mt-2">Try adjusting your filters</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="audit-log-table w-full">
            <thead>
              <tr>
                <th>Timestamp</th>
                <th>Ticket ID</th>
                <th>Platform</th>
                <th>Action</th>
                <th>Field</th>
                <th>Old Value</th>
                <th>New Value</th>
                <th>User</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id}>
                  <td className="text-sm">{formatTimestamp(log.timestamp)}</td>
                  <td className="font-mono text-sm font-semibold">{log.ticket_id}</td>
                  <td>
                    <span className="tag-glass">{log.platform}</span>
                  </td>
                  <td>{getActionBadge(log.action_type)}</td>
                  <td className="text-sm text-gray-600">{log.field_name || '-'}</td>
                  <td className="text-sm text-gray-600 max-w-xs truncate">
                    {log.old_value || '-'}
                  </td>
                  <td className="text-sm text-gray-600 max-w-xs truncate">
                    {log.new_value || '-'}
                  </td>
                  <td className="text-sm text-gray-600">{log.user_email}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="mt-4 text-center text-sm text-gray-600">
            Showing {logs.length} log{logs.length !== 1 ? 's' : ''}
          </div>
        </div>
      )}
    </div>
  );
};

export default AuditLogs;
