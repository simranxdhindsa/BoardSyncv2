// FILE: frontend/src/components/mapping/MappingComponents.js
// Enhanced Mapping Components with Auto-mapping Support

import React, { useState, useEffect } from 'react';
import { Link2, Plus, Trash2, RefreshCw, CheckCircle, AlertTriangle, Eye, Zap } from 'lucide-react';
import mappingService from '../../services/mappingService';
import { getUserSettings } from '../../services/api';

// Create Mapping Form Component
export const CreateMappingForm = ({ onSuccess }) => {
  const [asanaUrl, setAsanaUrl] = useState('');
  const [youtrackUrl, setYoutrackUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const response = await mappingService.createMapping(asanaUrl, youtrackUrl);
      
      if (response.success) {
        setAsanaUrl('');
        setYoutrackUrl('');
        if (onSuccess) onSuccess(response.data);
      } else {
        setError(response.message || 'Failed to create mapping');
      }
    } catch (err) {
      setError(err.message || 'Network error occurred');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
      <div className="flex items-center mb-4">
        <Link2 className="w-5 h-5 text-blue-600 mr-2" />
        <h2 className="text-xl font-semibold text-gray-900">Link Tickets Manually</h2>
      </div>
      
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Asana Task URL
          </label>
          <input
            type="url"
            value={asanaUrl}
            onChange={(e) => setAsanaUrl(e.target.value)}
            placeholder="https://app.asana.com/.../task/1211475287717816"
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 settings-input"
            required
          />
          <p className="text-xs text-gray-500 mt-1">
            Copy the full URL from your Asana task
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            YouTrack Issue URL
          </label>
          <input
            type="url"
            value={youtrackUrl}
            onChange={(e) => setYoutrackUrl(e.target.value)}
            placeholder="https://youtrack.cloud/issue/ARD-222/Title"
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 settings-input"
            required
          />
          <p className="text-xs text-gray-500 mt-1">
            Copy the full URL from your YouTrack issue
          </p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-3">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        <button
          type="submit"
          disabled={loading || !asanaUrl.trim() || !youtrackUrl.trim()}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed flex items-center justify-center font-medium transition-colors"
        >
          {loading ? (
            <>
              <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
              Linking...
            </>
          ) : (
            <>
              <Plus className="w-4 h-4 mr-2" />
              Link Tickets
            </>
          )}
        </button>
      </form>

      <div className="mt-4 bg-blue-50 border border-blue-200 rounded-lg p-3">
        <h3 className="text-sm font-medium text-blue-900 mb-2 flex items-center">
          <CheckCircle className="w-4 h-4 mr-1" />
          How it works:
        </h3>
        <ul className="text-xs text-blue-700 space-y-1">
          <li>• Paste the full URL from both systems</li>
          <li>• System extracts task/issue IDs automatically</li>
          <li>• Future syncs will recognize this link</li>
          <li>• Title changes won't break the connection</li>
        </ul>
      </div>
    </div>
  );
};

