import React, { useState, useEffect } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { 
  getUserSettings, 
  updateUserSettings, 
  getAsanaProjects, 
  getYouTrackProjects,
  testConnections,
  logout 
} from '../../services/api';
import { 
  Settings, 
  Key, 
  Link, 
  TestTube, 
  Save, 
  RefreshCw, 
  CheckCircle, 
  AlertTriangle,
  LogOut,
  User,
  Shield,
  Zap,
  Plus,
  X
} from 'lucide-react';
import FluidText from '../FluidText';

const UserSettings = ({ onBack }) => {
  const { user, logout: authLogout } = useAuth();
  
  // Settings state
  const [settings, setSettings] = useState({
    asana_pat: '',
    youtrack_base_url: '',
    youtrack_token: '',
    asana_project_id: '',
    youtrack_project_id: '',
    custom_field_mappings: {
      tag_mapping: {},
      priority_mapping: {},
      status_mapping: {},
      custom_fields: {}
    }
  });

  // UI state
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState(null);
  const [successMessage, setSuccessMessage] = useState('');
  
  // Projects state
  const [asanaProjects, setAsanaProjects] = useState([]);
  const [youtrackProjects, setYoutrackProjects] = useState([]);
  const [loadingProjects, setLoadingProjects] = useState({ asana: false, youtrack: false });
  
  // Connection test state
  const [connectionStatus, setConnectionStatus] = useState({ asana: null, youtrack: null });
  
  // Field mapping state
  const [newMapping, setNewMapping] = useState({ type: 'tag_mapping', key: '', value: '' });
  const [activeTab, setActiveTab] = useState('api');

  // Load user settings on mount
  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await getUserSettings();
      const userSettings = response.data || response;
      
      setSettings({
        asana_pat: userSettings.asana_pat || '',
        youtrack_base_url: userSettings.youtrack_base_url || '',
        youtrack_token: userSettings.youtrack_token || '',
        asana_project_id: userSettings.asana_project_id || '',
        youtrack_project_id: userSettings.youtrack_project_id || '',
        custom_field_mappings: userSettings.custom_field_mappings || {
          tag_mapping: {},
          priority_mapping: {},
          status_mapping: {},
          custom_fields: {}
        }
      });
    } catch (err) {
      setError('Failed to load settings: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (field) => (e) => {
    setSettings(prev => ({
      ...prev,
      [field]: e.target.value
    }));
    clearMessages();
  };

  const clearMessages = () => {
    setError(null);
    setSuccessMessage('');
  };

  // Load Asana projects
  const loadAsanaProjects = async () => {
    if (!settings.asana_pat) {
      setError('Please enter your Asana PAT first');
      return;
    }

    setLoadingProjects(prev => ({ ...prev, asana: true }));
    
    try {
      const response = await getAsanaProjects();
      setAsanaProjects(response.data || response);
    } catch (err) {
      setError('Failed to load Asana projects: ' + err.message);
    } finally {
      setLoadingProjects(prev => ({ ...prev, asana: false }));
    }
  };

  // Load YouTrack projects
  const loadYoutrackProjects = async () => {
    if (!settings.youtrack_base_url || !settings.youtrack_token) {
      setError('Please enter your YouTrack URL and token first');
      return;
    }

    setLoadingProjects(prev => ({ ...prev, youtrack: true }));
    
    try {
      const response = await getYouTrackProjects();
      setYoutrackProjects(response.data || response);
    } catch (err) {
      setError('Failed to load YouTrack projects: ' + err.message);
    } finally {
      setLoadingProjects(prev => ({ ...prev, youtrack: false }));
    }
  };

  // Test API connections
  const handleTestConnections = async () => {
    setTesting(true);
    setConnectionStatus({ asana: null, youtrack: null });
    clearMessages();
    
    try {
      const response = await testConnections();
      const results = response.data || response.results || response;
      
      setConnectionStatus({
        asana: results.asana || false,
        youtrack: results.youtrack || false
      });
      
      if (results.asana && results.youtrack) {
        setSuccessMessage('All connections successful!');
      } else {
        setError('Some connections failed. Please check your credentials.');
      }
    } catch (err) {
      setError('Connection test failed: ' + err.message);
      setConnectionStatus({ asana: false, youtrack: false });
    } finally {
      setTesting(false);
    }
  };

  // Save settings
  const handleSaveSettings = async () => {
    setSaving(true);
    clearMessages();
    
    try {
      const response = await updateUserSettings(settings);
      setSuccessMessage('Settings saved successfully!');
      
      // Auto-load projects after saving credentials
      if (settings.asana_pat && asanaProjects.length === 0) {
        setTimeout(loadAsanaProjects, 500);
      }
      if (settings.youtrack_base_url && settings.youtrack_token && youtrackProjects.length === 0) {
        setTimeout(loadYoutrackProjects, 500);
      }
    } catch (err) {
      setError('Failed to save settings: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  // Add custom field mapping
  const addCustomMapping = () => {
    if (!newMapping.key.trim() || !newMapping.value.trim()) return;
    
    setSettings(prev => ({
      ...prev,
      custom_field_mappings: {
        ...prev.custom_field_mappings,
        [newMapping.type]: {
          ...prev.custom_field_mappings[newMapping.type],
          [newMapping.key.trim()]: newMapping.value.trim()
        }
      }
    }));
    
    setNewMapping({ ...newMapping, key: '', value: '' });
  };

  // Remove custom mapping
  const removeCustomMapping = (type, key) => {
    setSettings(prev => {
      const updatedMappings = { ...prev.custom_field_mappings[type] };
      delete updatedMappings[key];
      
      return {
        ...prev,
        custom_field_mappings: {
          ...prev.custom_field_mappings,
          [type]: updatedMappings
        }
      };
    });
  };

  // Handle logout
  const handleLogout = async () => {
    if (window.confirm('Are you sure you want to logout?')) {
      try {
        await authLogout();
        window.location.reload();
      } catch (err) {
        console.error('Logout failed:', err);
      }
    }
  };

  const tabs = [
    { id: 'api', label: 'API Configuration', icon: Key },
    { id: 'mapping', label: 'Field Mapping', icon: Link },
    { id: 'profile', label: 'Profile', icon: User }
  ];

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex items-center">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" />
          <span>Loading settings...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <FluidText className="text-3xl font-bold text-gray-900 mb-2" sensitivity={2}>
            User Settings
          </FluidText>
          <p className="text-gray-600">
            Configure your API credentials and customize sync behavior
          </p>
        </div>
        
        <div className="flex items-center space-x-4">
          {onBack && (
            <button
              onClick={onBack}
              className="flex items-center bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors"
            >
              Back to Dashboard
            </button>
          )}
          <button
            onClick={handleLogout}
            className="flex items-center bg-red-100 text-red-700 px-4 py-2 rounded-lg hover:bg-red-200 transition-colors"
          >
            <LogOut className="w-4 h-4 mr-2" />
            Logout
          </button>
        </div>
      </div>

      {/* Messages */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
          <div className="flex items-center">
            <AlertTriangle className="w-5 h-5 text-red-600 mr-2" />
            <p className="text-red-800">{error}</p>
          </div>
        </div>
      )}

      {successMessage && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6">
          <div className="flex items-center">
            <CheckCircle className="w-5 h-5 text-green-600 mr-2" />
            <p className="text-green-800">{successMessage}</p>
          </div>
        </div>
      )}

      {/* Tab Navigation */}
      <div className="glass-panel bg-white border border-gray-200 rounded-lg mb-6">
        <div className="flex border-b border-gray-200">
          {tabs.map(tab => {
            const IconComponent = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center px-6 py-4 font-medium transition-colors ${
                  activeTab === tab.id
                    ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50'
                    : 'text-gray-700 hover:text-blue-600'
                }`}
              >
                <IconComponent className="w-4 h-4 mr-2" />
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab Content */}
      <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6">
        {activeTab === 'api' && (
          <div className="space-y-6">
            <div>
              <FluidText className="text-xl font-semibold text-gray-900 mb-4" sensitivity={1.2}>
                API Configuration
              </FluidText>
              <p className="text-gray-600 mb-6">
                Enter your API credentials to enable synchronization between Asana and YouTrack
              </p>
            </div>

            {/* Asana Configuration */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">Asana Settings</h3>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Personal Access Token (PAT)
                </label>
                <input
                  type="password"
                  value={settings.asana_pat}
                  onChange={handleInputChange('asana_pat')}
                  placeholder="Enter your Asana PAT"
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div className="flex space-x-4">
                <div className="flex-1">
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Project
                  </label>
                  <div className="flex space-x-2">
                    <select
                      value={settings.asana_project_id}
                      onChange={handleInputChange('asana_project_id')}
                      className="flex-1 px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      disabled={asanaProjects.length === 0}
                    >
                      <option value="">Select Asana Project</option>
                      {asanaProjects.map(project => (
                        <option key={project.id} value={project.id}>
                          {project.name}
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={loadAsanaProjects}
                      disabled={loadingProjects.asana || !settings.asana_pat}
                      className="px-4 py-3 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition-colors disabled:opacity-50"
                    >
                      {loadingProjects.asana ? (
                        <RefreshCw className="w-4 h-4 animate-spin" />
                      ) : (
                        <RefreshCw className="w-4 h-4" />
                      )}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* YouTrack Configuration */}
            <div className="space-y-4 pt-6 border-t border-gray-200">
              <h3 className="text-lg font-medium text-gray-900">YouTrack Settings</h3>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Base URL
                </label>
                <input
                  type="url"
                  value={settings.youtrack_base_url}
                  onChange={handleInputChange('youtrack_base_url')}
                  placeholder="https://your-instance.youtrack.cloud"
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  API Token
                </label>
                <input
                  type="password"
                  value={settings.youtrack_token}
                  onChange={handleInputChange('youtrack_token')}
                  placeholder="Enter your YouTrack API token"
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div className="flex space-x-4">
                <div className="flex-1">
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Project
                  </label>
                  <div className="flex space-x-2">
                    <select
                      value={settings.youtrack_project_id}
                      onChange={handleInputChange('youtrack_project_id')}
                      className="flex-1 px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      disabled={youtrackProjects.length === 0}
                    >
                      <option value="">Select YouTrack Project</option>
                      {youtrackProjects.map(project => (
                        <option key={project.id} value={project.id}>
                          {project.name}
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={loadYoutrackProjects}
                      disabled={loadingProjects.youtrack || !settings.youtrack_base_url || !settings.youtrack_token}
                      className="px-4 py-3 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition-colors disabled:opacity-50"
                    >
                      {loadingProjects.youtrack ? (
                        <RefreshCw className="w-4 h-4 animate-spin" />
                      ) : (
                        <RefreshCw className="w-4 h-4" />
                      )}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* Connection Test */}
            <div className="pt-6 border-t border-gray-200">
              <button
                onClick={handleTestConnections}
                disabled={testing || (!settings.asana_pat || !settings.youtrack_base_url || !settings.youtrack_token)}
                className="flex items-center bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                {testing ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    Testing Connections...
                  </>
                ) : (
                  <>
                    <TestTube className="w-4 h-4 mr-2" />
                    Test Connections
                  </>
                )}
              </button>

              {(connectionStatus.asana !== null || connectionStatus.youtrack !== null) && (
                <div className="mt-4 space-y-2">
                  <div className="flex items-center">
                    {connectionStatus.asana ? (
                      <CheckCircle className="w-5 h-5 text-green-600 mr-2" />
                    ) : (
                      <AlertTriangle className="w-5 h-5 text-red-600 mr-2" />
                    )}
                    <span className={connectionStatus.asana ? 'text-green-800' : 'text-red-800'}>
                      Asana: {connectionStatus.asana ? 'Connected' : 'Failed'}
                    </span>
                  </div>
                  <div className="flex items-center">
                    {connectionStatus.youtrack ? (
                      <CheckCircle className="w-5 h-5 text-green-600 mr-2" />
                    ) : (
                      <AlertTriangle className="w-5 h-5 text-red-600 mr-2" />
                    )}
                    <span className={connectionStatus.youtrack ? 'text-green-800' : 'text-red-800'}>
                      YouTrack: {connectionStatus.youtrack ? 'Connected' : 'Failed'}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 'mapping' && (
          <div className="space-y-6">
            <div>
              <FluidText className="text-xl font-semibold text-gray-900 mb-4" sensitivity={1.2}>
                Custom Field Mapping
              </FluidText>
              <p className="text-gray-600 mb-6">
                Configure how fields are mapped between Asana and YouTrack during synchronization
              </p>
            </div>

            {/* Add New Mapping */}
            <div className="glass-panel bg-gray-50 border border-gray-200 rounded-lg p-4">
              <h4 className="text-sm font-medium text-gray-900 mb-3">Add New Mapping</h4>
              <div className="flex space-x-3">
                <select
                  value={newMapping.type}
                  onChange={(e) => setNewMapping(prev => ({ ...prev, type: e.target.value }))}
                  className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="tag_mapping">Tag Mapping</option>
                  <option value="priority_mapping">Priority Mapping</option>
                  <option value="status_mapping">Status Mapping</option>
                  <option value="custom_fields">Custom Fields</option>
                </select>
                
                <input
                  type="text"
                  placeholder="Source field"
                  value={newMapping.key}
                  onChange={(e) => setNewMapping(prev => ({ ...prev, key: e.target.value }))}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                
                <input
                  type="text"
                  placeholder="Target field"
                  value={newMapping.value}
                  onChange={(e) => setNewMapping(prev => ({ ...prev, value: e.target.value }))}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                
                <button
                  onClick={addCustomMapping}
                  disabled={!newMapping.key.trim() || !newMapping.value.trim()}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
                >
                  <Plus className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Existing Mappings */}
            {Object.entries(settings.custom_field_mappings).map(([mappingType, mappings]) => (
              <div key={mappingType} className="space-y-3">
                <h4 className="text-sm font-medium text-gray-900 capitalize">
                  {mappingType.replace('_', ' ')}
                </h4>
                {Object.keys(mappings).length === 0 ? (
                  <p className="text-gray-500 text-sm italic">No mappings configured</p>
                ) : (
                  <div className="space-y-2">
                    {Object.entries(mappings).map(([key, value]) => (
                      <div key={key} className="flex items-center justify-between bg-gray-50 rounded-lg px-4 py-2">
                        <div className="flex items-center space-x-3">
                          <span className="font-medium text-gray-900">{key}</span>
                          <span className="text-gray-400">→</span>
                          <span className="text-gray-700">{value}</span>
                        </div>
                        <button
                          onClick={() => removeCustomMapping(mappingType, key)}
                          className="text-red-600 hover:text-red-800 transition-colors"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {activeTab === 'profile' && (
          <div className="space-y-6">
            <div>
              <FluidText className="text-xl font-semibold text-gray-900 mb-4" sensitivity={1.2}>
                Profile Settings
              </FluidText>
              <p className="text-gray-600 mb-6">
                Manage your account information and security settings
              </p>
            </div>

            {/* User Info */}
            <div className="space-y-4">
              <div className="flex items-center space-x-4">
                <div className="w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center">
                  <User className="w-8 h-8 text-blue-600" />
                </div>
                <div>
                  <h3 className="text-lg font-medium text-gray-900">{user?.username}</h3>
                  <p className="text-gray-600">{user?.email}</p>
                  <p className="text-sm text-gray-500">
                    Member since {new Date(user?.created_at).toLocaleDateString()}
                  </p>
                </div>
              </div>
            </div>

            {/* Security Section */}
            <div className="pt-6 border-t border-gray-200 space-y-4">
              <h4 className="text-lg font-medium text-gray-900 flex items-center">
                <Shield className="w-5 h-5 mr-2" />
                Security
              </h4>
              
              <button
                onClick={() => {/* TODO: Implement change password modal */}}
                className="flex items-center bg-blue-100 text-blue-700 px-4 py-2 rounded-lg hover:bg-blue-200 transition-colors"
              >
                <Key className="w-4 h-4 mr-2" />
                Change Password
              </button>
            </div>

            {/* Account Actions */}
            <div className="pt-6 border-t border-gray-200 space-y-4">
              <h4 className="text-lg font-medium text-gray-900">Account Actions</h4>
              
              <div className="space-y-3">
                <button
                  onClick={handleLogout}
                  className="flex items-center bg-red-100 text-red-700 px-4 py-2 rounded-lg hover:bg-red-200 transition-colors"
                >
                  <LogOut className="w-4 h-4 mr-2" />
                  Logout
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Save Button (always visible) */}
        <div className="pt-6 border-t border-gray-200 flex justify-end space-x-4">
          <button
            onClick={handleSaveSettings}
            disabled={saving}
            className="flex items-center bg-green-600 text-white px-6 py-3 rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50 font-medium"
          >
            {saving ? (
              <>
                <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="w-4 h-4 mr-2" />
                Save Settings
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export default UserSettings