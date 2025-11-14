import React, { useState, useEffect } from 'react';
import { RefreshCw, Zap, Activity, Play, Square, Clock, Inbox, Loader, Code, Package, AlertCircle, CheckCircle, Search, Layers, Edit } from 'lucide-react';
import FluidText from './FluidText';
import '../styles/dashboard-glass.css';
import {
  getAutoSyncStatus,
  startAutoSync,
  stopAutoSync,
  getAutoCreateStatus,
  startAutoCreate,
  stopAutoCreate,
  getReverseAutoCreateStatus,
  startReverseAutoCreate,
  stopReverseAutoCreate,
  getYouTrackUsers,
  getUserSettings
} from '../services/api';

const Dashboard = ({ selectedColumn, onColumnSelect, onAnalyze, loading }) => {
  const [autoSyncRunning, setAutoSyncRunning] = useState(false);
  const [autoCreateRunning, setAutoCreateRunning] = useState(false);
  const [reverseAutoCreateRunning, setReverseAutoCreateRunning] = useState(false);
  const [autoSyncInterval, setAutoSyncInterval] = useState(15);
  const [autoCreateInterval, setAutoCreateInterval] = useState(15);
  const [showIntervalModal, setShowIntervalModal] = useState(null); // 'sync' or 'create'
  const [tempIntervalValue, setTempIntervalValue] = useState(15);
  const [tempIntervalUnit, setTempIntervalUnit] = useState('seconds'); // 'seconds', 'minutes', 'hours'
  const [autoSyncLastInfo, setAutoSyncLastInfo] = useState('');
  const [autoCreateLastInfo, setAutoCreateLastInfo] = useState('');
  const [reverseAutoCreateLastInfo, setReverseAutoCreateLastInfo] = useState('');
  const [toggleLoading, setToggleLoading] = useState({ sync: false, create: false, reverseCreate: false });
  const [columns, setColumns] = useState([]);
  const [columnsLoading, setColumnsLoading] = useState(true);
  const [showUserSelectionModal, setShowUserSelectionModal] = useState(false);
  const [youtrackUsers, setYoutrackUsers] = useState([]);
  const [selectedCreators, setSelectedCreators] = useState('All');
  const [tempSelectedCreators, setTempSelectedCreators] = useState([]);
  const [usersLoading, setUsersLoading] = useState(false);

  const selectedColumnData = columns.find(col => col.value === selectedColumn);

  // Function to assign icons based on column name
  const getIconForColumn = (columnName) => {
    const lowerName = columnName.toLowerCase();

    if (lowerName.includes('backlog')) return Inbox;
    if (lowerName.includes('progress')) return Loader;
    if (lowerName.includes('dev')) return Code;
    if (lowerName.includes('stage')) return Package;
    if (lowerName.includes('blocked')) return AlertCircle;
    if (lowerName.includes('ready')) return CheckCircle;
    if (lowerName.includes('finding')) return Search;
    if (lowerName.includes('all')) return Layers;

    // Default icon
    return Activity;
  };

  // Load columns from user's column mappings
  useEffect(() => {
    loadColumnsFromSettings();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadColumnsFromSettings = async () => {
    setColumnsLoading(true);
    try {
      const response = await getUserSettings();
      const settings = response.data || response;

      // Generate columns from user's column mappings
      const mappedColumns = [];

      if (settings.column_mappings?.asana_to_youtrack) {
        settings.column_mappings.asana_to_youtrack.forEach(mapping => {
          const columnValue = mapping.asana_column.toLowerCase().replace(/\s+/g, '_');
          mappedColumns.push({
            value: columnValue,
            label: mapping.asana_column,
            color: 'hover:bg-blue-50 hover:border-blue-200',
            displayOnly: mapping.display_only,
            icon: getIconForColumn(mapping.asana_column)
          });
        });
      }

      // Always add "All Syncable" option at the end (only if there are mapped columns)
      if (mappedColumns.length > 0) {
        mappedColumns.push({
          value: 'all_syncable',
          label: 'All Syncable',
          color: 'hover:bg-blue-50 hover:border-blue-200',
          icon: Layers
        });
      }

      setColumns(mappedColumns);
    } catch (err) {
      console.error('Failed to load column mappings:', err);
      // Fallback to empty array if settings not configured
      setColumns([]);
    } finally {
      setColumnsLoading(false);
    }
  };

  // Load auto-sync and auto-create status on mount
  useEffect(() => {
    loadAutoStatus();
    
    // Refresh status every 30 seconds
    const interval = setInterval(loadAutoStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadAutoStatus = async () => {
    try {
      console.log('Loading auto status...'); // DEBUG

      const [syncStatus, createStatus, reverseCreateStatus] = await Promise.all([
        getAutoSyncStatus(),
        getAutoCreateStatus(),
        getReverseAutoCreateStatus()
      ]);

      console.log('Auto-sync status:', syncStatus); // DEBUG
      console.log('Auto-create status:', createStatus); // DEBUG
      console.log('Reverse auto-create status:', reverseCreateStatus); // DEBUG

      if (syncStatus.auto_sync) {
        setAutoSyncRunning(syncStatus.auto_sync.running);
        setAutoSyncInterval(syncStatus.auto_sync.interval);
        setAutoSyncLastInfo(syncStatus.auto_sync.last_info || '');
      }

      if (createStatus.auto_create) {
        setAutoCreateRunning(createStatus.auto_create.running);
        setAutoCreateInterval(createStatus.auto_create.interval);
        setAutoCreateLastInfo(createStatus.auto_create.last_info || '');
      }

      if (reverseCreateStatus.reverse_auto_create) {
        setReverseAutoCreateRunning(reverseCreateStatus.reverse_auto_create.running);
        setReverseAutoCreateLastInfo(reverseCreateStatus.reverse_auto_create.last_info || '');
      }
    } catch (error) {
      console.error('Failed to load auto status:', error);
      // REMOVED: alert() calls - just log to console
    }
  };

  const handleAutoSyncToggle = async () => {
    setToggleLoading(prev => ({ ...prev, sync: true }));
    try {
      console.log('Toggling auto-sync...', autoSyncRunning ? 'STOP' : 'START'); // DEBUG
      
      if (autoSyncRunning) {
        await stopAutoSync();
        setAutoSyncRunning(false);
      } else {
        await startAutoSync(autoSyncInterval);
        setAutoSyncRunning(true);
      }
    } catch (error) {
      console.error('Auto-sync toggle failed:', error);
      // REMOVED: alert() call - just log to console
    } finally {
      setToggleLoading(prev => ({ ...prev, sync: false }));
    }
  };

  const handleAutoCreateToggle = async () => {
    setToggleLoading(prev => ({ ...prev, create: true }));
    try {
      console.log('Toggling auto-create...', autoCreateRunning ? 'STOP' : 'START'); // DEBUG
      console.log('API Base URL:', process.env.NODE_ENV === 'production' ? process.env.REACT_APP_API_URL || 'https://boardsyncapi.onrender.com' : 'http://localhost:8080'); // DEBUG

      if (autoCreateRunning) {
        const result = await stopAutoCreate();
        console.log('Stop auto-create result:', result); // DEBUG
        setAutoCreateRunning(false);
      } else {
        const result = await startAutoCreate(autoCreateInterval);
        console.log('Start auto-create result:', result); // DEBUG
        setAutoCreateRunning(true);
      }
    } catch (error) {
      console.error('Auto-create toggle failed:', error);
      console.error('Error details:', error); // DEBUG
      // REMOVED: alert() call - just log to console
    } finally {
      setToggleLoading(prev => ({ ...prev, create: false }));
    }
  };

  const handleReverseAutoCreateToggle = async () => {
    setToggleLoading(prev => ({ ...prev, reverseCreate: true }));
    try {
      console.log('Toggling reverse auto-create...', reverseAutoCreateRunning ? 'STOP' : 'START'); // DEBUG

      if (reverseAutoCreateRunning) {
        const result = await stopReverseAutoCreate();
        console.log('Stop reverse auto-create result:', result); // DEBUG
        setReverseAutoCreateRunning(false);
      } else {
        // Use same interval as auto-create
        const result = await startReverseAutoCreate(autoCreateInterval, selectedCreators);
        console.log('Start reverse auto-create result:', result); // DEBUG
        setReverseAutoCreateRunning(true);
      }
    } catch (error) {
      console.error('Reverse auto-create toggle failed:', error);
      console.error('Error details:', error); // DEBUG
      // REMOVED: alert() call - just log to console
    } finally {
      setToggleLoading(prev => ({ ...prev, reverseCreate: false }));
    }
  };

  // User selection modal handlers
  const handleOpenUserSelectionModal = async () => {
    setShowUserSelectionModal(true);
    setUsersLoading(true);
    try {
      const users = await getYouTrackUsers();
      console.log('YouTrack users received:', users); // DEBUG
      setYoutrackUsers(users);

      // Initialize temp selection based on current selection
      if (selectedCreators === 'All') {
        setTempSelectedCreators([]);
      } else {
        setTempSelectedCreators(selectedCreators.split(','));
      }
    } catch (error) {
      console.error('Failed to load YouTrack users:', error);
    } finally {
      setUsersLoading(false);
    }
  };

  const handleCloseUserSelectionModal = () => {
    setShowUserSelectionModal(false);
    setTempSelectedCreators([]);
  };

  const handleToggleUser = (userName) => {
    setTempSelectedCreators(prev => {
      if (prev.includes(userName)) {
        return prev.filter(u => u !== userName);
      } else {
        return [...prev, userName];
      }
    });
  };

  const handleSelectAllUsers = () => {
    setTempSelectedCreators([]);
  };

  const handleSaveUserSelection = () => {
    if (tempSelectedCreators.length === 0) {
      setSelectedCreators('All');
    } else {
      setSelectedCreators(tempSelectedCreators.join(','));
    }
    handleCloseUserSelectionModal();
  };

  // Interval editing handlers
  const handleOpenIntervalModal = (type) => {
    const currentInterval = type === 'sync' ? autoSyncInterval : autoCreateInterval;
    setShowIntervalModal(type);
    setTempIntervalValue(currentInterval);
    setTempIntervalUnit('seconds');
  };

  const handleCloseIntervalModal = () => {
    setShowIntervalModal(null);
    setTempIntervalValue(15);
    setTempIntervalUnit('seconds');
  };

  const handleSaveInterval = async () => {
    let intervalInSeconds = tempIntervalValue;
    if (tempIntervalUnit === 'minutes') {
      intervalInSeconds = tempIntervalValue * 60;
    } else if (tempIntervalUnit === 'hours') {
      intervalInSeconds = tempIntervalValue * 3600;
    }

    if (intervalInSeconds < 15) {
      return;
    }

    if (showIntervalModal === 'sync') {
      setAutoSyncInterval(intervalInSeconds);
      if (autoSyncRunning) {
        try {
          await stopAutoSync();
          await startAutoSync(intervalInSeconds);
        } catch (error) {
          console.error('Failed to update sync interval:', error);
        }
      }
    } else if (showIntervalModal === 'create') {
      setAutoCreateInterval(intervalInSeconds);
      if (autoCreateRunning) {
        try {
          await stopAutoCreate();
          await startAutoCreate(intervalInSeconds);
        } catch (error) {
          console.error('Failed to update create interval:', error);
        }
      }
      // Also update reverse create if running (shares same interval)
      if (reverseAutoCreateRunning) {
        try {
          await stopReverseAutoCreate();
          await startReverseAutoCreate(intervalInSeconds, selectedCreators);
        } catch (error) {
          console.error('Failed to update reverse create interval:', error);
        }
      }
    }

    handleCloseIntervalModal();
  };

  const formatInterval = (seconds) => {
    if (seconds < 60) {
      return seconds + ' seconds';
    } else if (seconds < 3600) {
      return Math.floor(seconds / 60) + ' minutes';
    } else {
      return Math.floor(seconds / 3600) + ' hours';
    }
  };
  return (
    <div>
      {/* Main Content */}
      <div className="pt-4 pb-8">
        {/* Header Section with Fluid Text */}
        <div className="mb-8">
          <FluidText className="text-3xl font-bold text-gray-900 mb-2 block" sensitivity={2}>
            Pick a column and let's see how badly these two systems disagree with each other
          </FluidText>
        </div>

        {/* NEW: Auto Controls Section */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          {/* Auto-Sync Control */}
          <div className="glass-panel border border-gray-200 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center">
                <RefreshCw className={`w-5 h-5 mr-2 ${autoSyncRunning ? 'text-green-600 animate-spin' : 'text-gray-600'}`} />
                <h3 className="text-lg font-semibold text-gray-900">Auto-Sync</h3>
                <span className={`ml-2 px-2 py-1 rounded-full text-xs font-medium ${
                  autoSyncRunning ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'
                }`}>
                  {autoSyncRunning ? 'RUNNING' : 'STOPPED'}
                </span>
              </div>

              <button
                onClick={handleAutoSyncToggle}
                disabled={toggleLoading.sync}
                className={`flex items-center px-4 py-2 rounded-lg font-medium transition-colors ${
                  autoSyncRunning
                    ? 'bg-red-100 text-red-700 hover:bg-red-200'
                    : 'bg-green-100 text-green-700 hover:bg-green-200'
                } disabled:opacity-50`}
              >
                {toggleLoading.sync ? (
                  <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                ) : autoSyncRunning ? (
                  <Square className="w-4 h-4 mr-2" />
                ) : (
                  <Play className="w-4 h-4 mr-2" />
                )}
                {autoSyncRunning ? 'Stop' : 'Start'}
              </button>
            </div>

            <div className="space-y-2 text-sm text-gray-600">
              <div className="flex items-center">
                <Clock className="w-4 h-4 mr-2" />
                <span>Every {formatInterval(autoSyncInterval)}</span>
                <button
                  onClick={() => handleOpenIntervalModal('sync')}
                  className="interval-edit-button"
                  title="Edit interval"
                >
                  <Edit className="interval-edit-icon" />
                </button>
              </div>
              {autoSyncRunning && (
                <>
                  {autoSyncLastInfo && (
                    <div className="text-xs bg-gray-50 rounded p-2 mt-2">
                      Last run: {autoSyncLastInfo}
                    </div>
                  )}
                </>
              )}
              <div className="text-xs text-gray-500 mt-2">
                Your tickets stay in perfect sync, while the ignored ones remain undisturbed
              </div>

              {/* Reverse Sync Not Available Message */}
              <div className="glass-panel border border-amber-200 rounded-lg p-2 mt-5 bg-amber-50">
                <div className="flex items-start">
                  <AlertCircle className="w-4 h-4 mr-2 text-amber-600 flex-shrink-0 mt-0.5" />
                  <div className="text-xs text-gray-700">
                    <span className="font-medium text-amber-700">Note:</span> Auto-sync is not available for reverse create. Use the Reverse Create toggle below instead.
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Auto-Create Control */}
          <div className="glass-panel border border-gray-200 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center">
                <Zap className={`w-5 h-5 mr-2 ${autoCreateRunning ? 'text-blue-600' : 'text-gray-600'}`} />
                <h3 className="text-lg font-semibold text-gray-900">Auto-Create</h3>
                <span className={`ml-2 px-2 py-1 rounded-full text-xs font-medium ${
                  autoCreateRunning ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-600'
                }`}>
                  {autoCreateRunning ? 'RUNNING' : 'STOPPED'}
                </span>
              </div>

              <button
                onClick={handleAutoCreateToggle}
                disabled={toggleLoading.create}
                className={`flex items-center px-4 py-2 rounded-lg font-medium transition-colors ${
                  autoCreateRunning
                    ? 'bg-red-100 text-red-700 hover:bg-red-200'
                    : 'bg-blue-100 text-blue-700 hover:bg-blue-200'
                } disabled:opacity-50`}
              >
                {toggleLoading.create ? (
                  <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                ) : autoCreateRunning ? (
                  <Square className="w-4 h-4 mr-2" />
                ) : (
                  <Play className="w-4 h-4 mr-2" />
                )}
                {autoCreateRunning ? 'Stop' : 'Start'}
              </button>
            </div>

            <div className="space-y-2 text-sm text-gray-600">
              <div className="flex items-center">
                <Clock className="w-4 h-4 mr-2" />
                <span>Every {formatInterval(autoCreateInterval)}</span>
                <button
                  onClick={() => handleOpenIntervalModal('create')}
                  className="interval-edit-button"
                  title="Edit interval"
                >
                  <Edit className="interval-edit-icon" />
                </button>
              </div>
              {autoCreateRunning && (
                <>
                  {autoCreateLastInfo && (
                    <div className="text-xs bg-gray-50 rounded p-2 mt-2">
                      Last run: {autoCreateLastInfo}
                    </div>
                  )}
                </>
              )}
              <div className="text-xs text-gray-500 mt-2">
                Creates what's missing, but never touches the tickets you've sidelined
              </div>
            </div>

            {/* Reverse Create Section */}
            <div className="glass-panel border border-gray-200 rounded-lg p-3 mt-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center">
                  <RefreshCw className={`w-4 h-4 mr-2 ${reverseAutoCreateRunning ? 'text-purple-600' : 'text-gray-500'}`} />
                  <span className="text-sm font-medium text-gray-700">Reverse Create</span>
                  <span className={`ml-2 px-2 py-0.5 rounded-full text-xs font-medium ${
                    reverseAutoCreateRunning ? 'bg-purple-100 text-purple-800' : 'bg-gray-100 text-gray-600'
                  }`}>
                    {reverseAutoCreateRunning ? 'ON' : 'OFF'}
                  </span>
                </div>

                <button
                  onClick={handleReverseAutoCreateToggle}
                  disabled={toggleLoading.reverseCreate}
                  className={`flex items-center px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                    reverseAutoCreateRunning
                      ? 'bg-red-100 text-red-700 hover:bg-red-200'
                      : 'bg-purple-100 text-purple-700 hover:bg-purple-200'
                  } disabled:opacity-50`}
                >
                  {toggleLoading.reverseCreate ? (
                    <RefreshCw className="w-3 h-3 mr-1.5 animate-spin" />
                  ) : reverseAutoCreateRunning ? (
                    <Square className="w-3 h-3 mr-1.5" />
                  ) : (
                    <Play className="w-3 h-3 mr-1.5" />
                  )}
                  {reverseAutoCreateRunning ? 'Stop' : 'Start'}
                </button>
              </div>

              <div className="flex items-center justify-between mt-2 text-xs text-gray-600">
                <span>
                  Creators: <span className="font-medium text-gray-900">{selectedCreators === 'All' ? 'All Users' : `${selectedCreators.split(',').length} selected`}</span>
                </span>
                <button
                  onClick={handleOpenUserSelectionModal}
                  className="interval-edit-button"
                  title="Edit creators"
                >
                  <Edit className="interval-edit-icon" />
                </button>
              </div>

              {reverseAutoCreateRunning && reverseAutoCreateLastInfo && (
                <div className="text-xs bg-gray-50 rounded p-2 mt-2">
                  Last run: {reverseAutoCreateLastInfo}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Column Selection with Glass Theme */}
        <div className="glass-panel border border-gray-200 rounded-lg p-6 interactive-element">
          <div className="flex items-center mb-6">
            <Activity className="w-5 h-5 text-blue-600 mr-2" />
            <FluidText className="text-lg font-semibold text-gray-900" sensitivity={1.2}>
              Select Column
            </FluidText>
          </div>

          {columnsLoading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="w-5 h-5 mr-2 animate-spin text-blue-600" />
              <span className="text-gray-600">Loading columns...</span>
            </div>
          ) : columns.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-600 mb-2">No column mappings configured.</p>
              <p className="text-sm text-gray-500">Please configure your column mappings in Settings → Column Mapping</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 mb-6">
              {columns.map((column) => {
                const IconComponent = column.icon;
                return (
                  <div
                    key={column.value}
                    onClick={() => onColumnSelect(column.value)}
                    className={`glass-panel interactive-element p-4 rounded-lg border cursor-pointer transition-all ${
                      selectedColumn === column.value
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-blue-200 hover:bg-blue-50'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center">
                        {IconComponent && (
                          <IconComponent className="w-5 h-5 mr-2 text-gray-600" />
                        )}
                        <FluidText className="font-medium text-gray-900" sensitivity={0.8}>
                          {column.label}
                        </FluidText>
                      </div>
                      {column.displayOnly && (
                        <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-1 rounded">
                          Display Only
                        </span>
                      )}
                    </div>

                    {selectedColumn === column.value && (
                      <div className="mt-2">
                        <div className="w-full h-1 bg-blue-500 rounded"></div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          <button
            onClick={onAnalyze}
            disabled={!selectedColumn || loading}
            className="interactive-element w-full bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center font-medium transition-colors"
          >
            {loading ? (
              <>
                <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                <FluidText sensitivity={1}>
                  Analyzing {selectedColumnData?.label}...
                </FluidText>
              </>
            ) : (
              <>
                <Zap className="w-4 h-4 mr-2" />
                <FluidText sensitivity={1}>
                  Analyze {selectedColumn ? selectedColumnData?.label : 'Column'}
                </FluidText>
              </>
            )}
          </button>

          {selectedColumn && !loading && (
            <p className="mt-4 text-gray-600 text-sm text-center select-none">
              Let's see what breaks when we touch <strong>{selectedColumnData?.label}</strong>
            </p>
          )}
        </div>

        {/* Footer Status */}
        <div className="mt-8 text-center text-sm text-gray-500">
          <FluidText sensitivity={0.5}>
            Asana-YouTrack Sync • v1.1 • Making Two Apps Talk to Each Other
          </FluidText>
        </div>
      </div>
      {/* Interval Edit Modal */}
      {showIntervalModal && (
        <div className="modal-overlay" onClick={handleCloseIntervalModal}>
          <div className="glass-panel interval-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="interval-modal-title">
              Edit {showIntervalModal === 'sync' ? 'Auto-Sync' : 'Auto-Create'} Interval
            </h3>

            <div className="interval-modal-content">
              <div className="interval-input-group">
                <label className="interval-label">Interval Value (minimum 15 seconds)</label>
                <input
                  type="number"
                  min="1"
                  value={tempIntervalValue}
                  onChange={(e) => setTempIntervalValue(Number(e.target.value))}
                  className="interval-input"
                />
              </div>

              <div className="interval-input-group">
                <label className="interval-label">Time Unit</label>
                <select
                  value={tempIntervalUnit}
                  onChange={(e) => setTempIntervalUnit(e.target.value)}
                  className="interval-select"
                >
                  <option value="seconds">Seconds</option>
                  <option value="minutes">Minutes</option>
                  <option value="hours">Hours</option>
                </select>
              </div>

              <div className="interval-preview">
                {(() => {
                  let totalSeconds = tempIntervalValue;
                  if (tempIntervalUnit === 'minutes') {
                    totalSeconds = tempIntervalValue * 60;
                  } else if (tempIntervalUnit === 'hours') {
                    totalSeconds = tempIntervalValue * 3600;
                  }

                  if (totalSeconds < 15) {
                    return (
                      <span style={{ color: '#dc2626' }}>
                        ⚠ Minimum interval is 15 seconds
                      </span>
                    );
                  }

                  return `Will run every ${tempIntervalValue} ${tempIntervalUnit}`;
                })()}
              </div>
            </div>

            <div className="interval-modal-actions">
              <button
                onClick={handleCloseIntervalModal}
                className="interval-modal-button cancel"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveInterval}
                disabled={(tempIntervalUnit === 'seconds' && tempIntervalValue < 15)}
                className={`interval-modal-button save ${
                  (tempIntervalUnit === 'seconds' && tempIntervalValue < 15) ? 'disabled' : ''
                }`}
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* User Selection Modal */}
      {showUserSelectionModal && (
        <div className="modal-overlay" onClick={handleCloseUserSelectionModal}>
          <div className="glass-panel interval-modal" onClick={(e) => e.stopPropagation()} style={{ maxWidth: '500px' }}>
            <h3 className="interval-modal-title">
              Select Creators for Reverse Auto-Create
            </h3>

            <div className="interval-modal-content">
              {usersLoading ? (
                <div className="flex items-center justify-center py-8">
                  <RefreshCw className="w-5 h-5 mr-2 animate-spin text-purple-600" />
                  <span className="text-gray-600">Loading users...</span>
                </div>
              ) : (
                <>
                  <div className="mb-4">
                    <button
                      onClick={handleSelectAllUsers}
                      className={`w-full px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                        tempSelectedCreators.length === 0
                          ? 'bg-purple-100 text-purple-800 border-2 border-purple-500'
                          : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                      }`}
                    >
                      All Users
                    </button>
                  </div>

                  <div className="space-y-2 max-h-96 overflow-y-auto">
                    {youtrackUsers.length === 0 ? (
                      <div className="text-center py-4 text-gray-500">
                        No users found
                      </div>
                    ) : (
                      youtrackUsers.map((user, index) => {
                        const userName = user.name || user.fullName || user.login || `User ${index + 1}`;
                        return (
                          <button
                            key={user.login || user.name || user.id || index}
                            onClick={() => handleToggleUser(userName)}
                            className={`w-full px-4 py-2 rounded-lg text-sm text-left transition-colors ${
                              tempSelectedCreators.includes(userName)
                                ? 'bg-purple-100 text-purple-800 border-2 border-purple-500'
                                : 'bg-gray-50 text-gray-700 hover:bg-gray-100'
                            }`}
                          >
                            <div className="flex items-center justify-between">
                              <span className="font-medium">{userName}</span>
                              {tempSelectedCreators.includes(userName) && (
                                <CheckCircle className="w-4 h-4 text-purple-600" />
                              )}
                            </div>
                          </button>
                        );
                      })
                    )}
                  </div>

                  <div className="mt-4 p-3 bg-blue-50 rounded-lg">
                    <p className="text-xs text-blue-800">
                      {tempSelectedCreators.length === 0
                        ? 'All YouTrack users will be included'
                        : `${tempSelectedCreators.length} creator${tempSelectedCreators.length > 1 ? 's' : ''} selected`}
                    </p>
                  </div>
                </>
              )}
            </div>

            <div className="interval-modal-actions">
              <button
                onClick={handleCloseUserSelectionModal}
                className="interval-modal-button cancel"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveUserSelection}
                disabled={usersLoading}
                className={`interval-modal-button save ${usersLoading ? 'disabled' : ''}`}
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Dashboard;