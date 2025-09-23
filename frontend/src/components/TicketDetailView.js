import React, { useState, useEffect } from 'react';
import { ArrowLeft, RefreshCw, Tag, EyeOff, Eye, Plus, CheckCircle, Clock, AlertTriangle, Trash2, X } from 'lucide-react';
import { getTicketsByType, ignoreTicket, unignoreTicket, deleteTickets } from '../services/api';

const TicketDetailView = ({ type, column, onBack, onSync, onCreateSingle, onCreateMissing, setNavBarSlots }) => {
  const [tickets, setTickets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [actionLoading, setActionLoading] = useState({});
  const [ignoredTickets, setIgnoredTickets] = useState(new Set());
  
  // Enhanced delete UX state
  const [deleteMode, setDeleteMode] = useState(false);
  const [selectedTickets, setSelectedTickets] = useState(new Set());
  const [lastSelectedIndex, setLastSelectedIndex] = useState(-1);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteSource, setDeleteSource] = useState('');

  // Create all loading state
  const [createAllLoading, setCreateAllLoading] = useState(false);

  useEffect(() => {
    loadTickets();
  }, [type, column]);

  // Configure NavBar content when this view is active
  useEffect(() => {
    const typeInfo = getTypeInfo();
    const IconComponent = typeInfo.icon;
    const left = (
      <div className="flex items-center space-x-4">
        <button 
          onClick={onBack}
          disabled={deleteMode}
          className="flex items-center bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors disabled:opacity-50"
        >
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Results
        </button>
        <div className="flex items-center">
          <IconComponent className={`w-6 h-6 mr-2 text-${getTypeInfo().color}-600`} />
          <div>
            <h1 className="text-xl font-semibold text-gray-900">
              {getTypeInfo().title}
              
              {column && column !== 'all_syncable' && (
                <span className="text-blue-600 ml-2">
                  • {column.replace('_', ' ').toUpperCase()} Column
                </span>
              )}
              {deleteMode && (
                <span className="text-red-600 ml-2">• Delete Mode</span>
              )}
            </h1>
            {/* <p className="text-sm text-gray-600">
              {getTypeInfo().description}
              
              {column && column !== 'all_syncable' && (
                <span className="text-blue-600 ml-1">
                  (filtered by {column.replace('_', ' ')} column)
                </span>
              )}
            </p> */}
          </div>
        </div>
      </div>
    );

    const right = (
      <div className="flex items-center space-x-3">
        {deleteMode ? (
          <>
            {selectedTickets.size > 0 && (
              <button
                onClick={handleSelectAll}
                className="flex items-center bg-red-100 text-red-700 px-3 py-2 rounded-lg hover:bg-red-200 transition-colors text-sm"
              >
                {selectedTickets.size === tickets.length ? 'Deselect All' : 'Select All'}
              </button>
            )}
            <button
              onClick={handleExitDeleteMode}
              className="flex items-center bg-gray-100 text-gray-700 px-3 py-2 rounded-lg hover:bg-gray-200 transition-colors text-sm"
            >
              <X className="w-4 h-4 mr-1" />
              Cancel
            </button>
          </>
        ) : (
          <>
            {/* Create All Button for missing tickets */}
            {type === 'missing' && tickets.length > 0 && onCreateMissing && (
              <button
                onClick={handleCreateAll}
                disabled={createAllLoading}
                className="flex items-center bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50 font-medium"
              >
                {createAllLoading ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    Creating All...
                  </>
                ) : (
                  <>
                    <Plus className="w-4 h-4 mr-2" />
                    Create All ({tickets.length})
                  </>
                )}
              </button>
            )}
            {type !== 'ignored' && tickets.length > 0 && (
              <button
                onClick={handleEnterDeleteMode}
                className="flex items-center bg-red-500 text-white px-4 py-2 rounded-lg hover:bg-red-600 transition-colors font-medium"
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Delete
              </button>
            )}
            <button
              onClick={loadTickets}
              disabled={loading}
              className="flex items-center bg-blue-100 text-blue-700 px-4 py-2 rounded-lg hover:bg-blue-200 transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </>
        )}
      </div>
    );

    if (setNavBarSlots) {
      setNavBarSlots(left, right);
    }

    return () => {
      if (setNavBarSlots) {
        setNavBarSlots(null, null);
      }
    };
  }, [type, column, deleteMode, selectedTickets, tickets.length, loading, createAllLoading]);

  // Clear selection when exiting delete mode
  useEffect(() => {
    if (!deleteMode) {
      setSelectedTickets(new Set());
      setLastSelectedIndex(-1);
    }
  }, [deleteMode]);

  const loadTickets = async () => {
    setLoading(true);
    setError(null);
    
    console.log('Loading tickets for type:', type, 'column:', column); // DEBUG
    
    try {
      // CRITICAL: Ensure we pass the column parameter correctly
      const response = await getTicketsByType(type, column || '');
      console.log('Received tickets response:', response); // DEBUG
      
      setTickets(response.tickets || []);
      setDeleteMode(false);
    } catch (err) {
      console.error('Failed to load tickets:', err); // DEBUG
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Handle create all missing tickets
  const handleCreateAll = async () => {
    if (!onCreateMissing || tickets.length === 0) return;
    
    setCreateAllLoading(true);
    try {
      await onCreateMissing();
      setTimeout(loadTickets, 1000);
    } catch (err) {
      console.error('Failed to create all tickets:', err);
      alert('Failed to create all tickets: ' + err.message);
    } finally {
      setCreateAllLoading(false);
    }
  };

  const handleIgnoreTicket = async (ticketId) => {
    setActionLoading(prev => ({ ...prev, [`ignore_${ticketId}`]: true }));
    try {
      await ignoreTicket(ticketId);
      setIgnoredTickets(prev => new Set([...prev, ticketId]));
      
      setTimeout(() => {
        setTickets(prev => prev.filter(ticket => 
          (ticket.gid || ticket.asana_task?.gid || ticket.id) !== ticketId
        ));
      }, 1000);
    } catch (err) {
      console.error('Failed to ignore ticket:', err);
      alert('Failed to ignore ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`ignore_${ticketId}`]: false }));
    }
  };

  const handleUnignoreTicket = async (ticketId) => {
    setActionLoading(prev => ({ ...prev, [`unignore_${ticketId}`]: true }));
    try {
      await unignoreTicket(ticketId);
      setIgnoredTickets(prev => {
        const newSet = new Set(prev);
        newSet.delete(ticketId);
        return newSet;
      });
      
      if (type === 'ignored') {
        setTickets(prev => prev.filter(id => id !== ticketId));
      }
    } catch (err) {
      console.error('Failed to unignore ticket:', err);
      alert('Failed to unignore ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`unignore_${ticketId}`]: false }));
    }
  };

  const handleSyncTicket = async (ticketId) => {
    setActionLoading(prev => ({ ...prev, [`sync_${ticketId}`]: true }));
    try {
      await onSync(ticketId);
      setTimeout(loadTickets, 1000);
    } catch (err) {
      console.error('Failed to sync ticket:', err);
      alert('Failed to sync ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`sync_${ticketId}`]: false }));
    }
  };

  const handleCreateTicket = async (taskId) => {
    setActionLoading(prev => ({ ...prev, [`create_${taskId}`]: true }));
    try {
      await onCreateSingle(taskId);
      setTimeout(loadTickets, 1000);
    } catch (err) {
      console.error('Failed to create ticket:', err);
      alert('Failed to create ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`create_${taskId}`]: false }));
    }
  };

  // Enhanced ticket selection with shift+click support
  const handleTicketClick = (ticketId, index, event) => {
    if (!deleteMode) return;
    
    event.preventDefault();
    
    if (event.shiftKey && lastSelectedIndex !== -1) {
      const startIndex = Math.min(lastSelectedIndex, index);
      const endIndex = Math.max(lastSelectedIndex, index);
      
      const newSelected = new Set(selectedTickets);
      for (let i = startIndex; i <= endIndex; i++) {
        const ticket = tickets[i];
        const ticketId = ticket.gid || ticket.asana_task?.gid || ticket.id || ticket;
        newSelected.add(ticketId);
      }
      setSelectedTickets(newSelected);
    } else {
      setSelectedTickets(prev => {
        const newSet = new Set(prev);
        if (newSet.has(ticketId)) {
          newSet.delete(ticketId);
        } else {
          newSet.add(ticketId);
        }
        return newSet;
      });
      setLastSelectedIndex(index);
    }
  };

  // Delete mode controls
  const handleEnterDeleteMode = () => {
    setDeleteMode(true);
  };

  const handleExitDeleteMode = () => {
    setDeleteMode(false);
  };

  const handleSelectAll = () => {
    if (selectedTickets.size === tickets.length) {
      setSelectedTickets(new Set());
    } else {
      const allTicketIds = tickets.map(ticket => 
        ticket.gid || ticket.asana_task?.gid || ticket.id || ticket
      );
      setSelectedTickets(new Set(allTicketIds));
    }
  };

  // Delete confirmation handlers
  const handleDeleteClick = (source) => {
    setDeleteSource(source);
    setShowDeleteConfirm(true);
  };

  const handleDeleteConfirm = async () => {
    setDeleteLoading(true);
    try {
      const ticketIds = Array.from(selectedTickets);
      const response = await deleteTickets(ticketIds, deleteSource);
      
      const successCount = response.success_count || 0;
      const failureCount = response.failure_count || 0;
      const summary = response.summary || `Processed ${ticketIds.length} tickets`;
      
      alert(`Delete Operation Complete:\n${summary}\n\nSuccessful: ${successCount}\nFailed: ${failureCount}`);
      
      setDeleteMode(false);
      await loadTickets();
      
    } catch (err) {
      console.error('Delete operation failed:', err);
      alert('Delete operation failed: ' + err.message);
    } finally {
      setDeleteLoading(false);
      setShowDeleteConfirm(false);
      setDeleteSource('');
    }
  };

  const handleDeleteCancel = () => {
    setShowDeleteConfirm(false);
    setDeleteSource('');
  };

  const getTypeInfo = () => {
    const typeConfig = {
      matched: {
        title: 'Matched Tickets',
        description: 'Tickets that are synchronized between Asana and YouTrack',
        icon: CheckCircle,
        color: 'green'
      },
      mismatched: {
        title: 'Mismatched Tickets',
        description: 'Tickets with different statuses between Asana and YouTrack',
        icon: Clock,
        color: 'yellow'
      },
      missing: {
        title: 'Missing Tickets',
        description: 'Tickets that exist in Asana but not in YouTrack',
        icon: Plus,
        color: 'blue'
      },
      ignored: {
        title: 'Ignored Tickets',
        description: 'Tickets that are excluded from automatic synchronization',
        icon: EyeOff,
        color: 'purple'
      },
      findings: {
        title: 'Findings Tickets',
        description: 'Display-only tickets in the Findings column',
        icon: AlertTriangle,
        color: 'orange'
      },
      ready_for_stage: {
        title: 'Ready for Stage',
        description: 'Display-only tickets ready for staging',
        icon: CheckCircle,
        color: 'green'
      },
      blocked: {
        title: 'Blocked Tickets',
        description: 'Tickets that are currently blocked',
        icon: Clock,
        color: 'red'
      },
      orphaned: {
        title: 'Orphaned Tickets',
        description: 'YouTrack tickets without corresponding Asana tasks',
        icon: AlertTriangle,
        color: 'gray'
      }
    };
    
    return typeConfig[type] || typeConfig.matched;
  };

  const getDeleteSourceLabel = (source) => {
    switch (source) {
      case 'asana': return 'Asana Only';
      case 'youtrack': return 'YouTrack Only';
      case 'both': return 'Both Systems';
      default: return source;
    }
  };

  const renderTicketCard = (ticket, index) => {
    const ticketId = ticket.gid || ticket.asana_task?.gid || ticket.id || ticket;
    const ticketName = ticket.name || ticket.asana_task?.name || ticket.summary || ticketId;
    const isIgnored = ignoredTickets.has(ticketId);
    const isSelected = selectedTickets.has(ticketId);
    const canBeDeleted = type !== 'ignored';
    
    // Handle ignored tickets (which are just IDs)
    if (type === 'ignored' && typeof ticket === 'string') {
      return (
        <div key={ticket} className="glass-panel border border-gray-200 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="font-medium text-gray-900">Ticket ID: {ticket}</h3>
              <p className="text-sm text-gray-600">Permanently ignored from sync</p>
            </div>
            <button
              onClick={() => handleUnignoreTicket(ticket)}
              disabled={actionLoading[`unignore_${ticket}`]}
              className="bg-green-100 text-green-700 px-3 py-1 rounded hover:bg-green-200 transition-colors disabled:opacity-50 flex items-center"
            >
              {actionLoading[`unignore_${ticket}`] ? (
                <>
                  <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
                  Removing...
                </>
              ) : (
                <>
                  <Eye className="w-3 h-3 mr-1" />
                  Remove from Ignored
                </>
              )}
            </button>
          </div>
        </div>
      );
    }

    return (
      <div 
        key={ticketId} 
        className={`glass-panel border rounded-lg p-4 transition-all ${
          deleteMode 
            ? `cursor-pointer hover:shadow-md ${
                isSelected ? 'border-red-400 bg-red-50 shadow-md' : 'border-gray-200 hover:border-red-200'
              }`
            : 'border-gray-200 hover:shadow-md'
        }`}
        onClick={(e) => canBeDeleted && handleTicketClick(ticketId, index, e)}
      >
        <div className="flex items-start justify-between mb-3">
          <div className="flex-1">
            <div className="flex items-start justify-between">
              <div>
                <h3 className={`font-medium mb-1 ${isSelected ? 'text-red-900' : 'text-gray-900'}`}>
                  {ticketName}
                </h3>
                <p className={`text-sm ${isSelected ? 'text-red-700' : 'text-gray-600'}`}>
                  ID: {ticketId}
                </p>
              </div>
              
              {/* Selection indicator */}
              {deleteMode && canBeDeleted && isSelected && (
                <div className="flex items-center text-red-600">
                  <CheckCircle className="w-4 h-4" />
                </div>
              )}
            </div>
            
            {/* Show section info if available */}
            {ticket.memberships?.[0]?.section?.name && (
              <p className={`text-sm ${isSelected ? 'text-red-600' : 'text-gray-500'}`}>
                Section: {ticket.memberships[0].section.name}
              </p>
            )}
            
            {/* Show status comparison for mismatched tickets */}
            {type === 'mismatched' && (
              <div className="mt-2 space-y-1">
                <div className="flex items-center space-x-2">
                  <span className="status-badge matched text-xs">
                    Asana: {ticket.asana_status}
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <span className="status-badge mismatched text-xs">
                    YouTrack: {ticket.youtrack_status}
                  </span>
                </div>
              </div>
            )}
          </div>
          
          {/* Action buttons - only show when not in delete mode */}
          {!deleteMode && (
            <div className="flex flex-col space-y-2 ml-4">
              {type === 'mismatched' && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleSyncTicket(ticketId);
                  }}
                  disabled={actionLoading[`sync_${ticketId}`]}
                  className="bg-blue-600 text-white px-3 py-1 rounded text-sm hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center"
                >
                  {actionLoading[`sync_${ticketId}`] ? (
                    <>
                      <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
                      Syncing...
                    </>
                  ) : (
                    'Sync'
                  )}
                </button>
              )}
              
              {type === 'missing' && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleCreateTicket(ticketId);
                  }}
                  disabled={actionLoading[`create_${ticketId}`]}
                  className="bg-green-600 text-white px-3 py-1 rounded text-sm hover:bg-green-700 transition-colors disabled:opacity-50 flex items-center"
                >
                  {actionLoading[`create_${ticketId}`] ? (
                    <>
                      <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Plus className="w-3 h-3 mr-1" />
                      Create
                    </>
                  )}
                </button>
              )}
              
              {type !== 'ignored' && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleIgnoreTicket(ticketId);
                  }}
                  disabled={actionLoading[`ignore_${ticketId}`] || isIgnored}
                  className="bg-gray-100 text-gray-700 px-3 py-1 rounded text-sm hover:bg-gray-200 transition-colors disabled:opacity-50 flex items-center"
                >
                  {actionLoading[`ignore_${ticketId}`] ? (
                    <>
                      <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
                      Ignoring...
                    </>
                  ) : isIgnored ? (
                    <>
                      <EyeOff className="w-3 h-3 mr-1" />
                      Ignored!
                    </>
                  ) : (
                    <>
                      <EyeOff className="w-3 h-3 mr-1" />
                      Ignore
                    </>
                  )}
                </button>
              )}
            </div>
          )}
        </div>
        
        {/* Show tags if available */}
        {(ticket.tags || ticket.asana_tags) && (
          <div className="mt-3">
            <div className={`text-xs mb-1 ${isSelected ? 'text-red-600' : 'text-gray-500'}`}>Tags:</div>
            <div className="flex flex-wrap gap-1">
              {(ticket.tags || ticket.asana_tags || []).map((tag, tagIndex) => (
                <span key={tagIndex} className={`tag-glass inline-flex items-center ${isSelected ? 'bg-red-100 text-red-800' : ''}`}>
                  <Tag className="w-3 h-3 mr-1" />
                  {typeof tag === 'string' ? tag : tag.name}
                </span>
              ))}
            </div>
          </div>
        )}
        
        {/* Delete mode instruction */}
        {deleteMode && canBeDeleted && index === 0 && (
          <div className="mt-3 text-xs text-gray-500 border-t pt-2">
            💡 Click to select tickets • Shift+Click for range selection
          </div>
        )}
      </div>
    );
  };

  const typeInfo = getTypeInfo();
  const IconComponent = typeInfo.icon;
  const canDelete = type !== 'ignored';

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex items-center">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" />
          <span>Loading {typeInfo.title.toLowerCase()}...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
            <div className="flex items-center">
              <AlertTriangle className="w-5 h-5 text-red-600 mr-2" />
              <p className="text-red-800">Error loading tickets: {error}</p>
            </div>
            
            {/* <div className="mt-2 text-sm text-red-600">
              Debug: Type={type}, Column={column || 'none'}
            </div> */}
          </div>
        )}

        {/* Delete Panel - Only show when in delete mode with selections */}
        {deleteMode && selectedTickets.size > 0 && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-6 mb-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center">
                <Trash2 className="w-5 h-5 text-red-600 mr-2" />
                <h3 className="text-lg font-semibold text-red-900">
                  Delete Selected ({selectedTickets.size})
                </h3>
              </div>
            </div>
            
            <p className="text-red-700 text-sm mb-4">
              ⚠️ Warning: This action cannot be undone. Please choose carefully where to delete the selected tickets.
            </p>
            
            <div className="flex flex-wrap gap-3">
              <button
                onClick={() => handleDeleteClick('asana')}
                disabled={deleteLoading}
                className="bg-red-500 text-white px-4 py-2 rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50 flex items-center font-medium"
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Delete from Asana Only
              </button>
              
              <button
                onClick={() => handleDeleteClick('youtrack')}
                disabled={deleteLoading}
                className="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 flex items-center font-medium"
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Delete from YouTrack Only
              </button>
              
              <button
                onClick={() => handleDeleteClick('both')}
                disabled={deleteLoading}
                className="bg-red-800 text-white px-4 py-2 rounded-lg hover:bg-red-900 transition-colors disabled:opacity-50 flex items-center font-medium"
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Delete from Both Systems
              </button>
            </div>
            
            <div className="text-xs text-red-600 mt-3">
              Selected tickets: {Array.from(selectedTickets).join(', ')}
            </div>
          </div>
        )}

        {/* Delete Confirmation Modal */}
        {showDeleteConfirm && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
              <div className="flex items-center mb-4">
                <AlertTriangle className="w-6 h-6 text-red-600 mr-2" />
                <h3 className="text-lg font-semibold text-gray-900">Confirm Deletion</h3>
              </div>
              
              <div className="mb-6">
                <p className="text-gray-700 mb-3">
                  You are about to permanently delete <strong>{selectedTickets.size}</strong> tickets from <strong>{getDeleteSourceLabel(deleteSource)}</strong>.
                </p>
                
                <div className="bg-yellow-50 border border-yellow-200 rounded p-3 mb-3">
                  <p className="text-yellow-800 text-sm font-medium">
                    ⚠️ This action cannot be undone!
                  </p>
                </div>
                
                <div className="max-h-32 overflow-y-auto bg-gray-50 rounded p-2">
                  <p className="text-sm font-medium text-gray-700 mb-1">Tickets to be deleted:</p>
                  <div className="text-xs text-gray-600">
                    {Array.from(selectedTickets).map(id => (
                      <div key={id}>• {id}</div>
                    ))}
                  </div>
                </div>
              </div>
              
              <div className="flex space-x-3">
                <button
                  onClick={handleDeleteCancel}
                  disabled={deleteLoading}
                  className="flex-1 bg-gray-200 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-300 transition-colors disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  onClick={handleDeleteConfirm}
                  disabled={deleteLoading}
                  className="flex-1 bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 flex items-center justify-center"
                >
                  {deleteLoading ? (
                    <>
                      <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                      Deleting...
                    </>
                  ) : (
                    <>
                      <Trash2 className="w-4 h-4 mr-2" />
                      Confirm Delete
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        )}

        {tickets.length === 0 ? (
          <div className="text-center py-12">
            <IconComponent className={`w-16 h-16 mx-auto text-${typeInfo.color}-400 mb-4`} />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No {typeInfo.title.toLowerCase()} found
              
            </h3>
            <p className="text-gray-600">
              {type === 'ignored' && 'No tickets are currently being ignored.'}
              {type === 'matched' && column && column !== 'all_syncable' && `All tickets in ${column.replace('_', ' ')} column are either mismatched or missing.`}
              {type === 'matched' && (!column || column === 'all_syncable') && 'All tickets are either mismatched or missing.'}
              {type === 'mismatched' && column && column !== 'all_syncable' && `All tickets in ${column.replace('_', ' ')} column are properly synchronized.`}
              {type === 'mismatched' && (!column || column === 'all_syncable') && 'All tickets are properly synchronized.'}
              {type === 'missing' && column && column !== 'all_syncable' && `All Asana tickets in ${column.replace('_', ' ')} column already exist in YouTrack.`}
              {type === 'missing' && (!column || column === 'all_syncable') && 'All Asana tickets already exist in YouTrack.'}
              {!['ignored', 'matched', 'mismatched', 'missing'].includes(type) && 'No tickets found for this category.'}
            </p>
            {/* NEW: Debug info for troubleshooting */}
            {/* <div className="mt-4 text-xs text-gray-400">
              Debug: Loaded {tickets.length} tickets for type="{type}" column="{column || 'none'}"
            </div> */}
          </div>
        ) : (
          <>
            <div className="mb-6">
              <h2 className="text-2xl font-bold text-gray-900 mb-2">
                {typeInfo.title} ({tickets.length})
                {/* NEW: Show column context */}
                {column && column !== 'all_syncable' && (
                  <span className="text-blue-600 text-lg font-normal ml-2">
                    • {column.replace('_', ' ').toUpperCase()} Column
                  </span>
                )}
                {selectedTickets.size > 0 && (
                  <span className="text-red-600 ml-2">
                    • {selectedTickets.size} selected
                  </span>
                )}
              </h2>
              <p className="text-gray-600">
                {typeInfo.description}
                {column && column !== 'all_syncable' && (
                  <span className="text-blue-600 ml-1">
                    (showing only tickets from {column.replace('_', ' ')} column)
                  </span>
                )}
                {deleteMode && (
                  <span className="text-red-600 ml-2">
                    • Click tickets to select • Shift+Click for range selection
                  </span>
                )}
              </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {tickets.map((ticket, index) => renderTicketCard(ticket, index))}
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TicketDetailView;