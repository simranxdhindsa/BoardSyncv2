import React, { useState, useEffect } from 'react';
import { RefreshCw, Zap, Activity, Play, Square, Clock, Inbox, Loader, Code, CheckCircle, AlertCircle, Package, Search, Layers } from 'lucide-react';
import FluidText from './FluidText';
import '../styles/dashboard-glass.css';
import {
  getAutoSyncStatus,
  startAutoSync,
  stopAutoSync,
  getAutoCreateStatus,
  startAutoCreate,
  stopAutoCreate
} from '../services/api';

const Dashboard = ({ selectedColumn, onColumnSelect, onAnalyze, loading }) => {
  const [autoSyncRunning, setAutoSyncRunning] = useState(false);
  const [autoCreateRunning, setAutoCreateRunning] = useState(false);
  const [autoSyncInterval, setAutoSyncInterval] = useState(15);
  const [autoCreateInterval, setAutoCreateInterval] = useState(15);
  const [autoSyncCount, setAutoSyncCount] = useState(0);
  const [autoCreateCount, setAutoCreateCount] = useState(0);
  const [autoSyncLastInfo, setAutoSyncLastInfo] = useState('');
  const [autoCreateLastInfo, setAutoCreateLastInfo] = useState('');
  const [toggleLoading, setToggleLoading] = useState({ sync: false, create: false });

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

  const selectedColumnData = columns.find(col => col.value === selectedColumn);

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
        setAutoSyncCount(syncStatus.auto_sync.count || 0);
        setAutoSyncLastInfo(syncStatus.auto_sync.last_info || '');
      }
      
      if (createStatus.auto_create) {
        setAutoCreateRunning(createStatus.auto_create.running);
        setAutoCreateInterval(createStatus.auto_create.interval);
        setAutoCreateCount(createStatus.auto_create.count || 0);
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
                <span>Every {autoSyncInterval} seconds</span>
              </div>
              {autoSyncRunning && (
                <>
                  <div>Cycles completed: {autoSyncCount}</div>
                  {autoSyncLastInfo && (
                    <div className="auto-control-last-info">
                      Last run: {autoSyncLastInfo}
                    </div>
                  )}
                </>
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
                <span>Every {autoCreateInterval} seconds</span>
              </div>
              {autoCreateRunning && (
                <>
                  <div>Cycles completed: {autoCreateCount}</div>
                  {autoCreateLastInfo && (
                    <div className="auto-control-last-info">
                      Last run: {autoCreateLastInfo}
                    </div>
                  )}
                </>
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
            <div className="selected-column-info">
              <p className="selected-column-info-text">
                Let's see what breaks when we touch <strong>{selectedColumnData?.label}</strong>
              </p>
            </div>
          )}
        </div>

        {/* Footer Status */}
        <div className="dashboard-footer">
          <FluidText className="fluid-text" sensitivity={0.5}>
            Asana-YouTrack Sync • v1.1 • Making Two Apps Talk to Each Other
          </FluidText>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;