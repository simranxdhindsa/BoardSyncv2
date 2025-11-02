// frontend/src/components/ReverseSync/ReverseAnalysisResults.js
import React, { useState, useEffect } from 'react';
import { ArrowLeft, CheckCircle, AlertCircle, Plus, Eye, FileText, RefreshCw } from 'lucide-react';
import ReverseTicketDetailView from './ReverseTicketDetailView';

const ReverseAnalysisResults = ({ analysisData, selectedCreator, onBack, onCreateTickets, onReanalyze, loading }) => {
  const [detailView, setDetailView] = useState(null); // null or { type: 'matched' | 'missing' }
  const [creating, setCreating] = useState({});
  const [createAllLoading, setCreateAllLoading] = useState(false);
  const [createdTickets, setCreatedTickets] = useState(new Set());

  // LOCAL STATE for optimistic updates
  const [localAnalysisData, setLocalAnalysisData] = useState(analysisData);

  // Update local data when prop changes
  useEffect(() => {
    setLocalAnalysisData(analysisData);
  }, [analysisData]);

  const { matched = [], missing_asana = [] } = localAnalysisData || {};

  // Summary data
  const summaryData = {
    matched: matched.length,
    missing: missing_asana.length,
    total: matched.length + missing_asana.length,
    syncRate: matched.length + missing_asana.length > 0
      ? Math.round((matched.length / (matched.length + missing_asana.length)) * 100)
      : 100
  };

  // CREATE HANDLER - Wait for API success
  const handleCreateTicket = async (ticket, index) => {
    const ticketId = ticket.id;
    setCreating(prev => ({ ...prev, [ticketId]: true }));

    try {
      // Wait for actual create to complete
      await onCreateTickets([ticketId]);

      // Show success feedback
      setCreatedTickets(prev => new Set([...prev, ticketId]));
      setTimeout(() => {
        setCreatedTickets(prev => {
          const newSet = new Set(prev);
          newSet.delete(ticketId);
          return newSet;
        });
      }, 2000);
    } catch (error) {
      console.error('Create failed:', error);
    } finally {
      setCreating(prev => ({ ...prev, [ticketId]: false }));
    }
  };

  // CREATE ALL HANDLER - Wait for API success
  const handleCreateAll = async () => {
    setCreateAllLoading(true);

    try {
      const ticketIds = missing_asana.map(t => t.id);

      // Wait for create to complete
      await onCreateTickets(ticketIds);
    } catch (error) {
      console.error('Failed to create tickets:', error);
    } finally {
      setCreateAllLoading(false);
    }
  };

  const handleSummaryCardClick = (type) => {
    setDetailView({ type });
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
                  <RefreshCw style={{ width: '16px', height: '16px', marginRight: '8px', animation: 'spin 1s linear infinite' }} />
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
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900">
                Missing in Asana ({missing_asana.length})
              </h2>
              <div className="flex space-x-2">
                <button
                  onClick={() => handleSummaryCardClick('missing')}
                  className="glass-panel interactive-element bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors flex items-center"
                >
                  <Eye className="w-4 h-4 mr-2" />
                  View All
                </button>
                <button
                  onClick={handleCreateAll}
                  disabled={createAllLoading}
                  className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition-colors flex items-center disabled:opacity-50"
                >
                  {createAllLoading ? (
                    <>
                      <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                      Creating All...
                    </>
                  ) : (
                    <>
                      <Plus className="w-4 h-4 mr-2" />
                      Create All
                    </>
                  )}
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {missing_asana.slice(0, 6).map((ticket, index) => {
                const ticketId = ticket.id;
                const isCreating = creating[ticketId];
                const isCreated = createdTickets.has(ticketId);

                return (
                  <div key={ticketId} className="glass-panel border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
                    <div className="flex items-start justify-between mb-2">
                      <h3 className="font-medium text-gray-900 flex-1">{ticket.summary}</h3>
                      <span className="ml-2 inline-flex items-center px-2 py-1 rounded-md text-xs font-semibold bg-gradient-to-r from-blue-500 to-purple-600 text-white">
                        {ticket.id}
                      </span>
                    </div>
                    <div className="text-sm text-gray-600 mb-2">
                      State: {ticket.state || 'Unknown'}
                    </div>
                    <div className="text-sm text-gray-600 mb-2">
                      Subsystem: {ticket.subsystem || 'None'}
                    </div>

                    <div className="mb-3">
                      <div className="text-xs text-gray-500 mb-1">Creator:</div>
                      <span className="text-sm text-gray-700">{ticket.created_by || 'Unknown'}</span>
                    </div>

                    {isCreated ? (
                      <div className="w-full bg-green-100 text-green-800 px-3 py-2 rounded text-sm text-center flex items-center justify-center">
                        <CheckCircle className="w-4 h-4 mr-2" />
                        Created!
                      </div>
                    ) : (
                      <button
                        onClick={() => handleCreateTicket(ticket, index)}
                        disabled={isCreating}
                        className="w-full bg-blue-600 text-white px-3 py-2 rounded hover:bg-blue-700 text-sm transition-colors disabled:opacity-50 flex items-center justify-center"
                      >
                        {isCreating ? (
                          <>
                            <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                            Creating...
                          </>
                        ) : (
                          <>
                            <Plus className="w-4 h-4 mr-2" />
                            Create
                          </>
                        )}
                      </button>
                    )}
                  </div>
                );
              })}
            </div>

            {missing_asana.length > 6 && (
              <div className="mt-4 text-center">
                <button
                  onClick={() => handleSummaryCardClick('missing')}
                  className="glass-panel interactive-element bg-blue-50 border border-blue-200 text-blue-700 px-6 py-3 rounded-lg hover:bg-blue-100 hover:border-blue-300 transition-all font-medium inline-flex items-center"
                >
                  <Eye className="w-4 h-4 mr-2" />
                  View all {missing_asana.length} missing tickets
                  <ArrowLeft className="w-4 h-4 ml-2 rotate-180" />
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
