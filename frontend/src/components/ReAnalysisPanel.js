import React, { useState } from 'react';
import { RefreshCw, BarChart3, ArrowLeft, Zap } from 'lucide-react';

const ReAnalysisPanel = ({ 
  selectedColumn, 
  lastActionType, 
  lastActionCount, 
  onReAnalyze, 
  onBackToDashboard,
  loading 
}) => {
  const [selectedReAnalysisColumn, setSelectedReAnalysisColumn] = useState(selectedColumn);

  const columns = [
    { value: 'backlog', label: 'Backlog' },
    { value: 'in_progress', label: 'In Progress' },
    { value: 'dev', label: 'DEV' },
    { value: 'stage', label: 'STAGE' },
    { value: 'blocked', label: 'Blocked' },
    { value: 'all_syncable', label: 'All Columns' }
  ];

  const getActionMessage = () => {
    switch (lastActionType) {
      case 'sync':
        return `🔄 ${lastActionCount} tickets synced successfully!`;
      case 'create':
        return `✅ ${lastActionCount} tickets created successfully!`;
      case 'bulk_create':
        return `🚀 ${lastActionCount} tickets created in bulk!`;
      default:
        return 'Action completed successfully!';
    }
  };

  const handleQuickReAnalyze = () => {
    onReAnalyze(selectedColumn);
  };

  const handleCustomReAnalyze = () => {
    onReAnalyze(selectedReAnalysisColumn);
  };

  const handleAnalyzeAll = () => {
    onReAnalyze('all_syncable');
  };

  return (
    <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6 mb-6">
      {/* Success Message */}
      <div className="flex items-center mb-6">
        <div className="flex-1">
          <div className="text-lg font-semibold text-gray-900 mb-2">
            {getActionMessage()}
          </div>
          <p className="text-sm text-gray-600">
            Choose how to proceed with your analysis workflow.
          </p>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        {/* Quick Re-analyze Same Column */}
        <button
          onClick={handleQuickReAnalyze}
          disabled={loading}
          className="glass-panel interactive-element p-4 rounded-lg border border-blue-200 bg-blue-50 hover:bg-blue-100 disabled:opacity-50 transition-all"
        >
          <div className="flex items-center justify-center mb-2">
            <RefreshCw className={`w-5 h-5 text-blue-600 ${loading ? 'animate-spin' : ''}`} />
          </div>
          <div className="text-sm font-medium text-blue-900">
            Re-analyze "{selectedColumn?.replace('_', ' ').toUpperCase()}"
          </div>
          <div className="text-xs text-blue-700 mt-1">
            Same column, fresh data
          </div>
        </button>

        {/* Analyze All Columns */}
        <button
          onClick={handleAnalyzeAll}
          disabled={loading}
          className="glass-panel interactive-element p-4 rounded-lg border border-green-200 bg-green-50 hover:bg-green-100 disabled:opacity-50 transition-all"
        >
          <div className="flex items-center justify-center mb-2">
            <BarChart3 className="w-5 h-5 text-green-600" />
          </div>
          <div className="text-sm font-medium text-green-900">
            Analyze All Columns
          </div>
          <div className="text-xs text-green-700 mt-1">
            Complete overview
          </div>
        </button>

        {/* Back to Dashboard */}
        <button
          onClick={onBackToDashboard}
          className="glass-panel interactive-element p-4 rounded-lg border border-gray-200 hover:bg-gray-50 transition-all"
        >
          <div className="flex items-center justify-center mb-2">
            <ArrowLeft className="w-5 h-5 text-gray-600" />
          </div>
          <div className="text-sm font-medium text-gray-900">
            Back to Dashboard
          </div>
          <div className="text-xs text-gray-600 mt-1">
            Start fresh
          </div>
        </button>
      </div>

      {/* Advanced Re-analysis Options */}
      <div className="border-t border-gray-200 pt-6">
        <h4 className="text-sm font-semibold text-gray-900 mb-4">
          Or analyze a specific column:
        </h4>
        
        <div className="flex flex-wrap gap-2 mb-4">
          {columns.map((column) => (
            <button
              key={column.value}
              onClick={() => setSelectedReAnalysisColumn(column.value)}
              className={`glass-panel px-3 py-2 rounded-lg text-sm font-medium transition-all ${
                selectedReAnalysisColumn === column.value
                  ? 'border-blue-500 bg-blue-50 text-blue-900'
                  : 'border-gray-200 text-gray-700 hover:border-blue-300 hover:bg-blue-50'
              }`}
            >
              {column.label}
            </button>
          ))}
        </div>

        <button
          onClick={handleCustomReAnalyze}
          disabled={loading || selectedReAnalysisColumn === selectedColumn}
          className="glass-panel bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center font-medium transition-colors"
        >
          <Zap className="w-4 h-4 mr-2" />
          {loading ? 'Analyzing...' : `Analyze ${columns.find(c => c.value === selectedReAnalysisColumn)?.label || 'Selected'}`}
        </button>
      </div>
    </div>
  );
};

export default ReAnalysisPanel;