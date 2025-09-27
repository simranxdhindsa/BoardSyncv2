// Updated UserSettings Component - components/settings/UserSettings.js
import React, { useState, useEffect } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { 
  getUserSettings, 
  updateUserSettings, 
  getAsanaProjects, 
  getYouTrackProjects,
  testConnections
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
  Plus,
  X,
  Eye,
  EyeOff
} from 'lucide-react';
import FluidText from '../FluidText';
import '../../styles/settings-glass-theme.css';

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
  
  // Password visibility state
  const [showPasswords, setShowPasswords] = useState({
    asana_pat: false,
    youtrack_token: false
  });

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

  // Toggle password visibility
  const togglePasswordVisibility = (field) => {
    setShowPasswords(prev => ({
      ...prev,
      [field]: !prev[field]
    }));
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
        asana: !!results.asana,
        youtrack: !!results.youtrack
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
      <div className="settings-container">
        <div className="min-h-screen flex items-center justify-center">
          <div className="flex items-center">
            <RefreshCw className="settings-spinner" />
            <span>Loading settings...</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="settings-container">
      <div className="max-w-4xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="settings-header">
          <div className="flex items-center justify-between">
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
                  className="settings-button-secondary"
                >
                  Back to Dashboard
                </button>
              )}
              <button
                onClick={handleLogout}
                className="settings-button-secondary"
                style={{ 
                  background: 'rgba(239, 68, 68, 0.15)',
                  borderColor: 'rgba(239, 68, 68, 0.3)',
                  color: '#991b1b'
                }}
              >
                <LogOut className="w-4 h-4 mr-2" />
                Logout
              </button>
            </div>
          </div>
        </div>

        {/* Messages */}
        {error && (
          <div className="settings-error">
            <div className="flex items-center">
              <AlertTriangle className="w-5 h-5 mr-2" />
              <p>{error}</p>
            </div>
          </div>
        )}

        {successMessage && (
          <div className="settings-success">
            <div className="flex items-center">
              <CheckCircle className="w-5 h-5 mr-2" />
              <p>{successMessage}</p>
            </div>
          </div>
        )}

        {/* Tab Navigation */}
        <div className="settings-tabs">
          <div className="settings-tab-border flex">
            {tabs.map(tab => {
              const IconComponent = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`settings-tab ${activeTab === tab.id ? 'active' : ''}`}
                >
                  <IconComponent className="w-4 h-4 mr-2" />
                  {tab.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Tab Content */}
        <div className="settings-content">
          {activeTab === 'api' && (
            <div className="space-y-6">
              <div>
                <FluidText className="settings-section-header" sensitivity={1.2}>
                  API Configuration
                </FluidText>
                <p className="settings-section-description">
                  Enter your API credentials to enable synchronization between Asana and YouTrack
                </p>
              </div>

              {/* Asana Configuration */}
              <div className="settings-form-group">
                <h3 className="text-lg font-medium text-gray-900 mb-4">Asana Settings</h3>
                
                <div className="mb-4">
                  <label className="settings-label">
                    Personal Access Token (PAT)
                  </label>
                  <div className="auth-input-container">
                    <input
                      type={showPasswords.asana_pat ? 'text' : 'password'}
                      value={settings.asana_pat}
                      onChange={handleInputChange('asana_pat')}
                      placeholder="Enter your Asana PAT"
                      className="settings-input"
                      style={{ paddingRight: '3rem' }}
                    />
                    <button
                      type="button"
                      onClick={() => togglePasswordVisibility('asana_pat')}
                      className="auth-input-toggle"
                      style={{ 
                        position: 'absolute',
                        right: '0.75rem',
                        top: '50%',
                        transform: 'translateY(-50%)',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        zIndex: 13
                      }}
                    >
                      {showPasswords.asana_pat ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>

                <div className="settings-form-row">
                  <div className="flex-1">
                    <label className="settings-label">
                      Project
                    </label>
                    <select
                      value={settings.asana_project_id}
                      onChange={handleInputChange('asana_project_id')}
                      className="settings-select"
                      disabled={asanaProjects.length === 0}
                    >
                      <option value="">Select Asana Project</option>
                      {asanaProjects.map(project => (
                        <option key={project.id} value={project.id}>
                          {project.name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <button
                    onClick={loadAsanaProjects}
                    disabled={loadingProjects.asana || !settings.asana_pat}
                    className="settings-button-secondary"
                  >
                    {loadingProjects.asana ? (
                      <RefreshCw className="settings-spinner" />
                    ) : (
                      <RefreshCw className="w-4 h-4" />
                    )}
                  </button>
                </div>
              </div>

              {/* YouTrack Configuration */}
              <div className="settings-form-group">
                <div className="settings-divider"></div>
                <h3 className="text-lg font-medium text-gray-900 mb-4">YouTrack Settings</h3>
                
                <div className="mb-4">
                  <label className="settings-label">
                    Base URL
                  </label>
                  <input
                    type="url"
                    value={settings.youtrack_base_url}
                    onChange={handleInputChange('youtrack_base_url')}
                    placeholder="https://your-instance.youtrack.cloud"
                    className="settings-input"
                  />
                </div>

                <div className="mb-4">
                  <label className="settings-label">
                    API Token
                  </label>
                  <div className="auth-input-container">
                    <input
                      type={showPasswords.youtrack_token ? 'text' : 'password'}
                      value={settings.youtrack_token}
                      onChange={handleInputChange('youtrack_token')}
                      placeholder="Enter your YouTrack API token"
                      className="settings-input"
                      style={{ paddingRight: '3rem' }}
                    />
                    <button
                      type="button"
                      onClick={() => togglePasswordVisibility('youtrack_token')}
                      className="auth-input-toggle"
                      style={{ 
                        position: 'absolute',
                        right: '0.75rem',
                        top: '50%',
                        transform: 'translateY(-50%)',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        zIndex: 13
                      }}
                    >
                      {showPasswords.youtrack_token ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>

                <div className="settings-form-row">
                  <div className="flex-1">
                    <label className="settings-label">
                      Project
                    </label>
                    <select
                      value={settings.youtrack_project_id}
                      onChange={handleInputChange('youtrack_project_id')}
                      className="settings-select"
                      disabled={youtrackProjects.length === 0}
                    >
                      <option value="">Select YouTrack Project</option>
                      {youtrackProjects.map(project => (
                        <option key={project.id} value={project.id}>
                          {project.name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <button
                    onClick={loadYoutrackProjects}
                    disabled={loadingProjects.youtrack || !settings.youtrack_base_url || !settings.youtrack_token}
                    className="settings-button-secondary"
                  >
                    {loadingProjects.youtrack ? (
                      <RefreshCw className="settings-spinner" />
                    ) : (
                      <RefreshCw className="w-4 h-4" />
                    )}
                  </button>
                </div>
              </div>

              {/* Connection Test */}
              <div className="settings-form-group">
                <div className="settings-divider"></div>
                <button
                  onClick={handleTestConnections}
                  disabled={testing || (!settings.asana_pat || !settings.youtrack_base_url || !settings.youtrack_token)}
                  className="settings-button"
                >
                  {testing ? (
                    <>
                      <RefreshCw className="settings-spinner" />
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
                    <div className={`settings-connection-status ${
                      connectionStatus.asana ? 'settings-connection-success' : 'settings-connection-error'
                    }`}>
                      {connectionStatus.asana ? (
                        <CheckCircle className="w-5 h-5 mr-2" />
                      ) : (
                        <AlertTriangle className="w-5 h-5 mr-2" />
                      )}
                      Asana: {connectionStatus.asana ? 'Connected' : 'Failed'}
                    </div>
                    <div className={`settings-connection-status ${
                      connectionStatus.youtrack ? 'settings-connection-success' : 'settings-connection-error'
                    }`}>
                      {connectionStatus.youtrack ? (
                        <CheckCircle className="w-5 h-5 mr-2" />
                      ) : (
                        <AlertTriangle className="w-5 h-5 mr-2" />
                      )}
                      YouTrack: {connectionStatus.youtrack ? 'Connected' : 'Failed'}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'mapping' && (
            <div className="space-y-6">
              <div>
                <FluidText className="settings-section-header" sensitivity={1.2}>
                  Custom Field Mapping
                </FluidText>
                <p className="settings-section-description">
                  Configure how fields are mapped between Asana and YouTrack during synchronization
                </p>
              </div>

              {/* Add New Mapping */}
              <div className="settings-mapping-container">
                <h4 className="text-sm font-medium text-gray-900 mb-3">Add New Mapping</h4>
                <div className="settings-form-row">
                  <select
                    value={newMapping.type}
                    onChange={(e) => setNewMapping(prev => ({ ...prev, type: e.target.value }))}
                    className="settings-select"
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
                    className="settings-input"
                  />
                  
                  <input
                    type="text"
                    placeholder="Target field"
                    value={newMapping.value}
                    onChange={(e) => setNewMapping(prev => ({ ...prev, value: e.target.value }))}
                    className="settings-input"
                  />
                  
                  <button
                    onClick={addCustomMapping}
                    disabled={!newMapping.key.trim() || !newMapping.value.trim()}
                    className="settings-button"
                  >
                    <Plus className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Existing Mappings */}
              {Object.entries(settings.custom_field_mappings || {}).map(([mappingType, mappings]) => (
                <div key={mappingType} className="space-y-3">
                  <h4 className="text-sm font-medium text-gray-900 capitalize">
                    {mappingType.replace('_', ' ')}
                  </h4>
                  {!mappings || Object.keys(mappings).length === 0 ? (
                    <p className="text-gray-500 text-sm italic">No mappings configured</p>
                  ) : (
                    <div className="space-y-2">
                      {Object.entries(mappings || {}).map(([key, value]) => (
                        <div key={key} className="settings-mapping-item">
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
                <FluidText className="settings-section-header" sensitivity={1.2}>
                  Profile Settings
                </FluidText>
                <p className="settings-section-description">
                  Manage your account information and security settings
                </p>
              </div>

              {/* User Info */}
              <div className="settings-form-group">
                <div className="flex items-center space-x-4">
                  <div className="settings-profile-avatar">
                    <User className="w-8 h-8 text-blue-600" />
                  </div>
                  <div className="settings-profile-info">
                    <h3 className="settings-profile-name">{user?.username}</h3>
                    <p className="settings-profile-email">{user?.email}</p>
                    <p className="settings-profile-date">
                      Member since {new Date(user?.created_at || Date.now()).toLocaleDateString()}
                    </p>
                  </div>
                </div>
              </div>

              {/* Security Section */}
              <div className="settings-form-group">
                <div className="settings-divider"></div>
                <h4 className="text-lg font-medium text-gray-900 flex items-center mb-4">
                  <Shield className="w-5 h-5 mr-2" />
                  Security
                </h4>
                
                <button
                  onClick={() => alert('Change password feature coming soon!')}
                  className="settings-button-secondary"
                >
                  <Key className="w-4 h-4 mr-2" />
                  Change Password
                </button>
              </div>

              {/* Account Actions */}
              <div className="settings-form-group">
                <div className="settings-divider"></div>
                <h4 className="text-lg font-medium text-gray-900 mb-4">Account Actions</h4>
                
                <div className="space-y-3">
                  <button
                    onClick={handleLogout}
                    className="settings-button-secondary"
                    style={{ 
                      background: 'rgba(239, 68, 68, 0.15)',
                      borderColor: 'rgba(239, 68, 68, 0.3)',
                      color: '#991b1b'
                    }}
                  >
                    <LogOut className="w-4 h-4 mr-2" />
                    Logout
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Save Button (always visible) */}
          <div className="settings-actions">
            <button
              onClick={handleSaveSettings}
              disabled={saving}
              className="settings-button"
            >
              {saving ? (
                <>
                  <RefreshCw className="settings-spinner" />
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
    </div>
  );
};

export default UserSettings;