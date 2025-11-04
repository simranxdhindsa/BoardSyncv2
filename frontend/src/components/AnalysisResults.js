import React, { useState, useEffect } from 'react';
import { AlertTriangle, CheckCircle, Clock, Plus, ArrowLeft, RefreshCw, Tag, Eye, EyeOff, RotateCcw, History } from 'lucide-react';
import TicketDetailView from './TicketDetailView';
import SyncHistory from './SyncHistory';
import { analyzeTickets, getUserSettings, getSyncHistory, rollbackSync } from '../services/api';
import '../styles/sync-history-glass.css';

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
  
  // Re-analyze functionality
  const [reAnalyzeLoading, setReAnalyzeLoading] = useState(false);
  
  // LOCAL STATE for optimistic updates
  const [localAnalysisData, setLocalAnalysisData] = useState(analysisData);

  // Column mappings state
  const [columnMappings, setColumnMappings] = useState([]);

  // Rollback state
  const [showSyncHistory, setShowSyncHistory] = useState(false);
  const [lastOperationId, setLastOperationId] = useState(null);
  const [undoingSync, setUndoingSync] = useState(false);

  // Update local data when prop changes
  useEffect(() => {
    setLocalAnalysisData(analysisData);
  }, [analysisData]);

  // Load last operation on mount
  useEffect(() => {
    const loadLastOperation = async () => {
      try {
        const response = await getSyncHistory(1);
        const operations = response.operations || response.data || [];
        if (operations.length > 0 && operations[0].status === 'completed') {
          setLastOperationId(operations[0].id);
        }
      } catch (error) {
        console.error('Failed to load last operation:', error);
      }
    };
    loadLastOperation();
  }, []);

  // Load column mappings on mount
  useEffect(() => {
    const loadColumnMappings = async () => {
      try {
        const response = await getUserSettings();
        const settings = response.data || response;
        setColumnMappings(settings.column_mappings?.asana_to_youtrack || []);
      } catch (error) {
        console.error('Failed to load column mappings:', error);
      }
    };
    loadColumnMappings();
  }, []);

  // Set navbar slots for Sync History button
  useEffect(() => {
    if (setNavBarSlots && !detailView) {
      setNavBarSlots(
        null, // left slot
        <button
          onClick={() => setShowSyncHistory(!showSyncHistory)}
          className="flex items-center justify-center h-10 w-10 bg-gradient-to-br from-purple-500 to-pink-600 rounded-lg shadow-sm text-white hover:shadow-md transition-shadow"
          title="Sync History"
        >
          <History className="w-7 h-7" strokeWidth={2.5} />
        </button>
      );
    }
    return () => {
      if (setNavBarSlots && !detailView) {
        setNavBarSlots(null, null);
      }
    };
  }, [setNavBarSlots, showSyncHistory, detailView]);

  // Data extraction
  let analysis = null;
  let summary = null;

  if (localAnalysisData) {
    analysis = localAnalysisData.analysis;
    
    if (!analysis && localAnalysisData.data) {
      analysis = localAnalysisData.data.analysis || localAnalysisData.data;
    }
    
    if (!analysis) {
      analysis = localAnalysisData;
    }

    summary = localAnalysisData.summary || 
              localAnalysisData.data?.summary ||
              localAnalysisData;
  }

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

  // OPTIMISTIC UPDATE HELPER - Only called after successful API response
  const moveTicketToMatched = (ticketId, fromCategory) => {
    setLocalAnalysisData(prev => {
      const newData = { ...prev };
      const newAnalysis = { ...safeAnalysis };
      
      // Find and remove ticket from source category
      let movedTicket = null;
      if (fromCategory === 'mismatched' && newAnalysis.mismatched) {
        const ticketIndex = newAnalysis.mismatched.findIndex(t => 
          (t.asana_task?.gid || t.gid) === ticketId
        );
        if (ticketIndex !== -1) {
          movedTicket = newAnalysis.mismatched[ticketIndex];
          newAnalysis.mismatched = newAnalysis.mismatched.filter((_, i) => i !== ticketIndex);
        }
      } else if (fromCategory === 'missing' && newAnalysis.missing_youtrack) {
        const ticketIndex = newAnalysis.missing_youtrack.findIndex(t => t.gid === ticketId);
        if (ticketIndex !== -1) {
          movedTicket = newAnalysis.missing_youtrack[ticketIndex];
          newAnalysis.missing_youtrack = newAnalysis.missing_youtrack.filter((_, i) => i !== ticketIndex);
        }
      }
      
      // Add to matched if ticket was found
      if (movedTicket) {
        newAnalysis.matched = newAnalysis.matched || [];
        const matchedTicket = {
          asana_task: movedTicket.asana_task || { 
            gid: movedTicket.gid, 
            name: movedTicket.name 
          },
          youtrack_issue: movedTicket.youtrack_issue || { 
            id: ticketId 
          },
          asana_status: movedTicket.asana_status,
          youtrack_status: movedTicket.asana_status,
          asana_tags: movedTicket.asana_tags || movedTicket.tags?.map(t => t.name) || []
        };
        newAnalysis.matched.push(matchedTicket);
      }
      
      // Update summary counts
      const newSummary = {
        ...safeSummary,
        matched: (newAnalysis.matched?.length || 0),
        mismatched: (newAnalysis.mismatched?.length || 0),
        missing_youtrack: (newAnalysis.missing_youtrack?.length || 0)
      };
      
      return {
        ...newData,
        analysis: newAnalysis,
        summary: newSummary
      };
    });
  };

  // SILENT BACKGROUND REFRESH
  const silentRefreshAnalysis = async () => {
    try {
      const data = await analyzeTickets(selectedColumn);
      setLocalAnalysisData({
        ...data,
        analyzedColumn: selectedColumn
      });
    } catch (error) {
      console.error('Silent refresh failed:', error);
    }
  };

  const handleReAnalyze = async () => {
    setReAnalyzeLoading(true);
    try {
      const data = await analyzeTickets(selectedColumn);
      setLocalAnalysisData({
        ...data,
        analyzedColumn: selectedColumn
      });
    } catch (error) {
      console.error('Re-analysis failed:', error);
      alert('Re-analysis failed: ' + error.message);
    } finally {
      setReAnalyzeLoading(false);
    }
  };

  const handleSummaryCardClick = (type) => {
    setDetailView({ type, column: selectedColumn });
  };

  const handleBackFromDetail = () => {
    setDetailView(null);
  };

  const handleUndoLastSync = async () => {
    if (!lastOperationId) {
      alert('No recent sync operation to undo');
      return;
    }

    if (!window.confirm('Are you sure you want to undo the last sync? This will reverse all changes made during the sync.')) {
      return;
    }

    setUndoingSync(true);
    try {
      const response = await rollbackSync(lastOperationId);

      if (response.success) {
        alert('Sync undone successfully! Reloading analysis...');
        setLastOperationId(null);
        // Reload the analysis
        window.location.reload();
      } else {
        alert('Rollback completed with errors: ' + (response.result?.errors?.join(', ') || 'Unknown error'));
      }
    } catch (error) {
      alert('Failed to undo sync: ' + error.message);
    } finally {
      setUndoingSync(false);
    }
  };

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
        onTicketMoved={moveTicketToMatched}
        onSilentRefresh={silentRefreshAnalysis}
      />
    );
  }

  // SYNC HANDLER - Wait for API success before moving
  const handleSyncTicket = async (ticketId) => {
    setSyncing(prev => ({ ...prev, [ticketId]: true }));
    
    try {
      // Wait for actual sync to complete
      await onSync(ticketId);
      
      // Only move ticket after successful sync
      moveTicketToMatched(ticketId, 'mismatched');
      
      // Show success feedback
      setSyncedTickets(prev => new Set([...prev, ticketId]));
      setTimeout(() => {
        setSyncedTickets(prev => {
          const newSet = new Set(prev);
          newSet.delete(ticketId);
          return newSet;
        });
      }, 2000);
      
      // Silent refresh in background
      setTimeout(() => silentRefreshAnalysis(), 3000);
    } catch (error) {
      console.error('Sync failed:', error);
      alert('Sync failed: ' + error.message);
    } finally {
      setSyncing(prev => ({ ...prev, [ticketId]: false }));
    }
  };

  // SYNC ALL HANDLER - Wait for all API calls
  const handleSyncAll = async () => {
    const mismatchedTickets = syncableMismatched;
    if (mismatchedTickets.length === 0) return;
    
    setSyncAllLoading(true);
    
    try {
      // Wait for all syncs to complete
      const syncPromises = mismatchedTickets.map(ticket => 
        onSync(ticket.asana_task?.gid || ticket.gid)
      );
      
      await Promise.all(syncPromises);
      
      // Only move tickets after all syncs succeed
      mismatchedTickets.forEach(ticket => {
        const ticketId = ticket.asana_task?.gid || ticket.gid;
        moveTicketToMatched(ticketId, 'mismatched');
      });
      
      // Silent refresh in background
      setTimeout(() => silentRefreshAnalysis(), 3000);
    } catch (error) {
      console.error('Some tickets failed to sync:', error);
      alert('Some tickets failed to sync. Please try again.');
      // Refresh to show accurate state
      await handleReAnalyze();
    } finally {
      setSyncAllLoading(false);
    }
  };

  // CREATE HANDLER - Wait for API success
  const handleCreateTicket = async (task, index) => {
    const taskId = task.gid;
    setCreating(prev => ({ ...prev, [taskId]: true }));
    
    try {
      // Wait for actual create to complete
      await onCreateSingle(taskId);
      
      // Only move ticket after successful create
      moveTicketToMatched(taskId, 'missing');
      
      // Show success feedback
      setCreatedTickets(prev => new Set([...prev, taskId]));
      setTimeout(() => {
        setCreatedTickets(prev => {
          const newSet = new Set(prev);
          newSet.delete(taskId);
          return newSet;
        });
      }, 2000);
      
      // Silent refresh in background
      setTimeout(() => silentRefreshAnalysis(), 3000);
    } catch (error) {
      console.error('Create failed:', error);
      alert('Create failed: ' + error.message);
    } finally {
      setCreating(prev => ({ ...prev, [taskId]: false }));
    }
  };

  // CREATE ALL HANDLER - Wait for API success
  const handleCreateAll = async () => {
    setCreateAllLoading(true);

    try {
      const missingTickets = syncableMissing;
      
      // Wait for create to complete
      await onCreateMissing();
      
      // Only move tickets after successful create
      missingTickets.forEach(task => {
        moveTicketToMatched(task.gid, 'missing');
      });
      
      // Silent refresh in background
      setTimeout(() => silentRefreshAnalysis(), 3000);
    } catch (error) {
      console.error('Failed to create tickets:', error);
      alert('Failed to create tickets: ' + error.message);
      // Refresh to show accurate state
      await handleReAnalyze();
    } finally {
      setCreateAllLoading(false);
    }
  };

  // Helper function to check if a column is marked as display_only
  const isDisplayOnlyColumn = (columnName) => {
    if (!columnMappings || columnMappings.length === 0) return false;
    const mapping = columnMappings.find(m =>
      m.asana_column.toLowerCase().replace(/\s+/g, '_') === columnName.toLowerCase()
    );
    return mapping?.display_only === true;
  };

  // Helper function to get all display-only column data from analysis
  const getDisplayOnlyColumns = () => {
    const displayOnlyData = [];

    // Get all display-only columns from column mappings
    columnMappings.forEach(mapping => {
      if (mapping.display_only) {
        const columnKey = mapping.asana_column.toLowerCase().replace(/\s+/g, '_');
        const tickets = [];

        // Check backend's special fields first (findings_tickets, ready_for_stage, blocked_tickets)
        if (safeAnalysis[columnKey] && Array.isArray(safeAnalysis[columnKey])) {
          tickets.push(...safeAnalysis[columnKey]);
        } else if (columnKey === 'findings' && safeAnalysis.findings_tickets) {
          tickets.push(...safeAnalysis.findings_tickets);
        } else if (columnKey === 'ready_for_stage' && safeAnalysis.ready_for_stage) {
          tickets.push(...safeAnalysis.ready_for_stage);
        } else if (columnKey === 'blocked' && safeAnalysis.blocked_tickets) {
          tickets.push(...safeAnalysis.blocked_tickets.map(bt => bt.asana_task || bt));
        }

        // Also collect from matched, mismatched, and missing arrays based on asana_status/section name
        const allTickets = [
          ...(safeAnalysis.matched || []).map(t => ({ ...(t.asana_task || t), _source: 'matched' })),
          ...(safeAnalysis.mismatched || []).map(t => ({ ...(t.asana_task || t), _source: 'mismatched' })),
          ...(safeAnalysis.missing_youtrack || []).map(t => ({ ...t, _source: 'missing' }))
        ];

        allTickets.forEach(ticket => {
          const ticketStatus = (ticket.memberships?.[0]?.section?.name || '').toLowerCase().replace(/\s+/g, '_');
          if (ticketStatus === columnKey) {
            // Avoid duplicates
            const isDuplicate = tickets.some(t => t.gid === ticket.gid);
            if (!isDuplicate) {
              tickets.push(ticket);
            }
          }
        });

        if (tickets.length > 0) {
          displayOnlyData.push({
            key: columnKey,
            label: mapping.asana_column,
            tickets: tickets,
            count: tickets.length
          });
        }
      }
    });

    return displayOnlyData;
  };

  const displayOnlyColumns = getDisplayOnlyColumns();

  // Check if the current selected column is display-only
  const isCurrentColumnDisplayOnly = selectedColumn && isDisplayOnlyColumn(selectedColumn);

  // Filter tickets to EXCLUDE display-only column tickets from sync sections
  const getSyncableTickets = (tickets) => {
    if (!tickets || !Array.isArray(tickets)) return [];
    return tickets.filter(ticket => {
      const status = ticket.asana_status || ticket.memberships?.[0]?.section?.name || '';
      const columnKey = status.toLowerCase().replace(/\s+/g, '_');
      return !isDisplayOnlyColumn(columnKey);
    });
  };

  // Filtered ticket arrays (exclude display-only columns)
  const syncableMismatched = getSyncableTickets(safeAnalysis.mismatched);
  const syncableMissing = getSyncableTickets(safeAnalysis.missing_youtrack);

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

  if (!localAnalysisData) {
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

  const hasAnyData = summaryData.matched > 0 || summaryData.mismatched > 0 || summaryData.missing_youtrack > 0 || summaryData.findings_tickets > 0;
  
  if (!hasAnyData) {
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

  return (
    <div className="min-h-screen">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* Header with Re-analyze Button */}
        <div className="mb-8 flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              Analysis Results - {selectedColumn?.toUpperCase?.().replace(/_/g, ' ') || 'ALL'}
            </h1>
            <p className="text-gray-600">Review mismatches, sync tickets, and manage tags. Click on any summary card to see detailed views.</p>
          </div>

          <div className="flex items-center space-x-3">
            {lastOperationId && (
              <button
                onClick={handleUndoLastSync}
                disabled={undoingSync}
                className="undo-sync-button"
              >
                {undoingSync ? (
                  <>
                    <div className="sync-loading-spinner inline-block mr-2"></div>
                    Undoing...
                  </>
                ) : (
                  <>
                    <RotateCcw className="w-4 h-4 mr-2" />
                    Undo Last Sync
                  </>
                )}
              </button>
            )}

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
        </div>

        {/* Sync History Panel */}
        {showSyncHistory && (
          <div className="mb-8">
            <SyncHistory
              onSuccess={(msg) => alert(msg)}
              onError={(msg) => alert(msg)}
            />
          </div>
        )}

        {/* High Priority Alerts */}
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

        {/* Summary Cards - CLICKABLE */}
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

        {/* Mismatched Tickets Preview - Only show if NOT display-only column */}
        {!isCurrentColumnDisplayOnly && syncableMismatched.length > 0 && (
          <div className="glass-panel border border-gray-200 rounded-lg p-6 mb-6">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900">
                Mismatched Tickets ({syncableMismatched.length})
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
                  {syncableMismatched.slice(0, 5).map((ticket) => {
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
              {syncableMismatched.length > 5 && (
                <div className="mt-4 text-center">
                  <button
                    onClick={() => handleSummaryCardClick('mismatched')}
                    className="glass-panel interactive-element bg-blue-50 border border-blue-200 text-blue-700 px-6 py-3 rounded-lg hover:bg-blue-100 hover:border-blue-300 transition-all font-medium inline-flex items-center"
                  >
                    <Eye className="w-4 h-4 mr-2" />
                    View all {syncableMismatched.length} mismatched tickets
                    <ArrowLeft className="w-4 h-4 ml-2 rotate-180" />
                  </button>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Missing Tickets Preview - Only show if NOT display-only column */}
        {!isCurrentColumnDisplayOnly && syncableMissing.length > 0 && (
          <div className="glass-panel border border-gray-200 rounded-lg p-6 mb-6">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900">
                Missing in YouTrack ({syncableMissing.length})
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
              {syncableMissing.slice(0, 6).map((task, index) => {
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
            
            {syncableMissing.length > 6 && (
              <div className="mt-4 text-center">
                <button
                  onClick={() => handleSummaryCardClick('missing')}
                  className="glass-panel interactive-element bg-blue-50 border border-blue-200 text-blue-700 px-6 py-3 rounded-lg hover:bg-blue-100 hover:border-blue-300 transition-all font-medium inline-flex items-center"
                >
                  <Eye className="w-4 h-4 mr-2" />
                  View all {syncableMissing.length} missing tickets
                  <ArrowLeft className="w-4 h-4 ml-2 rotate-180" />
                </button>
              </div>
            )}
          </div>
        )}

        {/* Display Only Sections - Dynamic - Only show if IS display-only column */}
        {isCurrentColumnDisplayOnly && displayOnlyColumns.length > 0 && (
          <div className="glass-panel border border-gray-200 rounded-lg p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Display Only Sections</h2>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {displayOnlyColumns.map((column, index) => {
                // Assign different colors to different columns
                const colors = [
                  { bg: 'bg-green-50', border: 'border-green-200', text: 'text-green-700', hover: 'hover:bg-green-100 hover:border-green-300' },
                  { bg: 'bg-orange-50', border: 'border-orange-200', text: 'text-orange-700', hover: 'hover:bg-orange-100 hover:border-orange-300' },
                  { bg: 'bg-purple-50', border: 'border-purple-200', text: 'text-purple-700', hover: 'hover:bg-purple-100 hover:border-purple-300' },
                  { bg: 'bg-pink-50', border: 'border-pink-200', text: 'text-pink-700', hover: 'hover:bg-pink-100 hover:border-pink-300' },
                  { bg: 'bg-indigo-50', border: 'border-indigo-200', text: 'text-indigo-700', hover: 'hover:bg-indigo-100 hover:border-indigo-300' }
                ];
                const colorScheme = colors[index % colors.length];

                return (
                  <div key={column.key}>
                    <h3 className="text-lg font-medium text-gray-700 mb-3">
                      {column.label} ({column.count})
                    </h3>
                    <div className="space-y-2">
                      {column.tickets.slice(0, 3).map((task) => (
                        <div key={task.gid} className={`glass-panel ${colorScheme.bg} ${colorScheme.border} border rounded-lg p-3`}>
                          <p className="font-medium text-gray-900">{task.name}</p>
                          <div className="mt-1">
                            <TagsDisplay tags={task.tags?.map(t => t.name) || []} />
                          </div>
                          <p className={`text-sm ${colorScheme.text} mt-1`}>Display only - not synced</p>
                        </div>
                      ))}
                      {column.tickets.length > 3 && (
                        <button
                          onClick={() => handleSummaryCardClick(column.key)}
                          className={`glass-panel interactive-element ${colorScheme.bg} ${colorScheme.border} border ${colorScheme.text} px-4 py-2 rounded-lg ${colorScheme.hover} transition-all text-sm font-medium inline-flex items-center`}
                        >
                          <Eye className="w-3 h-3 mr-2" />
                          View all {column.tickets.length} tickets
                          <ArrowLeft className="w-3 h-3 ml-2 rotate-180" />
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default AnalysisResults;