// Mappings List Component
export const MappingsList = ({ refreshTrigger }) => {
  const [mappings, setMappings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [youtrackBaseUrl, setYoutrackBaseUrl] = useState('');

  useEffect(() => {
  const loadBaseUrl = async () => {
    try {
      const settings = await getUserSettings();
      setYoutrackBaseUrl(settings.data?.youtrack_base_url || settings.youtrack_base_url || '');
    } catch (err) {
      console.error('Failed to load settings:', err);
    }
  };
  loadBaseUrl();
}, []);

  useEffect(() => {
    fetchMappings();
  }, [refreshTrigger]);

  const fetchMappings = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await mappingService.getAllMappings();
      if (response.success) {
        setMappings(response.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch mappings:', err);
      setError('Failed to load mappings');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Are you sure you want to delete this mapping?')) return;

    try {
      const response = await mappingService.deleteMapping(id);
      if (response.success) {
        setMappings(mappings.filter(m => m.id !== id));
      }
    } catch (err) {
      console.error('Failed to delete mapping:', err);
      alert('Failed to delete mapping: ' + err.message);
    }
  };

  if (loading) {
    return (
      <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
        <div className="text-center py-8">
          <RefreshCw className="w-8 h-8 text-gray-400 mx-auto mb-2 animate-spin" />
          <p className="text-gray-600">Loading mappings...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center">
            <AlertTriangle className="w-5 h-5 text-red-600 mr-2" />
            <p className="text-red-800">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  if (mappings.length === 0) {
    return (
      <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
          <Link2 className="w-12 h-12 text-gray-400 mx-auto mb-3" />
          <p className="text-gray-600 font-medium">No ticket mappings yet</p>
          <p className="text-sm text-gray-500 mt-2">
            Create your first mapping to manually link Asana tasks with YouTrack issues
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="glass-panel bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900 flex items-center">
            <Link2 className="w-5 h-5 mr-2" />
            Active Ticket Mappings
          </h2>
          <p className="text-sm text-gray-600 mt-1">
            {mappings.length} ticket{mappings.length !== 1 ? 's' : ''} linked
          </p>
        </div>
        <button
          onClick={fetchMappings}
          className="flex items-center text-blue-600 hover:text-blue-800 transition-colors text-sm font-medium"
        >
          <RefreshCw className="w-4 h-4 mr-1" />
          Refresh
        </button>
      </div>

      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                Asana Task ID
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                YouTrack Issue ID
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                Created
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {mappings.map((mapping) => (
              <tr key={mapping.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center">
                    <span className="text-sm font-mono text-gray-900">
                      {mapping.asana_task_id}
                    </span>
                    <a
                      href={`https://app.asana.com/0/${mapping.asana_project_id}/${mapping.asana_task_id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="ml-2 text-blue-600 hover:text-blue-800"
                      title="Open in Asana"
                    >
                      <Eye className="w-4 h-4" />
                    </a>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center">
                    <span className="text-sm font-mono text-gray-900">
                      {mapping.youtrack_issue_id}
                    </span>
                    <a
                      href={`${youtrackBaseUrl}/issue/${mapping.youtrack_issue_id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="ml-2 text-blue-600 hover:text-blue-800"
                      title="Open in YouTrack"
                    >
                      <Eye className="w-4 h-4" />
                    </a>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {new Date(mapping.created_at).toLocaleDateString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                  <button
                    onClick={() => handleDelete(mapping.id)}
                    className="text-red-600 hover:text-red-800 font-medium flex items-center ml-auto transition-colors"
                  >
                    <Trash2 className="w-4 h-4 mr-1" />
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Auto-mapping Notification Component
export const AutoMappingNotification = ({ autoMappedTickets = [], onDismiss }) => {
  if (!autoMappedTickets || autoMappedTickets.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 max-w-md">
      <div className="bg-green-50 border-2 border-green-200 rounded-lg p-4 shadow-lg">
        <div className="flex items-start">
          <div className="flex-shrink-0">
            <Zap className="w-6 h-6 text-green-600" />
          </div>
          <div className="ml-3 flex-1">
            <h3 className="text-sm font-medium text-green-900">
              Auto-mapped {autoMappedTickets.length} ticket{autoMappedTickets.length !== 1 ? 's' : ''}
            </h3>
            <div className="mt-2 text-xs text-green-700">
              {autoMappedTickets.slice(0, 3).map((ticket, index) => (
                <div key={index} className="mb-1">
                  • {ticket.asana_task_id} ↔ {ticket.youtrack_issue_id}
                </div>
              ))}
              {autoMappedTickets.length > 3 && (
                <div className="text-green-600 font-medium mt-1">
                  +{autoMappedTickets.length - 3} more
                </div>
              )}
            </div>
          </div>
          {onDismiss && (
            <button
              onClick={onDismiss}
              className="ml-4 text-green-600 hover:text-green-800"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default { CreateMappingForm, MappingsList, AutoMappingNotification };