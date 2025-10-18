import React, { useState, useEffect } from 'react';
<<<<<<< HEAD
import { RefreshCw, Zap, Activity, Play, Square, Clock, Inbox, Loader, Code, CheckCircle, AlertCircle, Package, Search, Layers, Edit } from 'lucide-react';
import FluidText from './FluidText';
import '../styles/dashboard-glass.css';
=======
import { RefreshCw, Zap, Activity, Play, Square, Clock, Inbox, Loader, Code, Package, AlertCircle, CheckCircle, Search, Layers } from 'lucide-react';
import FluidText from './FluidText';
>>>>>>> features
import {
  getAutoSyncStatus,
  startAutoSync,
  stopAutoSync,
  getAutoCreateStatus,
  startAutoCreate,
<<<<<<< HEAD
  stopAutoCreate
=======
  stopAutoCreate,
  getUserSettings
>>>>>>> features
} from '../services/api';

const Dashboard = ({ selectedColumn, onColumnSelect, onAnalyze, loading }) => {
  const [autoSyncRunning, setAutoSyncRunning] = useState(false);
  const [autoCreateRunning, setAutoCreateRunning] = useState(false);
  const [autoSyncInterval, setAutoSyncInterval] = useState(15);
  const [autoCreateInterval, setAutoCreateInterval] = useState(15);
  const [autoSyncLastInfo, setAutoSyncLastInfo] = useState('');
  const [autoCreateLastInfo, setAutoCreateLastInfo] = useState('');
  const [toggleLoading, setToggleLoading] = useState({ sync: false, create: false });
<<<<<<< HEAD

  // Interval editing state
  const [showIntervalModal, setShowIntervalModal] = useState(null); // 'sync' or 'create'
  const [tempIntervalValue, setTempIntervalValue] = useState(15);
  const [tempIntervalUnit, setTempIntervalUnit] = useState('seconds'); // 'seconds', 'minutes', 'hours'

  const columns = [
    {
      value: 'backlog',
      label: 'Backlog only',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      icon: Inbox
    },
    {
      value: 'in_progress',
      label: 'In Progress only',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      icon: Loader
    },
    {
      value: 'dev',
      label: 'DEV only',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      icon: Code
    },
    {
      value: 'stage',
      label: 'STAGE only',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      icon: Package
    },
    {
      value: 'blocked',
      label: 'Blocked only',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      icon: AlertCircle
    },
    {
      value: 'ready_for_stage',
      label: 'Ready for Stage',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      icon: CheckCircle
    },
    {
      value: 'findings',
      label: 'Findings',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      displayOnly: true,
      icon: Search
    },
    {
      value: 'all_syncable',
      label: 'All Syncable',
      color: 'hover:bg-blue-50 hover:border-blue-200',
      icon: Layers
    }
  ];
=======
  const [columns, setColumns] = useState([]);
  const [columnsLoading, setColumnsLoading] = useState(true);
>>>>>>> features

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
      
      const [syncStatus, createStatus] = await Promise.all([
        getAutoSyncStatus(),
        getAutoCreateStatus()
      ]);
      
      console.log('Auto-sync status:', syncStatus); // DEBUG
      console.log('Auto-create status:', createStatus); // DEBUG
      
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
    // Convert to seconds based on unit
    let intervalInSeconds = tempIntervalValue;
    if (tempIntervalUnit === 'minutes') {
      intervalInSeconds = tempIntervalValue * 60;
    } else if (tempIntervalUnit === 'hours') {
      intervalInSeconds = tempIntervalValue * 3600;
    }

    // Validate minimum interval of 15 seconds - prevent save but don't show alert
    if (intervalInSeconds < 15) {
      return;
    }

    if (showIntervalModal === 'sync') {
      setAutoSyncInterval(intervalInSeconds);
      // If running, restart with new interval
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
      // If running, restart with new interval
      if (autoCreateRunning) {
        try {
          await stopAutoCreate();
          await startAutoCreate(intervalInSeconds);
        } catch (error) {
          console.error('Failed to update create interval:', error);
        }
      }
    }

    handleCloseIntervalModal();
  };

  const formatInterval = (seconds) => {
    if (seconds < 60) {
      return `${seconds} seconds`;
    } else if (seconds < 3600) {
      return `${Math.floor(seconds / 60)} minutes`;
    } else {
      return `${Math.floor(seconds / 3600)} hours`;
    }
  };

  return (
    <div>
      {/* Main Content */}
      <div className="dashboard-container">
        {/* Header Section with Fluid Text */}
        <div className="dashboard-header">
          <FluidText className="fluid-text dashboard-title text-gray-900" sensitivity={2}>
            Pick a column and let's see how badly these two systems disagree with each other
          </FluidText>
        </div>

        {/* NEW: Auto Controls Section */}
        <div className="auto-controls-grid">
          {/* Auto-Sync Control */}
          <div className="glass-panel auto-control-panel">
            <div className="auto-control-header">
              <div className="auto-control-title-group">
                <RefreshCw className={`auto-control-icon ${autoSyncRunning ? 'sync-running running' : 'stopped'}`} />
                <h3 className="auto-control-title">Auto-Sync</h3>
                <span className={`auto-status-badge ${autoSyncRunning ? 'running' : 'stopped'}`}>
                  {autoSyncRunning ? 'RUNNING' : 'STOPPED'}
                </span>
              </div>

              <button
                onClick={handleAutoSyncToggle}
                disabled={toggleLoading.sync}
                className={`auto-control-button ${autoSyncRunning ? 'stop' : 'start'} ${toggleLoading.sync ? 'disabled' : ''}`}
              >
                {toggleLoading.sync ? (
                  <RefreshCw className="auto-control-button-icon running" />
                ) : autoSyncRunning ? (
                  <Square className="auto-control-button-icon" />
                ) : (
                  <Play className="auto-control-button-icon" />
                )}
                {autoSyncRunning ? 'Stop' : 'Start'}
              </button>
            </div>

            <div className="auto-control-info">
              <div className="auto-control-info-row">
                <Clock className="auto-control-info-icon" />
                <span>Every {formatInterval(autoSyncInterval)}</span>
                <button
                  onClick={() => handleOpenIntervalModal('sync')}
                  className="interval-edit-button"
                  title="Edit interval"
                >
                  <Edit className="interval-edit-icon" />
                </button>
              </div>
              {autoSyncRunning && autoSyncLastInfo && (
                <div className="auto-control-last-info">
                  Last run: {autoSyncLastInfo}
                </div>
              )}
              <div className="auto-control-description">
                Your tickets stay in perfect sync, while the ignored ones remain undisturbed
              </div>
            </div>
          </div>

          {/* Auto-Create Control */}
          <div className="glass-panel auto-control-panel">
            <div className="auto-control-header">
              <div className="auto-control-title-group">
                <Zap className={`auto-control-icon ${autoCreateRunning ? 'create-running' : 'stopped'}`} />
                <h3 className="auto-control-title">Auto-Create</h3>
                <span className={`auto-status-badge ${autoCreateRunning ? 'running' : 'stopped'}`}>
                  {autoCreateRunning ? 'RUNNING' : 'STOPPED'}
                </span>
              </div>

              <button
                onClick={handleAutoCreateToggle}
                disabled={toggleLoading.create}
                className={`auto-control-button ${autoCreateRunning ? 'stop' : 'start'} ${toggleLoading.create ? 'disabled' : ''}`}
              >
                {toggleLoading.create ? (
                  <RefreshCw className="auto-control-button-icon running" />
                ) : autoCreateRunning ? (
                  <Square className="auto-control-button-icon" />
                ) : (
                  <Play className="auto-control-button-icon" />
                )}
                {autoCreateRunning ? 'Stop' : 'Start'}
              </button>
            </div>

            <div className="auto-control-info">
              <div className="auto-control-info-row">
                <Clock className="auto-control-info-icon" />
                <span>Every {formatInterval(autoCreateInterval)}</span>
                <button
                  onClick={() => handleOpenIntervalModal('create')}
                  className="interval-edit-button"
                  title="Edit interval"
                >
                  <Edit className="interval-edit-icon" />
                </button>
              </div>
              {autoCreateRunning && autoCreateLastInfo && (
                <div className="auto-control-last-info">
                  Last run: {autoCreateLastInfo}
                </div>
              )}
              <div className="auto-control-description">
                Creates what's missing, but never touches the tickets you've sidelined
              </div>
            </div>
          </div>
        </div>

        {/* Column Selection with Glass Theme */}
        <div className="glass-panel column-selection-panel interactive-element">
          <div className="column-selection-header">
            <Activity className="column-selection-icon" />
            <FluidText className="fluid-text column-selection-title" sensitivity={1.2}>
              Select Column
            </FluidText>
          </div>

<<<<<<< HEAD
          <div className="column-grid">
            {columns.map((column) => {
              const IconComponent = column.icon;
              return (
                <div
                  key={column.value}
                  onClick={() => onColumnSelect(column.value)}
                  className={`glass-panel column-card interactive-element ${
                    selectedColumn === column.value ? 'selected' : 'unselected'
                  }`}
                >
                  <div className="column-card-content">
                    {IconComponent && (
                      <IconComponent className="column-card-icon" size={20} />
                    )}
                    <FluidText className="fluid-text column-card-label" sensitivity={0.8}>
                      {column.label}
                    </FluidText>
                    {column.displayOnly && (
                      <span className="column-card-badge">
                        Display Only
                      </span>
                    )}
                  </div>

                  {selectedColumn === column.value && (
                    <div className="column-card-indicator">
                      <div className="column-card-indicator-bar"></div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
=======
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
>>>>>>> features

          <button
            onClick={onAnalyze}
            disabled={!selectedColumn || loading}
            className={`interactive-element analyze-button ${!selectedColumn || loading ? 'disabled' : 'active'}`}
          >
            {loading ? (
              <>
                <RefreshCw className="analyze-button-icon running" />
                <FluidText className="fluid-text" sensitivity={1}>
                  Analyzing {selectedColumnData?.label}...
                </FluidText>
              </>
            ) : (
              <>
                <Zap className="analyze-button-icon" />
                <FluidText className="fluid-text" sensitivity={1}>
                  Analyze {selectedColumn ? selectedColumnData?.label : 'Column'}
                </FluidText>
              </>
            )}
          </button>

          {selectedColumn && !loading && (
<<<<<<< HEAD
            <div className="selected-column-info">
              <p className="selected-column-info-text">
                Let's see what breaks when we touch <strong>{selectedColumnData?.label}</strong>
              </p>
            </div>
=======
            <p className="mt-4 text-gray-600 text-sm text-center select-none">
              Let's see what breaks when we touch <strong>{selectedColumnData?.label}</strong>
            </p>
>>>>>>> features
          )}
        </div>

        {/* Footer Status */}
        <div className="dashboard-footer">
          <FluidText className="fluid-text" sensitivity={0.5}>
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
                  // Calculate total seconds
                  let totalSeconds = tempIntervalValue;
                  if (tempIntervalUnit === 'minutes') {
                    totalSeconds = tempIntervalValue * 60;
                  } else if (tempIntervalUnit === 'hours') {
                    totalSeconds = tempIntervalValue * 3600;
                  }

                  // Show warning if below 15 seconds
                  if (totalSeconds < 15) {
                    return (
                      <span style={{ color: '#dc2626' }}>
                        ⚠ Minimum interval is 15 seconds
                      </span>
                    );
                  }

                  // Show normal preview
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
                disabled={
                  (tempIntervalUnit === 'seconds' && tempIntervalValue < 15)
                }
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
    </div>
  );
};

export default Dashboard;