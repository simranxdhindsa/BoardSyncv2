// FILE: frontend/src/components/TicketAuditTrail.js
// Ticket-specific audit trail component

import React, { useState, useEffect } from 'react';
import { getTicketHistory } from '../services/api';
import { Clock, Activity } from 'lucide-react';
import '../styles/sync-history-glass.css';

const TicketAuditTrail = ({ ticketId, onError }) => {
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (ticketId) {
      loadTicketHistory();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ticketId]);

  const loadTicketHistory = async () => {
    setLoading(true);
    try {
      const response = await getTicketHistory(ticketId);
      setHistory(response.history || []);
    } catch (error) {
      onError?.('Failed to load ticket history: ' + error.message);
      setHistory([]);
    } finally {
      setLoading(false);
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
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    let relative = '';
    if (diffMins < 1) relative = 'Just now';
    else if (diffMins < 60) relative = `${diffMins} min${diffMins > 1 ? 's' : ''} ago`;
    else if (diffHours < 24) relative = `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
    else if (diffDays < 7) relative = `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
    else relative = date.toLocaleDateString();

    const absolute = date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });

    return { relative, absolute };
  };

  if (loading) {
    return (
      <div className="text-center py-8">
        <div className="sync-loading-spinner mx-auto mb-3"></div>
        <p className="text-sm text-gray-600">Loading ticket history...</p>
      </div>
    );
  }

  if (history.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">
          <Activity className="w-12 h-12 mx-auto text-gray-400" />
        </div>
        <p className="empty-state-text">No history available</p>
        <p className="text-sm text-gray-500 mt-2">Changes to this ticket will appear here</p>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 flex items-center">
          <Activity className="w-5 h-5 mr-2" />
          Ticket History
        </h3>
        <span className="text-sm text-gray-600">{history.length} event{history.length !== 1 ? 's' : ''}</span>
      </div>

      <div className="timeline-container">
        {history.map((entry, index) => {
          const { relative, absolute } = formatTimestamp(entry.timestamp);
          const isLast = index === history.length - 1;

          return (
            <div key={entry.id} className="timeline-item">
              <div className="timeline-dot"></div>
              {!isLast && <div className="timeline-line"></div>}

              <div className="timeline-content">
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    {getActionBadge(entry.action_type)}
                    <span className="tag-glass text-xs">{entry.platform}</span>
                  </div>
                  <div className="text-right">
                    <div className="text-sm font-medium text-gray-700">{relative}</div>
                    <div className="text-xs text-gray-500">{absolute}</div>
                  </div>
                </div>

                <div className="space-y-1">
                  {entry.field_name && (
                    <div className="text-sm">
                      <span className="font-medium text-gray-700">Field:</span>{' '}
                      <span className="text-gray-600">{entry.field_name}</span>
                    </div>
                  )}

                  {(entry.old_value || entry.new_value) && (
                    <div className="text-sm">
                      {entry.old_value && (
                        <div>
                          <span className="font-medium text-gray-700">From:</span>{' '}
                          <span className="text-gray-600">{entry.old_value}</span>
                        </div>
                      )}
                      {entry.new_value && (
                        <div>
                          <span className="font-medium text-gray-700">To:</span>{' '}
                          <span className="text-gray-600">{entry.new_value}</span>
                        </div>
                      )}
                    </div>
                  )}

                  <div className="text-xs text-gray-500 mt-2">
                    <Clock className="w-3 h-3 inline mr-1" />
                    by {entry.user_email}
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default TicketAuditTrail;
