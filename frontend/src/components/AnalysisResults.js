import React, { useState } from 'react';
import { AlertTriangle, CheckCircle, Clock, Plus, ArrowLeft, RefreshCw, Tag, Eye, EyeOff } from 'lucide-react';
import TicketDetailView from './TicketDetailView';
import { analyzeTickets } from '../services/api';

const AnalysisResults = ({ 
  analysisData, 
  selectedColumn, 
  onBack, 
  onSync, 
  onCreateSingle, 
  onCreateMissing, 
  setNavBarSlots 
}) => {
  const [syncing, setSyncing] = useState({});
  const [creating, setCreating] = useState({});
  const [syncAllLoading, setSyncAllLoading] = useState(false);
  const [createAllLoading, setCreateAllLoading] = useState(false);
  const [syncedTickets, setSyncedTickets] = useState(new Set());
  const [createdTickets, setCreatedTickets] = useState(new Set());
  
  // Detail view state
  const [detailView, setDetailView] = useState(null);
  
  // NEW: Re-analyze functionality
  const [reAnalyzeLoading, setReAnalyzeLoading] = useState(false);
  const [currentAnalysisData, setCurrentAnalysisData] = useState(analysisData);

  // FIXED: Better data extraction with multiple fallback paths
  console.log('🔍 Analysis Data Debug:', currentAnalysisData);

  // Try multiple paths to get the analysis data
  let analysis = null;
  let summary = null;

  if (currentAnalysisData) {
    // Path 1: Direct analysis property
    analysis = currentAnalysisData.analysis;
    
    // Path 2: Data property with analysis
    if (!analysis && currentAnalysisData.data) {
      analysis = currentAnalysisData.data.analysis || currentAnalysisData.data;
    }
    
    // Path 3: Root level data (if response is already unwrapped)
    if (!analysis) {
      analysis = currentAnalysisData;
    }

    // Summary extraction
    summary = currentAnalysisData.summary || 
              currentAnalysisData.data?.summary ||
              currentAnalysisData;
  }

  console.log('🔍 Extracted Analysis:', analysis);
  console.log('🔍 Extracted Summary:', summary);

  // Ensure analysis has expected structure
  const safeAnalysis = analysis || {};
  const safeSummary = summary || {};

  // Build summary with fallbacks
  const summaryData = {
    findings_alerts: safeSummary.findings_alerts || 0,
    matched: safeSummary.matched || (safeAnalysis.matched ? safeAnalysis.matched.length : 0),
    mismatched: safeSummary.mismatched || (safeAnalysis.mismatched ? safeAnalysis.mismatched.length : 0),
    missing_youtrack: safeSummary.missing_youtrack || (safeAnalysis.missing_youtrack ? safeAnalysis.missing_youtrack.length : 0),
    tag_mismatches: safeSummary.tag_mismatches || 0,
    ignored: safeSummary.ignored || (safeAnalysis.ignored ? safeAnalysis.ignored.length : 0),
    ready_for_stage: safeSummary.ready_for_stage || (safeAnalysis.ready_for_stage ? safeAnalysis.ready_for_stage.length : 0),
    findings_tickets: safeSummary.findings_tickets || (safeAnalysis.findings_tickets ? safeAnalysis.findings_tickets.length : 0),
    blocked_tickets: safeSummary.blocked_tickets || (safeAnalysis.blocked_tickets ? safeAnalysis.blocked_tickets.length : 0),
    orphaned_youtrack: safeSummary.orphaned_youtrack || (safeAnalysis.orphaned_youtrack ? safeAnalysis.orphaned_youtrack.length : 0)
  };

  console.log('🔍 Final Summary Data:', summaryData);

  // FIXED: Early return check with better logging
  if (!currentAnalysisData) {
    console.warn('❌ No currentAnalysisData provided');
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-blue-50">
        <div className="text-center">
          <AlertTriangle className="w-12 h-12 text-yellow-600 mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-gray-900 mb-2">No Analysis Data</h2>
          <p className="text-gray-600 mb-4">Please run an analysis first to see results.</p>
          <button
            onClick={onBack}
            className="flex items-center px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Dashboard
          </button>
        </div>
      </div>
    );
  }

  // Check if we have any meaningful data
  const hasAnyData = summaryData.matched > 0 || summaryData.mismatched > 0 || summaryData.missing_youtrack > 0 || summaryData.findings_tickets > 0;
  
  if (!hasAnyData) {
    console.warn('❌ No meaningful analysis data found in:', currentAnalysisData);
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-blue-50">
        <div className="text-center">
          <CheckCircle className="w-12 h-12 text-green-600 mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-gray-900 mb-2">Perfect Sync!</h2>
          <p className="text-gray-600 mb-4">
            All tickets are perfectly synchronized. No actions needed for {selectedColumn || 'selected columns'}.
          </p>
          <button
            onClick={onBack}
            className="flex items-center px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Dashboard
          </button>
        </div>
      </div>
    );
  }

  // NEW: Re-analyze the same column
  const handleReAnalyze = async () => {
    setReAnalyzeLoading(true);
    try {
      console.log('🔄 Re-analyzing column:', selectedColumn);
      const data = await analyzeTickets(selectedColumn);
      console.log('🔄 Re-analysis response:', data);
      setCurrentAnalysisData(data);
    } catch (error) {
      console.error('❌ Re-analysis failed:', error);
      alert('Re-analysis failed: ' + error.message);
    } finally {
      setReAnalyzeLoading(false);
    }
  };

  // Handle clicking on summary cards to drill down
  const handleSummaryCardClick = (type) => {
    console.log('🎯 Opening detail view for type:', type, 'column:', selectedColumn);
    setDetailView({ type, column: selectedColumn });
  };

  // Handle back from detail view
  const handleBackFromDetail = () => {
    setDetailView(null);
  };

  // Show detail view if selected
  if (detailView) {
    return (
      <TicketDetailView
        type={detailView.type}
        column={detailView.column}
        onBack={handleBackFromDetail}
        onSync={onSync}
        onCreateSingle={onCreateSingle}
        onCreateMissing={onCreateMissing}
        setNavBarSlots={setNavBarSlots}
      />
    );
  }

  // Handle individual ticket sync
  const handleSyncTicket = async (ticketId) => {
    setSyncing(prev => ({ ...prev, [ticketId]: true }));
    
    try {
      await onSync(ticketId);
      setSyncedTickets(prev => new Set([...prev, ticketId]));
      
      setTimeout(() => {
        setSyncedTickets(prev => {
          const newSet = new Set(prev);
          newSet.delete(ticketId);
          return newSet;
        });
      }, 2000);
    } catch (error) {
      console.error('❌ Sync failed:', error);
    } finally {
      setSyncing(prev => ({ ...prev, [ticketId]: false }));
    }
  };

  // Handle sync all
  const handleSyncAll = async () => {
    const mismatchedTickets = safeAnalysis.mismatched || [];
    if (mismatchedTickets.length === 0) return;
    
    setSyncAllLoading(true);
    
    try {
      for (const ticket of mismatchedTickets) {
        await onSync(ticket.asana_task?.gid || ticket.gid);
      }
    } catch (error) {
      console.error('❌ Some tickets failed to sync');
    } finally {
      setSyncAllLoading(false);
    }
  };

  // Handle individual ticket creation
  const handleCreateTicket = async (task, index) => {
    const taskId = task.gid;
    setCreating(prev => ({ ...prev, [taskId]: true }));
    
    try {
      await onCreateSingle(taskId);
      setCreatedTickets(prev => new Set([...prev, taskId]));
      
      setTimeout(() => {
        setCreatedTickets(prev => {
          const newSet = new Set(prev);
          newSet.delete(taskId);
          return newSet;
        });
      }, 2000);
    } catch (error) {
      console.error('❌ Create failed:', error);
    } finally {
      setCreating(prev => ({ ...prev, [taskId]: false }));
    }
  };

  // Handle create all
  const handleCreateAll = async () => {
    setCreateAllLoading(true);
    
    try {
      await onCreateMissing();
    } catch (error) {
      console.error('❌ Failed to create tickets:', error);
    } finally {
      setCreateAllLoading(false);
    }
  };

  // Display tags component
  const TagsDisplay = ({ tags }) => {
    if (!tags || tags.length === 0) return <span className="text-gray-400">No tags</span>;
    
    return (
      <div className="flex flex-wrap gap-1">
        {tags.map((tag, index) => (
          <span key={index} className="tag-glass inline-flex items-center">
            <Tag className="w-3 h-3 mr-1" />
            {tag}
          </span>
        ))}
      </div>
    );
  };

  console.log('✅ Rendering analysis results with summary:', summaryData);

  return (
    <div className="min-h-screen">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* Header with Re-analyze Button */}
        <div className="mb-8 flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              Analysis Results - {selectedColumn?.toUpperCase?.() || 'ALL'}
            </h1>
            <p className="text-gray-600">Review mismatches, sync tickets, and manage tags. Click on any summary card to see detailed views.</p>
          </div>
          
          {/* NEW: Re-analyze Button */}
          <button
            onClick={handleReAnalyze}
            disabled={reAnalyzeLoading}
            className="flex items-center bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 font-medium"
          >
            {reAnalyzeLoading ? (
              <>
                <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                Re-analyzing...
              </>
            ) : (
              <>
                <RefreshCw className="w-4 h-4 mr-2" />
                Re-analyze 
              </>
            )}
          </button>
        </div>

        {/* High Priority Alerts - SAFE ACCESS */}
        {summaryData.findings_alerts > 0 && (
          <div className="glass-panel bg-red-50 border border-red-200 rounded-lg p-6 mb-6">
            <div className="flex items-center mb-4">
              <AlertTriangle className="w-6 h-6 text-red-600 mr-2" />
              <h2 className="text-xl font-semibold text-red-900">
                High Priority Alerts ({summaryData.findings_alerts})
              </h2>
            </div>
            <div className="glass-panel bg-red-100 border border-red-300 rounded-lg p-4">
              <p className="text-red-800 font-medium">
                Tickets found in Findings (Asana) but still active in YouTrack
              </p>
            </div>
          </div>
        )}

        {/* Summary Cards with Glass Theme - CLICKABLE - SAFE ACCESS */}
        <div className="grid grid-cols-1 md:grid-cols-5 gap-4 mb-8">
          <div 
            className="glass-panel bg-green-50 border border-green-200 rounded-lg p-4 cursor-pointer hover:shadow-lg transition-all"
            onClick={() => handleSummaryCardClick('matched')}
          >
            <div className="flex items-center">
              <CheckCircle className="w-6 h-6 text-green-600 mr-2" />
              <div>
                <h3 className="text-sm font-semibold text-green-900">Matched</h3>
                <p className="text-2xl font-bold text-green-600">{summaryData.matched}</p>
              </div>
            </div>
            <div className="mt-2 flex items-center text-xs text-green-700">
              <Eye className="w-3 h-3 mr-1" />
              Click to view details
            </div>
          </div>

          <div 
            className="glass-panel bg-yellow-50 border border-yellow-200 rounded-lg p-4 cursor-pointer hover:shadow-lg transition-all"
            onClick={() => handleSummaryCardClick('mismatched')}
          >
            <div className="flex items-center">
              <Clock className="w-6 h-6 text-yellow-600 mr-2" />
              <div>
                <h3 className="text-sm font-semibold text-yellow-900">Mismatched</h3>
                <p className="text-2xl font-bold text-yellow-600">{summaryData.mismatched}</p>
              </div>
            </div>
            <div className="mt-2 flex items-center text-xs text-yellow-700">
              <Eye className="w-3 h-3 mr-1" />
              Click to view details
            </div>
          </div>

          <div 
            className="glass-panel bg-blue-50 border border-blue-200 rounded-lg p-4 cursor-pointer hover:shadow-lg transition-all"
            onClick={() => handleSummaryCardClick('missing')}
          >
            <div className="flex items-center">
              <Plus className="w-6 h-6 text-blue-600 mr-2" />
              <div>
                <h3 className="text-sm font-semibold text-blue-900">Missing</h3>
                <p className="text-2xl font-bold text-blue-600">{summaryData.missing_youtrack}</p>
              </div>
            </div>
            <div className="mt-2 flex items-center text-xs text-blue-700">
              <Eye className="w-3 h-3 mr-1" />
              Click to view details
            </div>
          </div>

          <div 
            className="glass-panel bg-purple-50 border border-purple-200 rounded-lg p-4 cursor-pointer hover:shadow-lg transition-all"
            onClick={() => handleSummaryCardClick('ignored')}
          >
            <div className="flex items-center">
              <EyeOff className="w-6 h-6 text-purple-600 mr-2" />
              <div>
                <h3 className="text-sm font-semibold text-purple-900">Ignored</h3>
                <p className="text-2xl font-bold text-purple-600">{summaryData.ignored}</p>
              </div>
            </div>
            <div className="mt-2 flex items-center text-xs text-purple-700">
              <Eye className="w-3 h-3 mr-1" />
              Click to manage
            </div>
          </div>

          <div className="glass-panel bg-indigo-50 border border-indigo-200 rounded-lg p-4">
            <div className="flex items-center">
              <RefreshCw className="w-6 h-6 text-indigo-600 mr-2" />
              <div>
                <h3 className="text-sm font-semibold text-indigo-900">Sync Rate</h3>
                <p className="text-2xl font-bold text-indigo-600">
                  {Math.round(((summaryData.matched) / (summaryData.matched + summaryData.mismatched)) * 100) || 0}%
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Mismatched Tickets - SAFE ACCESS */}
        {summaryData.mismatched > 0 && (
          <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6 mb-6">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900">
                Mismatched Tickets ({summaryData.mismatched})
              </h2>
              <div className="flex space-x-2">
                <button 
                  onClick={() => handleSummaryCardClick('mismatched')}
                  className="glass-panel interactive-element bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors flex items-center"
                >
                  <Eye className="w-4 h-4 mr-2" />
                  View All
                </button>
                <button 
                  onClick={handleSyncAll}
                  disabled={syncAllLoading}
                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center"
                >
                  {syncAllLoading ? (
                    <>
                      <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                      Syncing All...
                    </>
                  ) : (
                    <>
                      <RefreshCw className="w-4 h-4 mr-2" />
                      Sync All
                    </>
                  )}
                </button>
              </div>
            </div>
            
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-3 font-medium text-gray-700">Ticket Name</th>
                    <th className="text-left p-3 font-medium text-gray-700">Status</th>
                    <th className="text-left p-3 font-medium text-gray-700">Tags/Subsystem</th>
                    <th className="text-left p-3 font-medium text-gray-700">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {(safeAnalysis.mismatched || []).slice(0, 5).map((ticket) => {
                    const ticketId = ticket.asana_task?.gid || ticket.gid;
                    const isSyncing = syncing[ticketId];
                    const isSynced = syncedTickets.has(ticketId);
                    
                    return (
                      <tr key={ticketId} className="border-b hover:bg-gray-50">
                        <td className="p-3">
                          <div className="font-medium text-gray-900">{ticket.asana_task?.name || ticket.name}</div>
                          <div className="text-sm text-gray-500">ID: {ticketId}</div>
                        </td>
                        <td className="p-3">
                          <div className="space-y-1">
                            <div className="flex items-center space-x-2">
                              <span className="status-badge matched">
                                Asana: {ticket.asana_status}
                              </span>
                            </div>
                            <div className="flex items-center space-x-2">
                              <span className="status-badge mismatched">
                                YouTrack: {ticket.youtrack_status}
                              </span>
                            </div>
                          </div>
                        </td>
                        <td className="p-3">
                          <div className="space-y-2">
                            <div>
                              <div className="text-xs text-gray-500 mb-1">Asana Tags:</div>
                              <TagsDisplay tags={ticket.asana_tags} />
                            </div>
                          </div>
                        </td>
                        <td className="p-3">
                          <div className="flex space-x-2">
                            {isSynced ? (
                              <div className="bg-green-100 text-green-800 px-3 py-1 rounded text-sm flex items-center">
                                <CheckCircle className="w-4 h-4 mr-1" />
                                Synced!
                              </div>
                            ) : (
                              <button
                                onClick={() => handleSyncTicket(ticketId)}
                                disabled={isSyncing}
                                className="bg-blue-600 text-white px-3 py-1 rounded text-sm hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center"
                              >
                                {isSyncing ? (
                                  <>
                                    <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
                                    Syncing...
                                  </>
                                ) : (
                                  'Sync'
                                )}
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              {(safeAnalysis.mismatched || []).length > 5 && (
                <div className="mt-4 text-center">
                  <button 
                    onClick={() => handleSummaryCardClick('mismatched')}
                    className="glass-panel interactive-element bg-blue-50 border border-blue-200 text-blue-700 px-6 py-3 rounded-lg hover:bg-blue-100 hover:border-blue-300 transition-all font-medium inline-flex items-center"
                  >
                    <Eye className="w-4 h-4 mr-2" />
                    View all {(safeAnalysis.mismatched || []).length} mismatched tickets
                    <ArrowLeft className="w-4 h-4 ml-2 rotate-180" />
                  </button>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Missing Tickets - SAFE ACCESS */}
        {summaryData.missing_youtrack > 0 && (
          <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6 mb-6">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900">
                Missing in YouTrack ({summaryData.missing_youtrack})
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
              {(safeAnalysis.missing_youtrack || []).slice(0, 6).map((task, index) => {
                const taskId = task.gid;
                const isCreating = creating[taskId];
                const isCreated = createdTickets.has(taskId);
                
                return (
                  <div key={taskId} className="glass-panel border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
                    <h3 className="font-medium text-gray-900 mb-2">{task.name}</h3>
                    <div className="text-sm text-gray-600 mb-2">
                      Section: {task.memberships?.[0]?.section?.name || 'No Section'}
                    </div>
                    
                    <div className="mb-3">
                      <div className="text-xs text-gray-500 mb-1">Tags:</div>
                      <TagsDisplay tags={task.tags?.map(t => t.name) || []} />
                    </div>
                    
                    {isCreated ? (
                      <div className="w-full bg-green-100 text-green-800 px-3 py-2 rounded text-sm text-center flex items-center justify-center">
                        <CheckCircle className="w-4 h-4 mr-2" />
                        Created!
                      </div>
                    ) : (
                      <button 
                        onClick={() => handleCreateTicket(task, index)}
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
            
            {(safeAnalysis.missing_youtrack || []).length > 6 && (
              <div className="mt-4 text-center">
                <button 
                  onClick={() => handleSummaryCardClick('missing')}
                  className="glass-panel interactive-element bg-blue-50 border border-blue-200 text-blue-700 px-6 py-3 rounded-lg hover:bg-blue-100 hover:border-blue-300 transition-all font-medium inline-flex items-center"
                >
                  <Eye className="w-4 h-4 mr-2" />
                  View all {(safeAnalysis.missing_youtrack || []).length} missing tickets
                  <ArrowLeft className="w-4 h-4 ml-2 rotate-180" />
                </button>
              </div>
            )}
          </div>
        )}

        {/* Display Only Sections - SAFE ACCESS */}
        {(summaryData.ready_for_stage > 0 || summaryData.findings_tickets > 0) && (
          <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Display Only Sections</h2>
            
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {summaryData.ready_for_stage > 0 && (
                <div>
                  <h3 className="text-lg font-medium text-gray-700 mb-3">
                    Ready for Stage ({summaryData.ready_for_stage})
                  </h3>
                  <div className="space-y-2">
                    {(safeAnalysis.ready_for_stage || []).slice(0, 3).map((task) => (
                      <div key={task.gid} className="glass-panel bg-green-50 border border-green-200 rounded-lg p-3">
                        <p className="font-medium text-gray-900">{task.name}</p>
                        <div className="mt-1">
                          <TagsDisplay tags={task.tags?.map(t => t.name) || []} />
                        </div>
                        <p className="text-sm text-green-700 mt-1">Display only - not synced</p>
                      </div>
                    ))}
                    {(safeAnalysis.ready_for_stage || []).length > 3 && (
                      <button 
                        onClick={() => handleSummaryCardClick('ready_for_stage')}
                        className="glass-panel interactive-element bg-green-50 border border-green-200 text-green-700 px-4 py-2 rounded-lg hover:bg-green-100 hover:border-green-300 transition-all text-sm font-medium inline-flex items-center"
                      >
                        <Eye className="w-3 h-3 mr-2" />
                        View all {(safeAnalysis.ready_for_stage || []).length} tickets
                        <ArrowLeft className="w-3 h-3 ml-2 rotate-180" />
                      </button>
                    )}
                  </div>
                </div>
              )}

              {summaryData.findings_tickets > 0 && (
                <div>
                  <h3 className="text-lg font-medium text-gray-700 mb-3">
                    Findings ({summaryData.findings_tickets})
                  </h3>
                  <div className="space-y-2">
                    {(safeAnalysis.findings_tickets || []).slice(0, 3).map((task) => (
                      <div key={task.gid} className="glass-panel bg-orange-50 border border-orange-200 rounded-lg p-3">
                        <p className="font-medium text-gray-900">{task.name}</p>
                        <div className="mt-1">
                          <TagsDisplay tags={task.tags?.map(t => t.name) || []} />
                        </div>
                        <p className="text-sm text-orange-700 mt-1">Display only - not synced</p>
                      </div>
                    ))}
                    {(safeAnalysis.findings_tickets || []).length > 3 && (
                      <button 
                        onClick={() => handleSummaryCardClick('findings')}
                        className="glass-panel interactive-element bg-orange-50 border border-orange-200 text-orange-700 px-4 py-2 rounded-lg hover:bg-orange-100 hover:border-orange-300 transition-all text-sm font-medium inline-flex items-center"
                      >
                        <Eye className="w-3 h-3 mr-2" />
                        View all {(safeAnalysis.findings_tickets || []).length} tickets
                        <ArrowLeft className="w-3 h-3 ml-2 rotate-180" />
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Debug Information Panel - Only show in development */}
        {process.env.NODE_ENV !== 'production' && (
          <div className="mt-8 p-4 bg-gray-100 rounded-lg text-xs">
            <h4 className="font-semibold mb-2">Debug Information:</h4>
            <div className="space-y-1">
              <div>Analysis Data Keys: {Object.keys(currentAnalysisData || {}).join(', ')}</div>
              <div>Analysis Keys: {Object.keys(safeAnalysis).join(', ')}</div>
              <div>Summary Data: {JSON.stringify(summaryData, null, 2)}</div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default AnalysisResults;