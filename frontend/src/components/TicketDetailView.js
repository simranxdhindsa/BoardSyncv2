// frontend/src/components/TicketDetailView.js
import { useState, useEffect, useCallback, useRef } from 'react';
import {
  ArrowLeft, RefreshCw, Tag, EyeOff, Eye, Plus, CheckCircle, Clock,
  AlertTriangle, Trash2, X, Bug, Copy, ExternalLink, Search,
  ChevronDown, ChevronUp, Calendar, User, Zap, FileText, RotateCw,
  AlertCircle, TrendingUp
} from 'lucide-react';
import {
  getTicketsByType, ignoreTicket, unignoreTicket, deleteTickets,
  getEnhancedAnalysis, getChangedMappings,
  syncEnhancedTickets, getAutoSyncDetailed
} from '../services/api';

const TicketDetailView = ({
  type,
  column,
  onBack,
  onSync,
  onCreateSingle,
  onCreateMissing,
  setNavBarSlots,
  onTicketMoved,
  onSilentRefresh
}) => {
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

  // Debug state
  const [showDebug, setShowDebug] = useState(false);
  const [copiedId, setCopiedId] = useState(null);

  // Sort state
  const [sortConfig, setSortConfig] = useState({
    sort_by: 'created_at',
    sort_order: 'desc'
  });

  // Change detection state
  const [changedMappings, setChangedMappings] = useState([]);
  const [showChangesModal, setShowChangesModal] = useState(false);
  const [syncingChanges, setSyncingChanges] = useState(false);

  // Track initial mount to prevent double-loading
  const isInitialMount = useRef(true);


  const getTypeInfo = useCallback(() => {
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
  }, [type]);

  // Load initial data on mount and when dependencies change
  useEffect(() => {
    isInitialMount.current = true;
    loadInitialData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
          <IconComponent className={`w-6 h-6 mr-2 text-${typeInfo.color}-600`} />
          <div>
            <h1 className="text-xl font-semibold text-gray-900">
              {typeInfo.title}

              {column && column !== 'all_syncable' && (
                <span className="text-blue-600 ml-2">
                  • {column.replace('_', ' ').toUpperCase()} Column
                </span>
              )}
              {deleteMode && (
                <span className="text-red-600 ml-2">• Delete Mode</span>
              )}
            </h1>
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
            {/* Changed tickets alert */}
            {changedMappings.length > 0 && (
              <button
                onClick={() => setShowChangesModal(true)}
                className="flex items-center bg-orange-100 text-orange-700 px-4 py-2 rounded-lg hover:bg-orange-200 transition-colors font-medium"
              >
                <AlertCircle className="w-4 h-4 mr-2" />
                {changedMappings.length} Changes Detected
              </button>
            )}

            <button
              onClick={() => setShowDebug(!showDebug)}
              className="flex items-center bg-purple-100 text-purple-700 px-4 py-2 rounded-lg hover:bg-purple-200 transition-colors font-medium"
            >
              <Bug className="w-4 h-4 mr-2" />
              {showDebug ? 'Hide' : 'Show'} Debug
            </button>
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
              onClick={loadInitialData}
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [type, column, deleteMode, selectedTickets, tickets.length, loading, createAllLoading, showDebug, changedMappings, sortConfig, getTypeInfo]);

  useEffect(() => {
    if (!deleteMode) {
      setSelectedTickets(new Set());
      setLastSelectedIndex(-1);
    }
  }, [deleteMode]);

  // Load all initial data
  const loadInitialData = async () => {
    setLoading(true);
    setError(null);

    try {
      // Load changed mappings
      await loadChangedMappings();

      // Load auto-sync details
      await loadAutoSyncDetails();

      // Load tickets
      await loadTickets();

    } catch (err) {
      console.error('Failed to load initial data:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Load changed mappings
  const loadChangedMappings = async () => {
    try {
      const response = await getChangedMappings();
      const changes = response.data?.changed_mappings || response.changed_mappings || [];
      setChangedMappings(changes);
      console.log('Loaded changed mappings:', changes);
    } catch (err) {
      console.error('Failed to load changed mappings:', err);
      setChangedMappings([]);
    }
  };

  // Load auto-sync details
  const loadAutoSyncDetails = async () => {
    try {
      const response = await getAutoSyncDetailed();
      console.log('Loaded auto-sync details:', response);
    } catch (err) {
      console.error('Failed to load auto-sync details:', err);
    }
  };

  // Load tickets - always use simple API, filter/sort on frontend
  const loadTickets = async () => {
    try {
      const columnParam = column && column !== 'all_syncable' ? column : '';

      // Use simple API (original behavior)
      console.log(`Loading tickets with simple API: type=${type}, column=${columnParam}`);
      const response = await getTicketsByType(type, columnParam);
      console.log('Simple API response:', response);

      let ticketData = [];
      if (response.data) {
        ticketData = response.data.tickets || response.data;
      } else if (response.tickets) {
        ticketData = response.tickets;
      } else if (Array.isArray(response)) {
        ticketData = response;
      }

      if (!Array.isArray(ticketData)) {
        console.warn('Ticket data is not an array:', ticketData);
        ticketData = [];
      }

      console.log(`Loaded ${ticketData.length} ${type} tickets`);
      setTickets(ticketData);
      setDeleteMode(false);

    } catch (err) {
      console.error('Failed to load tickets:', err);
      setError(err.message);
      setTickets([]);
    }
  };

  // Get sorted tickets (frontend only)
  const getSortedTickets = () => {
    let sorted = [...tickets];

    // Apply sorting
    sorted.sort((a, b) => {
      let aVal, bVal;

      switch (sortConfig.sort_by) {
        case 'created_at':
          aVal = new Date(a.created_at || a.asana_task?.created_at || 0);
          bVal = new Date(b.created_at || b.asana_task?.created_at || 0);
          break;
        case 'assignee_name':
          aVal = (a.assignee_name || a.asana_task?.assignee?.name || 'Unassigned').toLowerCase();
          bVal = (b.assignee_name || b.asana_task?.assignee?.name || 'Unassigned').toLowerCase();
          break;
        case 'priority':
          const priorityOrder = { 'urgent': 4, 'critical': 4, 'high': 3, 'medium': 2, 'normal': 2, 'low': 1 };
          aVal = priorityOrder[(a.priority || a.asana_task?.priority || '').toLowerCase()] || 0;
          bVal = priorityOrder[(b.priority || b.asana_task?.priority || '').toLowerCase()] || 0;
          break;
        default:
          return 0;
      }

      if (aVal < bVal) return sortConfig.sort_order === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortConfig.sort_order === 'asc' ? 1 : -1;
      return 0;
    });

    return sorted;
  };

  // Toggle sort order
  const handleSortChange = (field) => {
    setSortConfig(prev => ({
      sort_by: field,
      sort_order: prev.sort_by === field && prev.sort_order === 'asc' ? 'desc' : 'asc'
    }));
  };

  // Sync changed tickets
  const handleSyncChanges = async (changeIds = null) => {
    setSyncingChanges(true);
    try {
      const columnParam = column && column !== 'all_syncable' ? column : '';

      const body = changeIds ? { change_ids: changeIds } : {};

      console.log('Syncing changes:', { column: columnParam, body });
      await syncEnhancedTickets(columnParam, body);

      // Refresh data
      await loadChangedMappings();
      await loadTickets();

      setShowChangesModal(false);

      alert('Changes synced successfully!');

    } catch (err) {
      console.error('Failed to sync changes:', err);
      alert('Failed to sync changes: ' + err.message);
    } finally {
      setSyncingChanges(false);
    }
  };

  // Remove ticket from local state
  const removeTicketFromView = (ticketId) => {
    setTickets(prev => prev.filter(ticket => {
      const currentTicketId = ticket.gid || ticket.asana_task?.gid || ticket.id || ticket;
      return currentTicketId !== ticketId;
    }));
  };

  // CREATE ALL
  const handleCreateAll = async () => {
    if (!onCreateMissing || tickets.length === 0) return;

    setCreateAllLoading(true);
    try {
      const ticketsCopy = [...tickets];

      await onCreateMissing();

      setTickets([]);

      ticketsCopy.forEach(ticket => {
        if (onTicketMoved) {
          onTicketMoved(ticket.gid, 'missing');
        }
      });

      if (onSilentRefresh) {
        setTimeout(() => onSilentRefresh(), 3000);
      }
    } catch (err) {
      console.error('Failed to create all tickets:', err);
      alert('Failed to create all tickets: ' + err.message);
    } finally {
      setCreateAllLoading(false);
    }
  };

  // IGNORE TICKET
  const handleIgnoreTicket = async (ticketId) => {
    setActionLoading(prev => ({ ...prev, [`ignore_${ticketId}`]: true }));
    try {
      await ignoreTicket(ticketId);

      setIgnoredTickets(prev => new Set([...prev, ticketId]));

      setTimeout(() => {
        removeTicketFromView(ticketId);
      }, 1000);
    } catch (err) {
      console.error('Failed to ignore ticket:', err);
      alert('Failed to ignore ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`ignore_${ticketId}`]: false }));
    }
  };

  // UNIGNORE TICKET
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
        removeTicketFromView(ticketId);
      }
    } catch (err) {
      console.error('Failed to unignore ticket:', err);
      alert('Failed to unignore ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`unignore_${ticketId}`]: false }));
    }
  };

  // SYNC TICKET
  const handleSyncTicket = async (ticketId) => {
    setActionLoading(prev => ({ ...prev, [`sync_${ticketId}`]: true }));
    try {
      await onSync(ticketId);

      removeTicketFromView(ticketId);

      if (onTicketMoved) {
        onTicketMoved(ticketId, 'mismatched');
      }

      if (onSilentRefresh) {
        setTimeout(() => onSilentRefresh(), 3000);
      }
    } catch (err) {
      console.error('Failed to sync ticket:', err);
      alert('Failed to sync ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`sync_${ticketId}`]: false }));
    }
  };

  // CREATE TICKET
  const handleCreateTicket = async (taskId) => {
    setActionLoading(prev => ({ ...prev, [`create_${taskId}`]: true }));
    try {
      await onCreateSingle(taskId);

      removeTicketFromView(taskId);

      if (onTicketMoved) {
        onTicketMoved(taskId, 'missing');
      }

      if (onSilentRefresh) {
        setTimeout(() => onSilentRefresh(), 3000);
      }
    } catch (err) {
      console.error('Failed to create ticket:', err);
      alert('Failed to create ticket: ' + err.message);
    } finally {
      setActionLoading(prev => ({ ...prev, [`create_${taskId}`]: false }));
    }
  };

  const handleTicketClick = (ticketId, index, event) => {
    if (!deleteMode) return;

    event.preventDefault();

    if (event.shiftKey) {
      event.preventDefault();
      window.getSelection().removeAllRanges();
    }

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

  const handleDeleteClick = (source) => {
    setDeleteSource(source);
    setShowDeleteConfirm(true);
  };

  // DELETE TICKETS
  const handleDeleteConfirm = async () => {
    setDeleteLoading(true);
    try {
      const ticketIds = Array.from(selectedTickets);

      const response = await deleteTickets(ticketIds, deleteSource);

      setTickets(prev => prev.filter(ticket => {
        const ticketId = ticket.gid || ticket.asana_task?.gid || ticket.id || ticket;
        return !selectedTickets.has(ticketId);
      }));

      setShowDeleteConfirm(false);
      setDeleteMode(false);
      setSelectedTickets(new Set());

      const successCount = response.success_count || 0;
      const failureCount = response.failure_count || 0;
      const summary = response.summary || `Processed ${ticketIds.length} tickets`;

      setTimeout(() => {
        alert(`Delete Operation Complete:\n${summary}\n\nSuccessful: ${successCount}\nFailed: ${failureCount}`);
      }, 100);

      if (onSilentRefresh) {
        setTimeout(() => onSilentRefresh(), 3000);
      }

    } catch (err) {
      console.error('Delete operation failed:', err);
      alert('Delete operation failed: ' + err.message);
    } finally {
      setDeleteLoading(false);
      setDeleteSource('');
    }
  };

  const handleDeleteCancel = () => {
    setShowDeleteConfirm(false);
    setDeleteSource('');
  };

  const handleCopyTicketTitle = (ticketName, ticketId) => {
    navigator.clipboard.writeText(ticketName).then(() => {
      setCopiedId(ticketId);
      setTimeout(() => setCopiedId(null), 2000);
    }).catch(err => {
      console.error('Failed to copy:', err);
    });
  };

  const handleOpenAsanaLink = (ticketId) => {
    const asanaUrl = `https://app.asana.com/0/0/${ticketId}`;
    window.open(asanaUrl, '_blank');
  };

  const handleOpenYouTrackSearch = (ticketName) => {
    const encodedQuery = encodeURIComponent(ticketName);
    const youtrackUrl = `https://loop.youtrack.cloud/agiles/183-4/current?query=${encodedQuery}`;
    window.open(youtrackUrl, '_blank');
  };

  // Format date
  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    try {
      const date = new Date(dateString);
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    } catch (err) {
      return 'Invalid Date';
    }
  };

  // Get priority badge color
  const getPriorityColor = (priority) => {
    if (!priority) return 'bg-gray-100 text-gray-700';
    const p = priority.toLowerCase();
    if (p === 'urgent' || p === 'critical') return 'bg-red-100 text-red-700';
    if (p === 'high') return 'bg-orange-100 text-orange-700';
    if (p === 'medium' || p === 'normal') return 'bg-yellow-100 text-yellow-700';
    if (p === 'low') return 'bg-green-100 text-green-700';
    return 'bg-gray-100 text-gray-700';
  };

  // Get change indicators
  const getChangeIndicators = (ticketId) => {
    const change = changedMappings.find(c => c.asana_task_id === ticketId);
    if (!change) return null;

    const indicators = [];
    if (change.title_changed) indicators.push({ icon: FileText, label: 'Title', color: 'text-blue-600' });
    if (change.description_changed) indicators.push({ icon: FileText, label: 'Description', color: 'text-purple-600' });
    if (change.status_mismatch) indicators.push({ icon: RotateCw, label: 'Status', color: 'text-orange-600' });

    return indicators.length > 0 ? indicators : null;
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
    const ticketId = ticket.gid ||
                    ticket.asana_task?.gid ||
                    ticket.youtrack_issue?.id ||
                    ticket.id ||
                    ticket;

    const ticketName = ticket.name ||
                      ticket.asana_task?.name ||
                      ticket.youtrack_issue?.summary ||
                      ticket.summary ||
                      ticketId;

    const isIgnored = ignoredTickets.has(ticketId);
    const isSelected = selectedTickets.has(ticketId);
    const canBeDeleted = type !== 'ignored';
    const isCopied = copiedId === ticketId;

    // Extract enhanced fields
    const createdDate = ticket.created_at || ticket.asana_task?.created_at;
    const assignee = ticket.assignee_name || ticket.asana_task?.assignee?.name || 'Unassigned';
    const priority = ticket.priority || ticket.asana_task?.priority;
    const changeIndicators = getChangeIndicators(ticketId);

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

    const ticketTags = ticket.tags ||
                      ticket.asana_tags ||
                      ticket.asana_task?.tags ||
                      [];

    const ticketSection = ticket.memberships?.[0]?.section?.name ||
                         ticket.asana_task?.memberships?.[0]?.section?.name ||
                         'No Section';

    return (
      <div
        key={ticketId}
        className={`glass-panel border rounded-lg p-4 transition-all ${
          deleteMode
            ? `cursor-pointer hover:shadow-md select-none ${
                isSelected ? 'border-red-400 bg-red-50 shadow-md' : 'border-gray-200 hover:border-red-200'
              }`
            : 'border-gray-200 hover:shadow-md'
        }`}
        onClick={(e) => canBeDeleted && handleTicketClick(ticketId, index, e)}
        onMouseDown={(e) => {
          if (deleteMode && canBeDeleted) {
            e.preventDefault();
          }
        }}
      >
        <div className="flex items-start justify-between mb-3">
          <div className="flex-1">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-2">
                  <h3 className={`font-medium ${isSelected ? 'text-red-900' : 'text-gray-900'}`}>
                    {ticketName}
                  </h3>
                </div>

                {/* Enhanced metadata row */}
                <div className="flex items-center gap-3 mb-2 text-sm text-gray-600">
                  <div className="flex items-center gap-1">
                    <Calendar className="w-3 h-3" />
                    <span>{formatDate(createdDate)}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <User className="w-3 h-3" />
                    <span>{assignee}</span>
                  </div>
                  {priority && (
                    <div className="flex items-center gap-1">
                      <Zap className="w-3 h-3" />
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${getPriorityColor(priority)}`}>
                        {priority}
                      </span>
                    </div>
                  )}
                </div>

                {/* Change indicators */}
                {changeIndicators && changeIndicators.length > 0 && (
                  <div className="flex items-center gap-2 mb-2">
                    {changeIndicators.map((indicator, idx) => {
                      const IconComponent = indicator.icon;
                      return (
                        <div
                          key={idx}
                          className="flex items-center gap-1 px-2 py-1 bg-orange-50 border border-orange-200 rounded text-xs"
                          title={`${indicator.label} changed`}
                        >
                          <IconComponent className={`w-3 h-3 ${indicator.color}`} />
                          <span className="text-orange-700">{indicator.label}</span>
                        </div>
                      );
                    })}
                  </div>
                )}

                <p className={`text-sm ${isSelected ? 'text-red-700' : 'text-gray-600'}`}>
                  ID: {ticketId}
                </p>
              </div>

              {deleteMode && canBeDeleted && isSelected && (
                <div className="flex items-center text-red-600">
                  <CheckCircle className="w-4 h-4" />
                </div>
              )}
            </div>

            <p className={`text-sm ${isSelected ? 'text-red-600' : 'text-gray-500'}`}>
              Section: {ticketSection}
            </p>

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

              <div className="flex items-center gap-1">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleCopyTicketTitle(ticketName, ticketId);
                  }}
                  className={`p-1 rounded hover:bg-gray-200 transition-colors ${isCopied ? 'bg-green-100' : ''}`}
                  title="Copy ticket title"
                >
                  <Copy className={`w-3 h-3 ${isCopied ? 'text-green-600' : 'text-gray-600'}`} />
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleOpenYouTrackSearch(ticketName);
                  }}
                  className="p-1 rounded hover:bg-gray-200 transition-colors"
                  title="Search in YouTrack"
                >
                  <Search className="w-3 h-3 text-orange-600" />
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleOpenAsanaLink(ticketId);
                  }}
                  className="p-1 rounded hover:bg-gray-200 transition-colors"
                  title="Open in Asana"
                >
                  <ExternalLink className="w-3 h-3 text-blue-600" />
                </button>
              </div>
            </div>
          )}
        </div>

        {ticketTags && ticketTags.length > 0 && (
          <div className="mt-3">
            <div className={`text-xs mb-1 ${isSelected ? 'text-red-600' : 'text-gray-500'}`}>Tags:</div>
            <div className="flex flex-wrap gap-1">
              {ticketTags.map((tag, tagIndex) => (
                <span key={tagIndex} className={`tag-glass inline-flex items-center ${isSelected ? 'bg-red-100 text-red-800' : ''}`}>
                  <Tag className="w-3 h-3 mr-1" />
                  {typeof tag === 'string' ? tag : tag.name}
                </span>
              ))}
            </div>
          </div>
        )}

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
      <div
        className={`max-w-6xl mx-auto px-6 py-8 ${deleteMode ? 'select-none' : ''}`}
        style={deleteMode ? { userSelect: 'none', WebkitUserSelect: 'none', MozUserSelect: 'none', msUserSelect: 'none' } : {}}
      >
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
            <div className="flex items-center">
              <AlertTriangle className="w-5 h-5 text-red-600 mr-2" />
              <p className="text-red-800">Error loading tickets: {error}</p>
            </div>
          </div>
        )}

        {/* Change Alert Banner */}
        {changedMappings.length > 0 && (
          <div className="bg-orange-50 border border-orange-200 rounded-lg p-4 mb-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <AlertCircle className="w-5 h-5 text-orange-600 mr-2" />
                <div>
                  <h3 className="text-lg font-semibold text-orange-900">
                    {changedMappings.length} Ticket{changedMappings.length !== 1 ? 's' : ''} with Changes Detected
                  </h3>
                  <p className="text-sm text-orange-700">
                    Titles, descriptions, or statuses have changed in Asana
                  </p>
                </div>
              </div>
              <button
                onClick={() => setShowChangesModal(true)}
                className="flex items-center bg-orange-600 text-white px-4 py-2 rounded-lg hover:bg-orange-700 transition-colors font-medium"
              >
                <Eye className="w-4 h-4 mr-2" />
                View Details
              </button>
            </div>
          </div>
        )}

        {/* Sort Controls */}
        <div className="flex items-center justify-between mb-6 gap-4">
          <div className="flex items-center gap-4">
            <h2 className="text-2xl font-bold text-gray-900">
              {typeInfo.title} ({tickets.length})
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
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            {/* Sort */}
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-700">Sort:</label>
              <select
                value={sortConfig.sort_by}
                onChange={(e) => handleSortChange(e.target.value)}
                className="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500"
              >
                <option value="created_at">Created Date</option>
                <option value="assignee_name">Assignee</option>
                <option value="priority">Priority</option>
              </select>
              <button
                onClick={() => setSortConfig(prev => ({
                  ...prev,
                  sort_order: prev.sort_order === 'asc' ? 'desc' : 'asc'
                }))}
                className="p-1.5 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                title={sortConfig.sort_order === 'asc' ? 'Ascending' : 'Descending'}
              >
                {sortConfig.sort_order === 'asc' ? (
                  <ChevronUp className="w-4 h-4" />
                ) : (
                  <ChevronDown className="w-4 h-4" />
                )}
              </button>
            </div>
          </div>
        </div>

        {/* Debug Panel */}
        {showDebug && tickets.length > 0 && (
          <div className="bg-purple-50 border border-purple-200 rounded-lg p-6 mb-6">
            <div className="flex items-center mb-4">
              <Bug className="w-5 h-5 text-purple-600 mr-2" />
              <h3 className="text-lg font-semibold text-purple-900">
                Debug View - Ticket Titles ({tickets.length})
              </h3>
            </div>

            <div className="max-h-96 overflow-y-auto bg-white rounded-lg p-4 space-y-2">
              {tickets.map((ticket, index) => {
                const ticketId = ticket.gid ||
                                ticket.asana_task?.gid ||
                                ticket.youtrack_issue?.id ||
                                ticket.id ||
                                ticket;

                const ticketName = ticket.name ||
                                  ticket.asana_task?.name ||
                                  ticket.youtrack_issue?.summary ||
                                  ticket.summary ||
                                  ticketId;

                const isCopied = copiedId === ticketId;

                return (
                  <div
                    key={ticketId}
                    className="flex items-center justify-between p-3 bg-purple-50 rounded-lg hover:bg-purple-100 transition-colors border border-purple-200"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-purple-900">
                          {index + 1}.
                        </span>
                        <span className="text-sm text-gray-900 truncate">
                          {ticketName}
                        </span>
                      </div>
                      <div className="text-xs text-gray-600 mt-1">
                        ID: {ticketId}
                      </div>
                    </div>

                    <div className="flex items-center gap-2 ml-4">
                      <button
                        onClick={() => handleCopyTicketTitle(ticketName, ticketId)}
                        className={`p-2 rounded-lg transition-colors ${
                          isCopied
                            ? 'bg-green-500 text-white'
                            : 'bg-purple-200 text-purple-700 hover:bg-purple-300'
                        }`}
                        title={isCopied ? 'Copied!' : 'Copy ticket title'}
                      >
                        <Copy className="w-4 h-4" />
                      </button>

                      <button
                        onClick={() => handleOpenYouTrackSearch(ticketName)}
                        className="p-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 transition-colors"
                        title="Search in YouTrack"
                      >
                        <Search className="w-4 h-4" />
                      </button>

                      <button
                        onClick={() => handleOpenAsanaLink(ticketId)}
                        className="p-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
                        title="Open in Asana"
                      >
                        <ExternalLink className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="mt-4 text-sm text-purple-700">
              💡 Click copy icon to copy ticket title • Click search to find in YouTrack • Click external link to open in Asana
            </div>
          </div>
        )}

        {/* Delete Panel */}
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
              Selected tickets: {Array.from(selectedTickets).slice(0, 10).join(', ')}
              {selectedTickets.size > 10 && ` ... and ${selectedTickets.size - 10} more`}
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

        {/* Changes Comparison Modal */}
        {showChangesModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 overflow-y-auto p-4">
            <div className="bg-white rounded-lg max-w-4xl w-full my-8 max-h-[90vh] overflow-y-auto">
              <div className="sticky top-0 bg-white border-b border-gray-200 p-6 z-10">
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <TrendingUp className="w-6 h-6 text-orange-600 mr-2" />
                    <h2 className="text-2xl font-bold text-gray-900">
                      Changed Tickets ({changedMappings.length})
                    </h2>
                  </div>
                  <button
                    onClick={() => setShowChangesModal(false)}
                    className="text-gray-500 hover:text-gray-700"
                  >
                    <X className="w-6 h-6" />
                  </button>
                </div>

                <p className="text-gray-600 mt-2">
                  Review and sync tickets where Asana data has been updated
                </p>
              </div>

              <div className="p-6 space-y-4">
                {changedMappings.map((change, index) => (
                  <div key={index} className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
                    <div className="flex items-start justify-between mb-4">
                      <div>
                        <h3 className="font-semibold text-gray-900 mb-1">
                          Ticket ID: {change.asana_task_id}
                        </h3>
                        <div className="flex items-center gap-2">
                          {change.title_changed && (
                            <span className="px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded flex items-center gap-1">
                              <FileText className="w-3 h-3" />
                              Title
                            </span>
                          )}
                          {change.description_changed && (
                            <span className="px-2 py-1 bg-purple-100 text-purple-700 text-xs rounded flex items-center gap-1">
                              <FileText className="w-3 h-3" />
                              Description
                            </span>
                          )}
                          {change.status_mismatch && (
                            <span className="px-2 py-1 bg-orange-100 text-orange-700 text-xs rounded flex items-center gap-1">
                              <RotateCw className="w-3 h-3" />
                              Status
                            </span>
                          )}
                        </div>
                      </div>

                      <button
                        onClick={() => handleSyncChanges([change.asana_task_id])}
                        disabled={syncingChanges}
                        className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center whitespace-nowrap"
                      >
                        {syncingChanges ? (
                          <>
                            <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                            Syncing...
                          </>
                        ) : (
                          <>
                            <RotateCw className="w-4 h-4 mr-2" />
                            Sync This
                          </>
                        )}
                      </button>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {/* Old YouTrack Data */}
                      <div className="bg-gray-50 rounded p-3">
                        <h4 className="text-sm font-semibold text-gray-700 mb-2">
                          Old (YouTrack)
                        </h4>
                        <div className="space-y-2 text-sm">
                          {change.title_changed && (
                            <div>
                              <span className="text-gray-600 font-medium">Title:</span>
                              <p className="text-gray-900 mt-1">{change.old_title || 'N/A'}</p>
                            </div>
                          )}
                          {change.description_changed && (
                            <div>
                              <span className="text-gray-600 font-medium">Description:</span>
                              <p className="text-gray-900 mt-1 line-clamp-3">{change.old_description || 'No description'}</p>
                            </div>
                          )}
                          {change.status_mismatch && (
                            <div>
                              <span className="text-gray-600 font-medium">Status:</span>
                              <p className="text-gray-900 mt-1">{change.youtrack_status || 'N/A'}</p>
                            </div>
                          )}
                        </div>
                      </div>

                      {/* New Asana Data */}
                      <div className="bg-blue-50 rounded p-3">
                        <h4 className="text-sm font-semibold text-blue-700 mb-2">
                          New (Asana)
                        </h4>
                        <div className="space-y-2 text-sm">
                          {change.title_changed && (
                            <div>
                              <span className="text-gray-600 font-medium">Title:</span>
                              <p className="text-gray-900 mt-1 font-medium">{change.new_title || 'N/A'}</p>
                            </div>
                          )}
                          {change.description_changed && (
                            <div>
                              <span className="text-gray-600 font-medium">Description:</span>
                              <p className="text-gray-900 mt-1 line-clamp-3 font-medium">{change.new_description || 'No description'}</p>
                            </div>
                          )}
                          {change.status_mismatch && (
                            <div>
                              <span className="text-gray-600 font-medium">Status:</span>
                              <p className="text-gray-900 mt-1 font-medium">{change.asana_status || 'N/A'}</p>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              <div className="sticky bottom-0 bg-white border-t border-gray-200 p-6">
                <div className="flex justify-between items-center">
                  <p className="text-gray-600">
                    {changedMappings.length} ticket{changedMappings.length !== 1 ? 's' : ''} with changes
                  </p>
                  <div className="flex gap-3">
                    <button
                      onClick={() => setShowChangesModal(false)}
                      className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
                    >
                      Close
                    </button>
                    <button
                      onClick={() => handleSyncChanges()}
                      disabled={syncingChanges}
                      className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center font-medium"
                    >
                      {syncingChanges ? (
                        <>
                          <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                          Syncing All...
                        </>
                      ) : (
                        <>
                          <RotateCw className="w-4 h-4 mr-2" />
                          Sync All Changes
                        </>
                      )}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Empty State or Tickets List */}
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
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {getSortedTickets().map((ticket, index) => renderTicketCard(ticket, index))}
          </div>
        )}
      </div>
    </div>
  );
};

export default TicketDetailView;
