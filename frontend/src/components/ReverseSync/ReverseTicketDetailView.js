// frontend/src/components/ReverseSync/ReverseTicketDetailView.js
import React, { useState } from 'react';
import {
  ArrowLeft, CheckCircle, AlertCircle, PlusCircle, Loader2,
  Calendar, User, Tag, FileText, ExternalLink
} from 'lucide-react';

const ReverseTicketDetailView = ({
  type,
  analysisData,
  selectedCreator,
  onBack,
  onCreateTickets,
  loading
}) => {
  const [selectedIssues, setSelectedIssues] = useState([]);
  const [expandedTicket, setExpandedTicket] = useState(null);

  const { matched = [], missing_asana = [] } = analysisData;

  // Get tickets based on type
  const tickets = type === 'matched' ? matched : missing_asana;

  const toggleIssueSelection = (issueId) => {
    setSelectedIssues(prev =>
      prev.includes(issueId)
        ? prev.filter(id => id !== issueId)
        : [...prev, issueId]
    );
  };

  const toggleSelectAll = () => {
    if (type === 'missing') {
      if (selectedIssues.length === missing_asana.length) {
        setSelectedIssues([]);
      } else {
        setSelectedIssues(missing_asana.map(issue => issue.id));
      }
    }
  };

  const handleCreate = () => {
    if (selectedIssues.length === 0 && missing_asana.length > 0) {
      onCreateTickets([]);
    } else {
      onCreateTickets(selectedIssues);
    }
  };

  const formatDate = (timestamp) => {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const getTypeInfo = () => {
    const typeConfig = {
      matched: {
        title: 'Already in Asana',
        description: 'Tickets that exist in both YouTrack and Asana',
        icon: CheckCircle,
        color: 'green',
        bgColor: 'bg-green-50',
        borderColor: 'border-green-200',
        textColor: 'text-green-900'
      },
      missing: {
        title: 'Missing in Asana',
        description: 'Tickets that exist in YouTrack but not in Asana',
        icon: AlertCircle,
        color: 'amber',
        bgColor: 'bg-amber-50',
        borderColor: 'border-amber-200',
        textColor: 'text-amber-900'
      }
    };
    return typeConfig[type] || typeConfig.matched;
  };

  const typeInfo = getTypeInfo();
  const TypeIcon = typeInfo.icon;

  return (
    <div className="min-h-screen">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="mb-8">
          <button
            onClick={onBack}
            className="flex items-center text-gray-600 hover:text-gray-900 mb-4 transition-colors"
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Summary
          </button>

          <div className={`glass-panel ${typeInfo.bgColor} ${typeInfo.borderColor} border rounded-lg p-6`}>
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <TypeIcon className={`w-8 h-8 text-${typeInfo.color}-600 mr-4`} />
                <div>
                  <h1 className={`text-2xl font-bold ${typeInfo.textColor} mb-1`}>
                    {typeInfo.title} ({tickets.length})
                  </h1>
                  <p className={`text-sm text-${typeInfo.color}-700`}>
                    {typeInfo.description} • {selectedCreator === 'All' ? 'All Users' : selectedCreator}
                  </p>
                </div>
              </div>

              {type === 'missing' && missing_asana.length > 0 && (
                <div className="flex space-x-2">
                  <button
                    onClick={toggleSelectAll}
                    className="glass-panel bg-white text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-50 transition-colors border border-gray-200"
                  >
                    {selectedIssues.length === missing_asana.length ? 'Deselect All' : 'Select All'}
                  </button>
                  <button
                    onClick={handleCreate}
                    disabled={loading}
                    className="bg-gradient-to-r from-green-600 to-emerald-600 text-white px-6 py-2 rounded-lg hover:from-green-700 hover:to-emerald-700 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center shadow-lg"
                  >
                    {loading ? (
                      <>
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        Creating...
                      </>
                    ) : (
                      <>
                        <PlusCircle className="w-4 h-4 mr-2" />
                        Create {selectedIssues.length > 0 ? `${selectedIssues.length} Selected` : 'All'}
                      </>
                    )}
                  </button>
                </div>
              )}
            </div>

            {type === 'missing' && selectedIssues.length > 0 && (
              <div className="mt-4 px-4 py-2 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-700">
                {selectedIssues.length} ticket(s) selected for creation
              </div>
            )}
          </div>
        </div>

        {/* Tickets List */}
        <div className="space-y-4">
          {type === 'missing' && missing_asana.map((issue) => (
            <div
              key={issue.id}
              className={`glass-panel border rounded-lg p-6 transition-all ${
                selectedIssues.includes(issue.id)
                  ? 'border-blue-400 bg-blue-50 shadow-md'
                  : 'border-gray-200 bg-white hover:shadow-md'
              }`}
            >
              <div className="flex items-start">
                {/* Checkbox */}
                <input
                  type="checkbox"
                  checked={selectedIssues.includes(issue.id)}
                  onChange={() => toggleIssueSelection(issue.id)}
                  className="mt-1 mr-4 w-5 h-5 cursor-pointer"
                />

                {/* Ticket Content */}
                <div className="flex-1">
                  {/* Header */}
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center space-x-3">
                      <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-bold bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-sm">
                        {issue.id}
                      </span>
                      <h3 className="text-lg font-semibold text-gray-900">{issue.summary}</h3>
                    </div>
                    <button
                      onClick={() => setExpandedTicket(expandedTicket === issue.id ? null : issue.id)}
                      className="text-sm text-blue-600 hover:text-blue-700 font-medium"
                    >
                      {expandedTicket === issue.id ? 'Hide Details' : 'Show Details'}
                    </button>
                  </div>

                  {/* Metadata */}
                  <div className="flex flex-wrap gap-4 mb-3">
                    {issue.state && (
                      <div className="flex items-center text-sm">
                        <FileText className="w-4 h-4 mr-1.5 text-blue-600" />
                        <span className="text-gray-700">State:</span>
                        <span className="ml-1 px-2 py-0.5 rounded bg-blue-100 text-blue-800 font-medium">
                          {issue.state}
                        </span>
                      </div>
                    )}
                    {issue.subsystem && (
                      <div className="flex items-center text-sm">
                        <Tag className="w-4 h-4 mr-1.5 text-purple-600" />
                        <span className="text-gray-700">Subsystem:</span>
                        <span className="ml-1 px-2 py-0.5 rounded bg-purple-100 text-purple-800 font-medium">
                          {issue.subsystem}
                        </span>
                      </div>
                    )}
                    {issue.created_by && (
                      <div className="flex items-center text-sm">
                        <User className="w-4 h-4 mr-1.5 text-gray-600" />
                        <span className="text-gray-700">Creator:</span>
                        <span className="ml-1 font-medium text-gray-900">{issue.created_by}</span>
                      </div>
                    )}
                    {issue.created && (
                      <div className="flex items-center text-sm">
                        <Calendar className="w-4 h-4 mr-1.5 text-gray-600" />
                        <span className="text-gray-700">Created:</span>
                        <span className="ml-1 text-gray-900">{formatDate(issue.created)}</span>
                      </div>
                    )}
                  </div>

                  {/* Description Preview */}
                  {issue.description && (
                    <div className="mt-3 p-3 bg-gray-50 rounded-lg border border-gray-200">
                      <p className="text-sm text-gray-700 line-clamp-2">
                        {issue.description}
                      </p>
                    </div>
                  )}

                  {/* Expanded Details */}
                  {expandedTicket === issue.id && (
                    <div className="mt-4 pt-4 border-t border-gray-200">
                      <h4 className="font-semibold text-gray-900 mb-2">Full Description</h4>
                      <div className="p-4 bg-white rounded-lg border border-gray-200 text-sm text-gray-700 whitespace-pre-wrap max-h-64 overflow-y-auto">
                        {issue.description || 'No description available'}
                      </div>

                      {issue.attachments && issue.attachments.length > 0 && (
                        <div className="mt-4">
                          <h4 className="font-semibold text-gray-900 mb-2">
                            Attachments ({issue.attachments.length})
                          </h4>
                          <div className="space-y-2">
                            {issue.attachments.map((attachment, idx) => (
                              <div
                                key={idx}
                                className="flex items-center justify-between p-2 bg-gray-50 rounded border border-gray-200"
                              >
                                <span className="text-sm text-gray-700">{attachment.name}</span>
                                <span className="text-xs text-gray-500">
                                  {(attachment.size / 1024).toFixed(1)} KB
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}

          {type === 'matched' && matched.map((item) => (
            <div
              key={item.youtrack_issue.id}
              className="glass-panel border border-gray-200 bg-white rounded-lg p-6 hover:shadow-md transition-all"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  {/* Header */}
                  <div className="flex items-center space-x-3 mb-3">
                    <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-bold bg-gradient-to-r from-green-500 to-emerald-600 text-white shadow-sm">
                      {item.youtrack_issue.id}
                    </span>
                    <h3 className="text-lg font-semibold text-gray-900">
                      {item.youtrack_issue.summary}
                    </h3>
                    <CheckCircle className="w-5 h-5 text-green-600" />
                  </div>

                  {/* Asana Task Info */}
                  <div className="flex items-center space-x-4 text-sm">
                    <div className="flex items-center">
                      <ExternalLink className="w-4 h-4 mr-1.5 text-gray-600" />
                      <span className="text-gray-700">Asana Task ID:</span>
                      <span className="ml-1 font-mono text-gray-900 font-medium">
                        {item.asana_task_id}
                      </span>
                    </div>
                    <span className="px-3 py-1 rounded-full bg-green-100 text-green-800 font-medium text-xs">
                      ✓ Synced
                    </span>
                  </div>

                  {/* Metadata */}
                  {item.youtrack_issue.state && (
                    <div className="mt-3 flex items-center text-sm">
                      <FileText className="w-4 h-4 mr-1.5 text-blue-600" />
                      <span className="text-gray-700">State:</span>
                      <span className="ml-1 px-2 py-0.5 rounded bg-blue-100 text-blue-800 font-medium">
                        {item.youtrack_issue.state}
                      </span>
                    </div>
                  )}

                  {/* Description Preview */}
                  {item.youtrack_issue.description && (
                    <div className="mt-3 p-3 bg-gray-50 rounded-lg border border-gray-200">
                      <p className="text-sm text-gray-700 line-clamp-2">
                        {item.youtrack_issue.description}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Empty State */}
        {tickets.length === 0 && (
          <div className="glass-panel border border-gray-200 rounded-lg p-12 text-center">
            <TypeIcon className={`w-16 h-16 text-${typeInfo.color}-600 mx-auto mb-4 opacity-50`} />
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              No {type === 'matched' ? 'Matched' : 'Missing'} Tickets
            </h3>
            <p className="text-gray-600">
              {type === 'matched'
                ? 'No tickets found that exist in both systems'
                : 'All tickets are already synced to Asana'}
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ReverseTicketDetailView;
