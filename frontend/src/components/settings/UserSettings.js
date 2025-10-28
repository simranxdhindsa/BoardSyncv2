// FILE: frontend/src/components/settings/UserSettings.js
// Complete UserSettings Component with ALL functionality

import React, { useState, useEffect } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import {
  getUserSettings,
  updateUserSettings,
  getAsanaProjects,
  getYouTrackProjects,
  getYouTrackBoards,
  testConnections,
  changePassword,
  deleteAccount
} from '../../services/api';
import { CreateMappingForm, MappingsList } from '../mapping/MappingComponents';
import ColumnMappingSettings from './ColumnMappingSettings';
import {
  Settings,
  Key,
  Link,
  Link2,
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
  EyeOff,
  Trash2,
  AlertCircle,
  Columns
} from 'lucide-react';
import FluidText from '../FluidText';
import '../../styles/settings-glass-theme.css';

const UserSettings = ({ onBack }) => {
  const { user, logout: authLogout } = useAuth();
  
  const [settings, setSettings] = useState({
    asana_pat: '',
    youtrack_base_url: '',
    youtrack_token: '',
    asana_project_id: '',
    youtrack_project_id: '',
    youtrack_board_id: '',
    custom_field_mappings: {
      tag_mapping: {},
      priority_mapping: {},
      status_mapping: {},
      custom_fields: {}
    },
    column_mappings: {
      asana_to_youtrack: [],
      youtrack_to_asana: []
    }
  });

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState(null);
  const [successMessage, setSuccessMessage] = useState('');
  
  const [asanaProjects, setAsanaProjects] = useState([]);
  const [youtrackProjects, setYoutrackProjects] = useState([]);
  const [youtrackBoards, setYoutrackBoards] = useState([]);
  const [loadingProjects, setLoadingProjects] = useState({ asana: false, youtrack: false, youtrackBoards: false });
  
  const [connectionStatus, setConnectionStatus] = useState({ asana: null, youtrack: null });
  
  const [newMapping, setNewMapping] = useState({ type: 'tag_mapping', key: '', value: '' });
  const [activeTab, setActiveTab] = useState('api');
  
  const [showPasswords, setShowPasswords] = useState({
    asana_pat: false,
    youtrack_token: false
  });

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deletionData, setDeletionData] = useState({
    password: '',
    confirmation: ''
  });
  const [isDeletingAccount, setIsDeletingAccount] = useState(false);

  const [showChangePasswordModal, setShowChangePasswordModal] = useState(false);
  const [passwordData, setPasswordData] = useState({
    oldPassword: '',
    newPassword: '',
    confirmPassword: ''
  });
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  const [mappingRefreshKey, setMappingRefreshKey] = useState(0);
  const [showCreateMappingForm, setShowCreateMappingForm] = useState(false);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [initialSettings, setInitialSettings] = useState(null);

  useEffect(() => {
    loadSettings();
  }, []);

  // Auto-load projects when settings are loaded and API tab is active
  useEffect(() => {
    if (activeTab === 'api' && settings.asana_pat && asanaProjects.length === 0 && !loadingProjects.asana) {
      loadAsanaProjects();
    }
  }, [activeTab, settings.asana_pat]);

  useEffect(() => {
    if (activeTab === 'api' && settings.youtrack_base_url && settings.youtrack_token && youtrackProjects.length === 0 && !loadingProjects.youtrack) {
      loadYoutrackProjects();
    }
  }, [activeTab, settings.youtrack_base_url, settings.youtrack_token]);

  const loadSettings = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await getUserSettings();
      const userSettings = response.data || response;

      const loadedSettings = {
        asana_pat: userSettings.asana_pat || '',
        youtrack_base_url: userSettings.youtrack_base_url || '',
        youtrack_token: userSettings.youtrack_token || '',
        asana_project_id: userSettings.asana_project_id || '',
        youtrack_project_id: userSettings.youtrack_project_id || '',
        youtrack_board_id: userSettings.youtrack_board_id || '',
        custom_field_mappings: userSettings.custom_field_mappings || {
          tag_mapping: {},
          priority_mapping: {},
          status_mapping: {},
          custom_fields: {}
        },
        column_mappings: userSettings.column_mappings || {
          asana_to_youtrack: [],
          youtrack_to_asana: []
        }
      };

      setSettings(loadedSettings);
      setInitialSettings(loadedSettings);
      setHasUnsavedChanges(false);
    } catch (err) {
      setError('Failed to load settings: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const togglePasswordVisibility = (field) => {
    setShowPasswords(prev => ({
      ...prev,
      [field]: !prev[field]
    }));
  };

  const handleInputChange = (field) => (e) => {
    const newSettings = {
      ...settings,
      [field]: e.target.value
    };
    setSettings(newSettings);

    // Check if API configuration has changed
    if (initialSettings) {
      const apiFieldsChanged =
        newSettings.asana_pat !== initialSettings.asana_pat ||
        newSettings.youtrack_base_url !== initialSettings.youtrack_base_url ||
        newSettings.youtrack_token !== initialSettings.youtrack_token ||
        newSettings.asana_project_id !== initialSettings.asana_project_id ||
        newSettings.youtrack_project_id !== initialSettings.youtrack_project_id;

      setHasUnsavedChanges(apiFieldsChanged);

      // Reset connection status when settings change
      if (apiFieldsChanged) {
        setConnectionStatus({ asana: null, youtrack: null });
      }
    }

    clearMessages();
  };

  const clearMessages = () => {
    setError(null);
    setSuccessMessage('');
  };

  const loadAsanaProjects = async () => {
    if (!settings.asana_pat) {
      setError('Please enter your Asana PAT first');
      return;
    }

    setLoadingProjects(prev => ({ ...prev, asana: true }));
    clearMessages();

    try {
      await updateUserSettings(settings);
      setInitialSettings(settings); // Update initial settings after save
      setHasUnsavedChanges(false); // Reset flag
      const response = await getAsanaProjects();
      setAsanaProjects(response.data || response);
      setSuccessMessage('Asana credentials saved and projects loaded successfully!');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setError('Failed to load Asana projects: ' + err.message);
    } finally {
      setLoadingProjects(prev => ({ ...prev, asana: false }));
    }
  };

  const loadYoutrackProjects = async () => {
    if (!settings.youtrack_base_url || !settings.youtrack_token) {
      setError('Please enter your YouTrack URL and token first');
      return;
    }

    setLoadingProjects(prev => ({ ...prev, youtrack: true }));
    clearMessages();

    try {
      await updateUserSettings(settings);
      setInitialSettings(settings); // Update initial settings after save
      setHasUnsavedChanges(false); // Reset flag
      const response = await getYouTrackProjects();
      setYoutrackProjects(response.data || response);
      setSuccessMessage('YouTrack credentials saved and projects loaded successfully!');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setError('Failed to load YouTrack projects: ' + err.message);
    } finally {
      setLoadingProjects(prev => ({ ...prev, youtrack: false }));
    }
  };

  const loadYoutrackBoards = async () => {
    if (!settings.youtrack_base_url || !settings.youtrack_token) {
      setError('Please enter your YouTrack URL and token first');
      return;
    }

    setLoadingProjects(prev => ({ ...prev, youtrackBoards: true }));
    clearMessages();

    try {
      await updateUserSettings(settings);
      setInitialSettings(settings);
      setHasUnsavedChanges(false);
      const response = await getYouTrackBoards();
      setYoutrackBoards(response.data || response);
      setSuccessMessage('YouTrack boards loaded successfully!');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setError('Failed to load YouTrack boards: ' + err.message);
    } finally {
      setLoadingProjects(prev => ({ ...prev, youtrackBoards: false }));
    }
  };

  const handleTestConnections = async () => {
    setTesting(true);
    setConnectionStatus({ asana: null, youtrack: null });
    clearMessages();

    try {
      // Call the API to test connections with current settings
      const response = await testConnections();
      const results = response.data?.results || response.results || {};

      // Frontend validation: Check if projects are loaded AND project IDs are selected
      const asanaSuccess = results.asana && asanaProjects.length > 0 && settings.asana_project_id;
      const youtrackSuccess = results.youtrack && youtrackProjects.length > 0 && settings.youtrack_project_id;

      setConnectionStatus({
        asana: asanaSuccess,
        youtrack: youtrackSuccess
      });

      if (asanaSuccess && youtrackSuccess) {
        setSuccessMessage('All connections successful! You can now save your settings.');
        setTimeout(() => setSuccessMessage(''), 3000);
      } else {
        const failedServices = [];
        if (!asanaSuccess) failedServices.push('Asana');
        if (!youtrackSuccess) failedServices.push('YouTrack');
        setError(`Connection test incomplete for: ${failedServices.join(', ')}. Please ensure credentials are valid, projects are loaded, and project IDs are selected.`);
      }
    } catch (err) {
      setError('Failed to test connections: ' + err.message);
      setConnectionStatus({ asana: false, youtrack: false });
    } finally {
      setTesting(false);
    }
  };

  
  const handleSaveSettings = async () => {
    setSaving(true);
    clearMessages();

    try {
      await updateUserSettings(settings);
      setInitialSettings(settings); // Update initial settings after successful save
      setHasUnsavedChanges(false); // Reset unsaved changes flag - this will disable the Save button
      // Keep connectionStatus as true so the Save button remains visible but disabled
      setSuccessMessage('Settings saved successfully!');
      setTimeout(() => setSuccessMessage(''), 3000); // Auto-dismiss after 3 seconds
    } catch (err) {
      setError('Failed to save settings: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  const addCustomMapping = async () => {
    if (!newMapping.key.trim() || !newMapping.value.trim()) return;

    const updatedSettings = {
      ...settings,
      custom_field_mappings: {
        ...settings.custom_field_mappings,
        [newMapping.type]: {
          ...settings.custom_field_mappings[newMapping.type],
          [newMapping.key.trim()]: newMapping.value.trim()
        }
      }
    };

    setSettings(updatedSettings);
    setNewMapping({ ...newMapping, key: '', value: '' });

    // Auto-save to backend
    try {
      await updateUserSettings(updatedSettings);
      setSuccessMessage('Field mapping added successfully!');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setError('Failed to save field mapping: ' + err.message);
    }
  };

  const removeCustomMapping = async (type, key) => {
    const updatedMappings = { ...settings.custom_field_mappings[type] };
    delete updatedMappings[key];

    const updatedSettings = {
      ...settings,
      custom_field_mappings: {
        ...settings.custom_field_mappings,
        [type]: updatedMappings
      }
    };

    setSettings(updatedSettings);

    // Auto-save to backend
    try {
      await updateUserSettings(updatedSettings);
      setSuccessMessage('Field mapping removed successfully!');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setError('Failed to remove field mapping: ' + err.message);
    }
  };

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

  const handleShowDeleteModal = () => {
    setError(null);
    setShowDeleteModal(true);
  };

  const handleDeleteAccount = async () => {
    if (deletionData.confirmation !== 'DELETE') {
      setError('Please type DELETE to confirm account deletion');
      return;
    }

    if (!deletionData.password) {
      setError('Please enter your password to confirm');
      return;
    }

    setIsDeletingAccount(true);
    setError(null);

    try {
      await deleteAccount({
        password: deletionData.password,
        confirmation: deletionData.confirmation
      });

      alert('Your account has been permanently deleted. You will be redirected to the login page.');
      window.location.href = '/';
    } catch (err) {
      setError('Failed to delete account: ' + err.message);
      setIsDeletingAccount(false);
    }
  };

  const closeDeleteModal = () => {
    setShowDeleteModal(false);
    setDeletionData({ password: '', confirmation: '' });
    setError(null);
  };

  const handleShowChangePasswordModal = () => {
    setError(null);
    setSuccessMessage('');
    setShowChangePasswordModal(true);
  };

  const handleChangePassword = async () => {
    // Validation
    if (!passwordData.oldPassword) {
      setError('Please enter your current password');
      return;
    }

    if (!passwordData.newPassword) {
      setError('Please enter a new password');
      return;
    }

    if (passwordData.newPassword.length < 6) {
      setError('New password must be at least 6 characters long');
      return;
    }

    if (passwordData.newPassword !== passwordData.confirmPassword) {
      setError('New passwords do not match');
      return;
    }

    setIsChangingPassword(true);
    setError(null);

    try {
      await changePassword(passwordData.oldPassword, passwordData.newPassword);

      setSuccessMessage('Password changed successfully!');
      setTimeout(() => setSuccessMessage(''), 3000);
      closeChangePasswordModal();
    } catch (err) {
      setError('Failed to change password: ' + err.message);
      setIsChangingPassword(false);
    }
  };

  const closeChangePasswordModal = () => {
    setShowChangePasswordModal(false);
    setPasswordData({
      oldPassword: '',
      newPassword: '',
      confirmPassword: ''
    });
    setError(null);
    setIsChangingPassword(false);
  };

  const handleMappingCreated = () => {
    setMappingRefreshKey(prev => prev + 1);
    setSuccessMessage('Ticket mapping created successfully!');
    setShowCreateMappingForm(false); // Return to list view after creating
  };

  const handleShowCreateForm = () => {
    setShowCreateMappingForm(true);
    clearMessages();
  };

  const handleCancelCreateForm = () => {
    setShowCreateMappingForm(false);
    clearMessages();
  };

  const handleTabChange = async (tabId) => {
    setActiveTab(tabId);

    // Reset to list view when switching to ticket mapping tab
    if (tabId === 'ticket_mapping') {
      setShowCreateMappingForm(false);
    }

    // Auto-load projects when API Configuration tab is opened
    if (tabId === 'api') {
      // Load Asana projects if PAT exists and projects not loaded
      if (settings.asana_pat && asanaProjects.length === 0) {
        await loadAsanaProjects();
      }

      // Load YouTrack projects if credentials exist and projects not loaded
      if (settings.youtrack_base_url && settings.youtrack_token && youtrackProjects.length === 0) {
        await loadYoutrackProjects();
      }
    }

    clearMessages();
  };

  const tabs = [
    { id: 'api', label: 'API Configuration', icon: Key },
    { id: 'column_mapping', label: 'Column Mapping', icon: Columns },
    { id: 'mapping', label: 'Field Mapping', icon: Link },
    { id: 'ticket_mapping', label: 'Ticket Mapping', icon: Link2 },
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
                className="delete-button-table"
              >
                <LogOut className="w-4 h-4 mr-2" />
                Logout
              </button>
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="settings-tabs">
          <div className="settings-tab-border flex">
            {tabs.map(tab => {
              const IconComponent = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => handleTabChange(tab.id)}
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
          {/* API Configuration Tab */}
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
                  <div className="settings-input-container">
                    <input
                      type={showPasswords.asana_pat ? 'text' : 'password'}
                      value={settings.asana_pat}
                      onChange={handleInputChange('asana_pat')}
                      placeholder="Enter your Asana PAT"
                      className="settings-input settings-input-with-icon"
                    />
                    <button
                      type="button"
                      onClick={() => togglePasswordVisibility('asana_pat')}
                      className="settings-input-toggle"
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
                  <div className="settings-input-container">
                    <input
                      type={showPasswords.youtrack_token ? 'text' : 'password'}
                      value={settings.youtrack_token}
                      onChange={handleInputChange('youtrack_token')}
                      placeholder="Enter your YouTrack API token"
                      className="settings-input settings-input-with-icon"
                    />
                    <button
                      type="button"
                      onClick={() => togglePasswordVisibility('youtrack_token')}
                      className="settings-input-toggle"
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

                <div className="settings-form-row">
                  <div className="flex-1">
                    <label className="settings-label">
                      Agile Board
                    </label>
                    <select
                      value={settings.youtrack_board_id}
                      onChange={handleInputChange('youtrack_board_id')}
                      className="settings-select"
                      disabled={youtrackBoards.length === 0}
                    >
                      <option value="">Select YouTrack Board</option>
                      {youtrackBoards.map(board => (
                        <option key={board.id} value={board.id}>
                          {board.name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <button
                    onClick={loadYoutrackBoards}
                    disabled={loadingProjects.youtrackBoards || !settings.youtrack_base_url || !settings.youtrack_token}
                    className="settings-button-secondary"
                  >
                    {loadingProjects.youtrackBoards ? (
                      <RefreshCw className="settings-spinner" />
                    ) : (
                      <RefreshCw className="w-4 h-4" />
                    )}
                  </button>
                </div>
              </div>

              {/* Connection Test / Save Button with Message on Right */}
              <div style={{ marginTop: '1.5rem', display: 'flex', alignItems: 'center', gap: '1.5rem' }}>
                {/* Dynamic Button: Test Connections or Save Settings */}
                {connectionStatus.asana && connectionStatus.youtrack ? (
                  // Show green Save Settings button when both connections are successful
                  <button
                    onClick={handleSaveSettings}
                    disabled={saving || !hasUnsavedChanges}
                    className="settings-button-success"
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
                ) : (
                  // Show blue Test Connection button by default
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
                )}

                {/* Success/Error Messages positioned to the right */}
                {successMessage && (
                  <div className="flex items-center" style={{ color: '#059669', fontSize: '0.95rem', fontWeight: '500' }}>
                    <CheckCircle className="w-5 h-5 mr-2" />
                    <p>{successMessage}</p>
                  </div>
                )}

                {error && (
                  <div className="flex items-center" style={{ color: '#991b1b', fontSize: '0.95rem', fontWeight: '500' }}>
                    <AlertTriangle className="w-5 h-5 mr-2" />
                    <p>{error}</p>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Column Mapping Tab */}
          {activeTab === 'column_mapping' && (
            <ColumnMappingSettings
              settings={settings}
              onSettingsUpdate={(updatedSettings) => {
                setSettings(updatedSettings);
                setInitialSettings(updatedSettings);
              }}
              onSuccess={setSuccessMessage}
              onError={setError}
            />
          )}

          {/* Field Mapping Tab */}
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

          {/* Ticket Mapping Tab */}
          {activeTab === 'ticket_mapping' && (
            <div className="space-y-6">
              <div>
                <FluidText className="settings-section-header" sensitivity={1.2}>
                  Ticket Mapping
                </FluidText>
                <p className="settings-section-description">
                  Manually link Asana tasks with YouTrack issues by pasting their URLs. Get task IDs for custom mapping.
                </p>
              </div>

              {/* Conditional View: List or Create Form */}
              {showCreateMappingForm ? (
                // Show Create Form
                <div>
                  <CreateMappingForm
                    onSuccess={handleMappingCreated}
                    onCancel={handleCancelCreateForm}
                  />

                  {/* Help Section */}
                  <div className="info-box mt-6">
                    <h4 className="text-sm font-medium mb-2 flex items-center">
                      <AlertCircle className="w-4 h-4 mr-1" />
                      When to use ticket mapping?
                    </h4>
                    <ul className="text-xs space-y-1">
                      <li>• When Asana and YouTrack ticket titles don't match</li>
                      <li>• For tickets created manually in YouTrack</li>
                      <li>• To link historical tickets created before automation</li>
                      <li>• When automatic matching fails due to special characters</li>
                    </ul>
                  </div>
                </div>
              ) : (
                // Show Mappings List
                <div>
                  <MappingsList
                    refreshTrigger={mappingRefreshKey}
                    onCreateNew={handleShowCreateForm}
                  />

                  {/* Help Section */}
                  <div className="info-box mt-6">
                    <h4 className="text-sm font-medium mb-2 flex items-center">
                      <AlertCircle className="w-4 h-4 mr-1" />
                      When to use ticket mapping?
                    </h4>
                    <ul className="text-xs space-y-1">
                      <li>• When Asana and YouTrack ticket titles don't match</li>
                      <li>• For tickets created manually in YouTrack</li>
                      <li>• To link historical tickets created before automation</li>
                      <li>• When automatic matching fails due to special characters</li>
                    </ul>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Profile Tab */}
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
                  onClick={handleShowChangePasswordModal}
                  className="settings-button-secondary"
                >
                  <Key className="w-4 h-4 mr-2" />
                  Change Password
                </button>
              </div>

              {/* Danger Zone */}
              <div className="settings-form-group">
                <div className="settings-divider"></div>
                <h4 className="text-lg font-medium text-red-900 flex items-center mb-4">
                  <AlertCircle className="w-5 h-5 mr-2" />
                  Delete Your Profile
                </h4>
                
                <div 
                  className="p-4 rounded-lg border-2 border-red-200"
                  style={{ background: 'rgba(239, 68, 68, 0.05)' }}
                >
                  <h5 className="font-medium text-gray-900 mb-2">Delete Account</h5>
                  <p className="text-sm text-gray-600 mb-4">
                    Permanently delete your account and all associated data. This action cannot be undone.
                  </p>
                  <button
                    onClick={handleShowDeleteModal}
                    className="delete-button-table"
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    Delete Account
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Change Password Modal */}
      {showChangePasswordModal && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
          onClick={closeChangePasswordModal}
          style={{ backdropFilter: 'blur(8px)' }}
        >
          <div
            className="glass-panel max-w-md w-full mx-4"
            style={{
              borderRadius: '20px',
              overflow: 'hidden'
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div
              className="p-6"
              style={{
                background: 'rgba(255, 255, 255, 0.3)',
                backdropFilter: 'blur(20px) saturate(150%)',
                WebkitBackdropFilter: 'blur(20px) saturate(150%)',
                borderBottom: '1px solid rgba(255, 255, 255, 0.3)'
              }}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="settings-profile-avatar" style={{ width: '3rem', height: '3rem' }}>
                    <Key className="w-5 h-5 text-blue-600" />
                  </div>
                  <h2 className="text-2xl font-bold text-gray-900">
                    Change Password
                  </h2>
                </div>
                <button
                  onClick={closeChangePasswordModal}
                  className="text-gray-400 hover:text-gray-600 transition-colors"
                  style={{
                    background: 'rgba(255, 255, 255, 0.3)',
                    borderRadius: '8px',
                    padding: '0.5rem',
                    border: '1px solid rgba(255, 255, 255, 0.4)'
                  }}
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>

            {/* Modal Body */}
            <div className="p-6 space-y-4">
              <div>
                <label className="settings-label">
                  Current Password
                </label>
                <input
                  type="password"
                  value={passwordData.oldPassword}
                  onChange={(e) => setPasswordData(prev => ({ ...prev, oldPassword: e.target.value }))}
                  placeholder="Enter current password"
                  className="settings-input"
                  disabled={isChangingPassword}
                />
              </div>

              <div>
                <label className="settings-label">
                  New Password
                </label>
                <input
                  type="password"
                  value={passwordData.newPassword}
                  onChange={(e) => setPasswordData(prev => ({ ...prev, newPassword: e.target.value }))}
                  placeholder="Enter new password (min 6 characters)"
                  className="settings-input"
                  disabled={isChangingPassword}
                />
              </div>

              <div>
                <label className="settings-label">
                  Confirm New Password
                </label>
                <input
                  type="password"
                  value={passwordData.confirmPassword}
                  onChange={(e) => setPasswordData(prev => ({ ...prev, confirmPassword: e.target.value }))}
                  placeholder="Re-enter new password"
                  className="settings-input"
                  disabled={isChangingPassword}
                />
              </div>

              {error && (
                <div className="error-box">
                  <p className="text-sm text-red-800">{error}</p>
                </div>
              )}

              {successMessage && (
                <div className="success-box">
                  <p className="text-sm text-green-800">{successMessage}</p>
                </div>
              )}
            </div>

            {/* Modal Footer */}
            <div
              className="p-6 flex justify-end space-x-3"
              style={{
                background: 'rgba(255, 255, 255, 0.2)',
                backdropFilter: 'blur(16px) saturate(150%)',
                WebkitBackdropFilter: 'blur(16px) saturate(150%)',
                borderTop: '1px solid rgba(255, 255, 255, 0.3)'
              }}
            >
              <button
                onClick={closeChangePasswordModal}
                disabled={isChangingPassword}
                className="settings-button-secondary"
              >
                Cancel
              </button>
              <button
                onClick={handleChangePassword}
                disabled={
                  isChangingPassword ||
                  !passwordData.oldPassword ||
                  !passwordData.newPassword ||
                  !passwordData.confirmPassword
                }
                className="settings-button"
              >
                {isChangingPassword ? (
                  <>
                    <RefreshCw className="settings-spinner" />
                    Changing Password...
                  </>
                ) : (
                  <>
                    <Key className="w-4 h-4 mr-2" />
                    Change Password
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Modal */}
      {showDeleteModal && (
        <div 
          className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
          onClick={closeDeleteModal}
        >
          <div 
            className="bg-white rounded-lg shadow-2xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="p-2 bg-red-100 rounded-lg">
                    <AlertCircle className="w-6 h-6 text-red-600" />
                  </div>
                  <h2 className="text-2xl font-bold text-gray-900">
                    Delete Account
                  </h2>
                </div>
                <button
                  onClick={closeDeleteModal}
                  className="text-gray-400 hover:text-gray-600 transition-colors"
                >
                  <X className="w-6 h-6" />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-6">
              <div className="bg-red-50 border-2 border-red-200 rounded-lg p-4">
                <div className="flex items-start space-x-3">
                  <AlertTriangle className="w-5 h-5 text-red-600 mt-0.5 flex-shrink-0" />
                  <div className="flex-1">
                    <h4 className="font-semibold text-red-900 mb-2">
                      Warning: This action is irreversible
                    </h4>
                    <ul className="text-sm text-red-800 space-y-1">
                      <li>• Your account will be permanently deleted</li>
                      <li>• All your settings and configurations will be lost</li>
                      <li>• Your sync history will be permanently removed</li>
                      <li>• All ignored tickets data will be deleted</li>
                      <li>• This action cannot be undone</li>
                    </ul>
                  </div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-900 mb-2">
                    Enter your password to confirm
                  </label>
                  <input
                    type="password"
                    value={deletionData.password}
                    onChange={(e) => setDeletionData(prev => ({ ...prev, password: e.target.value }))}
                    placeholder="Your password"
                    className="settings-input"
                    disabled={isDeletingAccount}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-900 mb-2">
                    Type <span className="font-bold text-red-600">DELETE</span> to confirm deletion
                  </label>
                  <input
                    type="text"
                    value={deletionData.confirmation}
                    onChange={(e) => setDeletionData(prev => ({ ...prev, confirmation: e.target.value }))}
                    placeholder="Type DELETE"
                    className="settings-input"
                    disabled={isDeletingAccount}
                  />
                </div>
              </div>

              {error && (
                <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                  <p className="text-sm text-red-800">{error}</p>
                </div>
              )}
            </div>

            <div className="p-6 border-t border-gray-200 flex justify-end space-x-3">
              <button
                onClick={closeDeleteModal}
                disabled={isDeletingAccount}
                className="settings-button-secondary"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteAccount}
                disabled={
                  isDeletingAccount ||
                  deletionData.confirmation !== 'DELETE' ||
                  !deletionData.password
                }
                className="multi-delete-button"
              >
                {isDeletingAccount ? (
                  <>
                    <RefreshCw className="settings-spinner" />
                    Deleting Account...
                  </>
                ) : (
                  <>
                    <Trash2 className="w-4 h-4 mr-2" />
                    Permanently Delete Account
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default UserSettings;