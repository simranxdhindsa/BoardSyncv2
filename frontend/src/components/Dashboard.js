import React, { useState, useEffect } from 'react';
import { RefreshCw, Zap, Activity, Play, Square, Clock } from 'lucide-react';
import FluidText from './FluidText';
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
      color: 'hover:bg-blue-50 hover:border-blue-200'
    },
    { 
      value: 'in_progress', 
      label: 'In Progress only', 
      color: 'hover:bg-blue-50 hover:border-blue-200'
    },
    { 
      value: 'dev', 
      label: 'DEV only', 
      color: 'hover:bg-blue-50 hover:border-blue-200'
    },
    { 
      value: 'stage', 
      label: 'STAGE only', 
      color: 'hover:bg-blue-50 hover:border-blue-200'
    },
    { 
      value: 'blocked', 
      label: 'Blocked only', 
      color: 'hover:bg-blue-50 hover:border-blue-200'
    },
    { 
      value: 'ready_for_stage', 
      label: 'Ready for Stage', 
      color: 'hover:bg-blue-50 hover:border-blue-200',
      displayOnly: true
    },
    { 
      value: 'findings', 
      label: 'Findings', 
      color: 'hover:bg-blue-50 hover:border-blue-200',
      displayOnly: true
    },
    { 
      value: 'all_syncable', 
      label: 'All Syncable', 
      color: 'hover:bg-blue-50 hover:border-blue-200'
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
          <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
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
                <span>Every {autoSyncInterval} seconds</span>
              </div>
              {autoSyncRunning && (
                <>
                  <div>Cycles completed: {autoSyncCount}</div>
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
            </div>
          </div>

          {/* Auto-Create Control */}
          <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
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
                <span>Every {autoCreateInterval} seconds</span>
              </div>
              {autoCreateRunning && (
                <>
                  <div>Cycles completed: {autoCreateCount}</div>
                  {autoCreateLastInfo && (
                    <div className="text-xs bg-gray-50 rounded p-2 mt-2">
                      Last run: {autoCreateLastInfo}
                    </div>
                  )}
                </>
              )}
              <div className="text-xs text-gray-500 mt-2">
                Creates what’s missing, but never touches the tickets you’ve sidelined
              </div>
            </div>
          </div>
        </div>

        {/* Column Selection with Glass Theme */}
        <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6 interactive-element">
          <div className="flex items-center mb-6">
            <Activity className="w-5 h-5 text-blue-600 mr-2" />
            <FluidText className="text-lg font-semibold text-gray-900" sensitivity={1.2}>
              Select Column
            </FluidText>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 mb-6">
            {columns.map((column) => (
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
                  <FluidText className="font-medium text-gray-900" sensitivity={0.8}>
                    {column.label}
                  </FluidText>
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
            ))}
          </div>
          
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
            <div className="glass-panel mt-4 p-3 bg-blue-50 rounded-lg border border-blue-200 pointer-events-none">
              <p className="text-blue-800 text-sm text-center select-none">
                Let's see what breaks when we touch <strong>{selectedColumnData?.label}</strong>
              </p>
            </div>
          )}
        </div>

        {/* Footer Status */}
        <div className="mt-8 text-center text-sm text-gray-500">
          <FluidText sensitivity={0.5}>
            Asana-YouTrack Sync • v1.1 • Making Two Apps Talk to Each Other
          </FluidText>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;