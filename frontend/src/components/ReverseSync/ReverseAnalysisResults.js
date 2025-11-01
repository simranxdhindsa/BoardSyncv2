// frontend/src/components/ReverseSync/ReverseAnalysisResults.js
import React, { useState } from 'react';
import { ArrowLeft, CheckCircle, AlertCircle, PlusCircle, Loader2, Eye, Calendar, User, Tag, FileText, RefreshCw } from 'lucide-react';
import ReverseTicketDetailView from './ReverseTicketDetailView';

const ReverseAnalysisResults = ({ analysisData, selectedCreator, onBack, onCreateTickets, onReanalyze, loading }) => {
  const [selectedIssues, setSelectedIssues] = useState([]);
  const [detailView, setDetailView] = useState(null); // null or { type: 'matched' | 'missing' }

  const { matched = [], missing_asana = [] } = analysisData;

  // Summary data
  const summaryData = {
    matched: matched.length,
    missing: missing_asana.length,
    total: matched.length + missing_asana.length,
    syncRate: matched.length + missing_asana.length > 0
      ? Math.round((matched.length / (matched.length + missing_asana.length)) * 100)
      : 100
  };

  const toggleIssueSelection = (issueId) => {
    setSelectedIssues(prev =>
      prev.includes(issueId)
        ? prev.filter(id => id !== issueId)
        : [...prev, issueId]
    );
  };

  const toggleSelectAll = () => {
    if (selectedIssues.length === missing_asana.length) {
      setSelectedIssues([]);
    } else {
      setSelectedIssues(missing_asana.map(issue => issue.id));
    }
  };

  const handleCreate = () => {
    if (selectedIssues.length === 0 && missing_asana.length > 0) {
      // Create all if none selected
      onCreateTickets([]);
    } else {
      // Create selected
      onCreateTickets(selectedIssues);
    }
  };

  const handleSummaryCardClick = (type) => {
    setDetailView({ type });
  };

  const formatDate = (timestamp) => {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  };

  // If detail view is active, show it
  if (detailView) {
    return (
      <ReverseTicketDetailView
        type={detailView.type}
        analysisData={analysisData}
        selectedCreator={selectedCreator}
        onBack={() => setDetailView(null)}
        onCreateTickets={onCreateTickets}
        loading={loading}
      />
    );
  }

  return (
    <div className="min-h-screen">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="mb-8 flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              Reverse Sync Analysis - {selectedCreator === 'All' ? 'ALL USERS' : selectedCreator.toUpperCase()}
            </h1>
            <p className="text-gray-600">
              Review YouTrack tickets and create missing ones in Asana. Click on any summary card to see detailed views.
            </p>
          </div>

          <div style={{ display: 'flex', gap: '12px' }}>
            <button
              onClick={onReanalyze}
              disabled={loading}
              className="glass-button"
              style={{
                padding: '8px 16px',
                background: loading ? 'rgba(59, 130, 246, 0.5)' : 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                color: 'white',
                border: 'none',
                borderRadius: '8px',
                display: 'flex',
                alignItems: 'center',
                cursor: loading ? 'not-allowed' : 'pointer',
                fontWeight: '500',
                transition: 'all 0.2s',
                opacity: loading ? 0.6 : 1
              }}
            >
              {loading ? (
                <>
                  <Loader2 style={{ width: '16px', height: '16px', marginRight: '8px', animation: 'spin 1s linear infinite' }} />
                  Analyzing...
                </>
              ) : (
                <>
                  <RefreshCw style={{ width: '16px', height: '16px', marginRight: '8px' }} />
                  Re-analyze
                </>
              )}
            </button>
            <button
              onClick={onBack}
              className="glass-button"
              style={{
                padding: '8px 16px',
                background: 'rgba(255, 255, 255, 0.8)',
                border: '1px solid rgba(226, 232, 240, 0.8)',
                color: '#4b5563',
                borderRadius: '8px',
                display: 'flex',
                alignItems: 'center',
                cursor: 'pointer',
                fontWeight: '500',
                transition: 'all 0.2s'
              }}
            >
              <ArrowLeft style={{ width: '16px', height: '16px', marginRight: '8px' }} />
              Back
            </button>
          </div>
        </div>

        {/* Summary Cards - CLICKABLE */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
          {/* Matched Card */}
          <div
            className="glass-panel bg-green-50 border border-green-200 rounded-lg p-6 cursor-pointer hover:shadow-lg transition-all"
            onClick={() => handleSummaryCardClick('matched')}
          >
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-semibold text-green-900 mb-1">Already in Asana</h3>
                <p className="text-3xl font-bold text-green-600">{summaryData.matched}</p>
              </div>
              <CheckCircle className="w-10 h-10 text-green-600 opacity-80" />
            </div>
            <div className="flex items-center text-xs text-green-700">
              <Eye className="w-3 h-3 mr-1" />
              Click to view details
            </div>
          </div>

          {/* Missing Card */}
          <div
            className="glass-panel bg-amber-50 border border-amber-200 rounded-lg p-6 cursor-pointer hover:shadow-lg transition-all"
            onClick={() => handleSummaryCardClick('missing')}
          >
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-semibold text-amber-900 mb-1">Missing in Asana</h3>
                <p className="text-3xl font-bold text-amber-600">{summaryData.missing}</p>
              </div>
              <AlertCircle className="w-10 h-10 text-amber-600 opacity-80" />
            </div>
            <div className="flex items-center text-xs text-amber-700">
              <Eye className="w-3 h-3 mr-1" />
              Click to view & create
            </div>
          </div>

          {/* Total Tickets Card */}
          <div className="glass-panel bg-blue-50 border border-blue-200 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-semibold text-blue-900 mb-1">Total Tickets</h3>
                <p className="text-3xl font-bold text-blue-600">{summaryData.total}</p>
              </div>
              <FileText className="w-10 h-10 text-blue-600 opacity-80" />
            </div>
            <p className="text-xs text-blue-700">
              Analyzed from YouTrack
            </p>
          </div>

          {/* Sync Rate Card */}
          <div className="glass-panel bg-indigo-50 border border-indigo-200 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-semibold text-indigo-900 mb-1">Sync Rate</h3>
                <p className="text-3xl font-bold text-indigo-600">{summaryData.syncRate}%</p>
              </div>
              <RefreshCw className="w-10 h-10 text-indigo-600 opacity-80" />
            </div>
            <p className="text-xs text-indigo-700">
              Tickets already synced
            </p>
          </div>
        </div>

        {/* Missing Tickets Preview - Only show if there are missing tickets */}
        {missing_asana.length > 0 && (
          <div className="glass-panel border border-gray-200 rounded-lg p-6 mb-6">
            <div className="flex justify-between items-center mb-6">
              <h2 className="text-xl font-semibold text-gray-900">
                Missing Tickets ({missing_asana.length})
              </h2>
              <div className="flex space-x-2">
                <button
                  onClick={toggleSelectAll}
                  className="glass-panel interactive-element bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors flex items-center"
                >
                  {selectedIssues.length === missing_asana.length ? 'Deselect All' : 'Select All'}
                </button>
                <button
                  onClick={() => handleSummaryCardClick('missing')}
                  className="glass-panel interactive-element bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors flex items-center"
                >
                  <Eye className="w-4 h-4 mr-2" />
                  View All
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
            </div>

            {selectedIssues.length > 0 && (
              <div className="mb-4 px-4 py-2 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-700">
                {selectedIssues.length} ticket(s) selected for creation
              </div>
            )}

            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-200">
                    <th className="text-left p-3 font-medium text-gray-700 w-12">
                      <input
                        type="checkbox"
                        checked={selectedIssues.length === missing_asana.length && missing_asana.length > 0}
                        onChange={toggleSelectAll}
                        className="w-4 h-4 cursor-pointer"
                      />
                    </th>
                    <th className="text-left p-3 font-medium text-gray-700">Ticket ID</th>
                    <th className="text-left p-3 font-medium text-gray-700">Summary</th>
                    <th className="text-left p-3 font-medium text-gray-700">State</th>
                    <th className="text-left p-3 font-medium text-gray-700">Subsystem</th>
                    <th className="text-left p-3 font-medium text-gray-700">Creator</th>
                    <th className="text-left p-3 font-medium text-gray-700">Created</th>
                  </tr>
                </thead>
                <tbody>
                  {missing_asana.slice(0, 10).map((issue, index) => (
                    <tr
                      key={issue.id}
                      className={`border-b border-gray-100 hover:bg-gray-50 transition-colors ${
                        index % 2 === 0 ? 'bg-white' : 'bg-gray-50'
                      }`}
                    >
                      <td className="p-3">
                        <input
                          type="checkbox"
                          checked={selectedIssues.includes(issue.id)}
                          onChange={() => toggleIssueSelection(issue.id)}
                          className="w-4 h-4 cursor-pointer"
                        />
                      </td>
                      <td className="p-3">
                        <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold bg-gradient-to-r from-blue-500 to-purple-600 text-white">
                          {issue.id}
                        </span>
                      </td>
                      <td className="p-3">
                        <div className="font-medium text-gray-900">{issue.summary}</div>
                        {issue.description && (
                          <div className="text-xs text-gray-500 mt-1 truncate max-w-md">
                            {issue.description.substring(0, 80)}...
                          </div>
                        )}
                      </td>
                      <td className="p-3">
                        <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-100 text-blue-800">
                          <FileText className="w-3 h-3 mr-1" />
                          {issue.state || 'N/A'}
                        </span>
                      </td>
                      <td className="p-3">
                        {issue.subsystem ? (
                          <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-purple-100 text-purple-800">
                            <Tag className="w-3 h-3 mr-1" />
                            {issue.subsystem}
                          </span>
                        ) : (
                          <span className="text-gray-400 text-xs">No subsystem</span>
                        )}
                      </td>
                      <td className="p-3">
                        <span className="inline-flex items-center text-sm text-gray-700">
                          <User className="w-3 h-3 mr-1 text-gray-500" />
                          {issue.created_by || 'N/A'}
                        </span>
                      </td>
                      <td className="p-3">
                        <span className="inline-flex items-center text-xs text-gray-600">
                          <Calendar className="w-3 h-3 mr-1 text-gray-500" />
                          {formatDate(issue.created)}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {missing_asana.length > 10 && (
              <div className="mt-4 text-center">
                <button
                  onClick={() => handleSummaryCardClick('missing')}
                  className="text-blue-600 hover:text-blue-700 font-medium text-sm flex items-center justify-center mx-auto"
                >
                  View all {missing_asana.length} missing tickets
                  <ArrowLeft className="w-4 h-4 ml-1 rotate-180" />
                </button>
              </div>
            )}
          </div>
        )}

        {/* Matched Tickets Preview */}
        {matched.length > 0 && (
          <div className="glass-panel border border-gray-200 rounded-lg p-6">
            <div className="flex justify-between items-center mb-6">
              <h2 className="text-xl font-semibold text-gray-900">
                Already Synced Tickets ({matched.length})
              </h2>
              <button
                onClick={() => handleSummaryCardClick('matched')}
                className="glass-panel interactive-element bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors flex items-center"
              >
                <Eye className="w-4 h-4 mr-2" />
                View All
              </button>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-200">
                    <th className="text-left p-3 font-medium text-gray-700">YouTrack ID</th>
                    <th className="text-left p-3 font-medium text-gray-700">Summary</th>
                    <th className="text-left p-3 font-medium text-gray-700">Asana Task ID</th>
                    <th className="text-left p-3 font-medium text-gray-700">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {matched.slice(0, 5).map((item, index) => (
                    <tr
                      key={item.youtrack_issue.id}
                      className={`border-b border-gray-100 hover:bg-green-50 transition-colors ${
                        index % 2 === 0 ? 'bg-white' : 'bg-gray-50'
                      }`}
                    >
                      <td className="p-3">
                        <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold bg-gradient-to-r from-green-500 to-emerald-600 text-white">
                          {item.youtrack_issue.id}
                        </span>
                      </td>
                      <td className="p-3">
                        <div className="font-medium text-gray-900">{item.youtrack_issue.summary}</div>
                      </td>
                      <td className="p-3">
                        <span className="text-sm text-gray-600 font-mono">{item.asana_task_id}</span>
                      </td>
                      <td className="p-3">
                        <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-green-100 text-green-800">
                          <CheckCircle className="w-3 h-3 mr-1" />
                          Synced
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {matched.length > 5 && (
              <div className="mt-4 text-center">
                <button
                  onClick={() => handleSummaryCardClick('matched')}
                  className="text-green-600 hover:text-green-700 font-medium text-sm flex items-center justify-center mx-auto"
                >
                  View all {matched.length} matched tickets
                  <ArrowLeft className="w-4 h-4 ml-1 rotate-180" />
                </button>
              </div>
            )}
          </div>
        )}

        {/* No Missing Tickets Message */}
        {missing_asana.length === 0 && (
          <div className="glass-panel border border-green-200 bg-green-50 rounded-lg p-8 text-center">
            <CheckCircle className="w-16 h-16 text-green-600 mx-auto mb-4" />
            <h3 className="text-xl font-semibold text-green-900 mb-2">
              All Tickets Synced!
            </h3>
            <p className="text-green-700">
              No missing tickets found for {selectedCreator === 'All' ? 'any user' : selectedCreator}.
              All YouTrack tickets are already in Asana.
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ReverseAnalysisResults;
