// frontend/src/components/FilterPanel.js
// COMPLETE FILE - Create this new file

import React, { useState, useEffect } from 'react';
import { X, Calendar, Users, Zap } from 'lucide-react';

const FilterPanel = ({ 
  filterOptions, 
  currentFilters, 
  onApply, 
  onClear, 
  onClose 
}) => {
  const [localFilters, setLocalFilters] = useState(currentFilters);

  useEffect(() => {
    setLocalFilters(currentFilters);
  }, [currentFilters]);

  const handleAssigneeToggle = (assignee) => {
    setLocalFilters(prev => ({
      ...prev,
      assignees: prev.assignees.includes(assignee)
        ? prev.assignees.filter(a => a !== assignee)
        : [...prev.assignees, assignee]
    }));
  };

  const handlePriorityToggle = (priority) => {
    setLocalFilters(prev => ({
      ...prev,
      priorities: prev.priorities.includes(priority)
        ? prev.priorities.filter(p => p !== priority)
        : [...prev.priorities, priority]
    }));
  };

  const handleDateChange = (field, value) => {
    setLocalFilters(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleApply = () => {
    onApply(localFilters);
    onClose();
  };

  const handleClear = () => {
    const clearedFilters = {
      assignees: [],
      priorities: [],
      start_date: null,
      end_date: null
    };
    setLocalFilters(clearedFilters);
    onClear();
    onClose();
  };

  const activeFilterCount = 
    localFilters.assignees.length + 
    localFilters.priorities.length + 
    (localFilters.start_date ? 1 : 0) + 
    (localFilters.end_date ? 1 : 0);

  const getPriorityColor = (priority) => {
    switch (priority?.toLowerCase()) {
      case 'urgent': return 'text-red-600';
      case 'high': return 'text-orange-600';
      case 'medium': return 'text-yellow-600';
      case 'low': return 'text-green-600';
      default: return 'text-gray-600';
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex justify-end">
      <div className="w-96 bg-white h-full shadow-2xl overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 z-10">
          <div className="flex items-center justify-between mb-2">
            <h2 className="text-xl font-semibold text-gray-900">Filters</h2>
            <button
              onClick={onClose}
              className="p-1 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <X className="w-5 h-5 text-gray-500" />
            </button>
          </div>
          {activeFilterCount > 0 && (
            <p className="text-sm text-blue-600 font-medium">
              {activeFilterCount} filter{activeFilterCount !== 1 ? 's' : ''} active
            </p>
          )}
        </div>

        {/* Filter Content */}
        <div className="px-6 py-4 space-y-6">
          {/* Assignees */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900 mb-3 flex items-center">
              <Users className="w-4 h-4 mr-2" />
              Assignees
            </h3>
            <div className="space-y-2">
              {filterOptions?.assignees?.map(assignee => (
                <label key={assignee} className="flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={localFilters.assignees.includes(assignee)}
                    onChange={() => handleAssigneeToggle(assignee)}
                    className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                  />
                  <span className="ml-2 text-sm text-gray-700">{assignee}</span>
                </label>
              ))}
              {(!filterOptions?.assignees || filterOptions.assignees.length === 0) && (
                <p className="text-sm text-gray-500 italic">No assignees available</p>
              )}
            </div>
          </div>

          {/* Priorities */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900 mb-3 flex items-center">
              <Zap className="w-4 h-4 mr-2" />
              Priorities
            </h3>
            <div className="space-y-2">
              {filterOptions?.priorities?.map(priority => (
                <label key={priority} className="flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={localFilters.priorities.includes(priority)}
                    onChange={() => handlePriorityToggle(priority)}
                    className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                  />
                  <span className={`ml-2 text-sm font-medium ${getPriorityColor(priority)}`}>
                    {priority}
                  </span>
                </label>
              ))}
              {(!filterOptions?.priorities || filterOptions.priorities.length === 0) && (
                <p className="text-sm text-gray-500 italic">No priorities available</p>
              )}
            </div>
          </div>

          {/* Date Range */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900 mb-3 flex items-center">
              <Calendar className="w-4 h-4 mr-2" />
              Created Date Range
            </h3>
            <div className="space-y-3">
              <div>
                <label className="block text-xs text-gray-600 mb-1">From</label>
                <input
                  type="date"
                  value={localFilters.start_date || ''}
                  onChange={(e) => handleDateChange('start_date', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 mb-1">To</label>
                <input
                  type="date"
                  value={localFilters.end_date || ''}
                  onChange={(e) => handleDateChange('end_date', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="sticky bottom-0 bg-white border-t border-gray-200 px-6 py-4">
          <div className="flex space-x-3">
            <button
              onClick={handleClear}
              className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium"
            >
              Clear All
            </button>
            <button
              onClick={handleApply}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
            >
              Apply Filters
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FilterPanel;