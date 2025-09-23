import { useState, useEffect } from 'react';
import { 
  getUserSettings, 
  updateUserSettings, 
  getAsanaProjects, 
  getYouTrackProjects,
  testConnections 
} from '../services/api';

export const useSettings = () => {
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
  const [error, setError] = useState(null);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  
  // Projects state
  const [asanaProjects, setAsanaProjects] = useState([]);
  const [youtrackProjects, setYoutrackProjects] = useState([]);
  const [projectsLoading, setProjectsLoading] = useState({ asana: false, youtrack: false });
  
  // Connection test state
  const [connectionStatus, setConnectionStatus] = useState({ asana: null, youtrack: null });
  const [lastTestTime, setLastTestTime] = useState(null);

  // Store original settings for comparison
  const [originalSettings, setOriginalSettings] = useState(null);

  // Load settings on hook initialization
  useEffect(() => {
    loadSettings();
  }, []);

  // Check for unsaved changes
  useEffect(() => {
    if (originalSettings) {
      const hasChanges = JSON.stringify(settings) !== JSON.stringify(originalSettings);
      setHasUnsavedChanges(hasChanges);
    }
  }, [settings, originalSettings]);

  // Load user settings from API
  const loadSettings = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await getUserSettings();
      const userSettings = response.data || response;
      
      const normalizedSettings = {
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
      };
      
      setSettings(normalizedSettings);
      setOriginalSettings(normalizedSettings);
      
      // Auto-load projects if credentials are available
      if (normalizedSettings.asana_pat) {
        loadAsanaProjects(false);
      }
      if (normalizedSettings.youtrack_base_url && normalizedSettings.youtrack_token) {
        loadYoutrackProjects(false);
      }
      
    } catch (err) {
      setError('Failed to load settings: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  // Update a specific setting
  const updateSetting = (key, value) => {
    setSettings(prev => ({
      ...prev,
      [key]: value
    }));
    setError(null);
  };

  // Update custom field mapping
  const updateCustomMapping = (mappingType, mappings) => {
    setSettings(prev => ({
      ...prev,
      custom_field_mappings: {
        ...prev.custom_field_mappings,
        [mappingType]: mappings
      }
    }));
  };

  // Add custom field mapping
  const addCustomMapping = (mappingType, key, value) => {
    if (!key.trim() || !value.trim()) return false;
    
    updateCustomMapping(mappingType, {
      ...settings.custom_field_mappings[mappingType],
      [key.trim()]: value.trim()
    });
    
    return true;
  };

  // Remove custom field mapping
  const removeCustomMapping = (mappingType, key) => {
    const updatedMappings = { ...settings.custom_field_mappings[mappingType] };
    delete updatedMappings[key];
    updateCustomMapping(mappingType, updatedMappings);
  };

  // Save settings to API
  const saveSettings = async () => {
    setSaving(true);
    setError(null);
    
    try {
      const response = await updateUserSettings(settings);
      setOriginalSettings(settings);
      setHasUnsavedChanges(false);
      return { success: true, message: 'Settings saved successfully!' };
    } catch (err) {
      const errorMessage = 'Failed to save settings: ' + err.message;
      setError(errorMessage);
      return { success: false, error: errorMessage };
    } finally {
      setSaving(false);
    }
  };

  // Load Asana projects
  const loadAsanaProjects = async (showError = true) => {
    if (!settings.asana_pat) {
      if (showError) setError('Please enter your Asana PAT first');
      return;
    }

    setProjectsLoading(prev => ({ ...prev, asana: true }));
    
    try {
      const response = await getAsanaProjects();
      const projects = response.data || response;
      setAsanaProjects(Array.isArray(projects) ? projects : []);
      return projects;
    } catch (err) {
      const errorMessage = 'Failed to load Asana projects: ' + err.message;
      if (showError) setError(errorMessage);
      setAsanaProjects([]);
      throw new Error(errorMessage);
    } finally {
      setProjectsLoading(prev => ({ ...prev, asana: false }));
    }
  };

  // Load YouTrack projects
  const loadYoutrackProjects = async (showError = true) => {
    if (!settings.youtrack_base_url || !settings.youtrack_token) {
      if (showError) setError('Please enter your YouTrack URL and token first');
      return;
    }

    setProjectsLoading(prev => ({ ...prev, youtrack: true }));
    
    try {
      const response = await getYouTrackProjects();
      const projects = response.data || response;
      setYoutrackProjects(Array.isArray(projects) ? projects : []);
      return projects;
    } catch (err) {
      const errorMessage = 'Failed to load YouTrack projects: ' + err.message;
      if (showError) setError(errorMessage);
      setYoutrackProjects([]);
      throw new Error(errorMessage);
    } finally {
      setProjectsLoading(prev => ({ ...prev, youtrack: false }));
    }
  };

  // Test API connections
  const testApiConnections = async () => {
    const requiredFields = {
      asana_pat: settings.asana_pat,
      youtrack_base_url: settings.youtrack_base_url,
      youtrack_token: settings.youtrack_token
    };

    const missingFields = Object.entries(requiredFields)
      .filter(([_, value]) => !value)
      .map(([key, _]) => key.replace('_', ' ').toUpperCase());

    if (missingFields.length > 0) {
      const errorMessage = `Missing required fields: ${missingFields.join(', ')}`;
      setError(errorMessage);
      return { success: false, error: errorMessage };
    }

    setConnectionStatus({ asana: null, youtrack: null });
    setError(null);
    
    try {
      const response = await testConnections();
      const results = response.data || response.results || response;
      
      const status = {
        asana: !!results.asana,
        youtrack: !!results.youtrack
      };
      
      setConnectionStatus(status);
      setLastTestTime(new Date());
      
      if (status.asana && status.youtrack) {
        return { 
          success: true, 
          message: 'All connections successful!',
          results: status
        };
      } else {
        const failedConnections = [];
        if (!status.asana) failedConnections.push('Asana');
        if (!status.youtrack) failedConnections.push('YouTrack');
        
        const errorMessage = `Failed connections: ${failedConnections.join(', ')}`;
        setError(errorMessage);
        return { 
          success: false, 
          error: errorMessage,
          results: status
        };
      }
    } catch (err) {
      const errorMessage = 'Connection test failed: ' + err.message;
      setError(errorMessage);
      setConnectionStatus({ asana: false, youtrack: false });
      return { success: false, error: errorMessage };
    }
  };

  // Reset settings to original values
  const resetSettings = () => {
    if (originalSettings) {
      setSettings(originalSettings);
      setHasUnsavedChanges(false);
      setError(null);
    }
  };

  // Check if settings are valid for sync operations
  const isConfigurationComplete = () => {
    return !!(
      settings.asana_pat &&
      settings.youtrack_base_url &&
      settings.youtrack_token &&
      settings.asana_project_id &&
      settings.youtrack_project_id
    );
  };

  // Get connection status summary
  const getConnectionSummary = () => {
    const { asana, youtrack } = connectionStatus;
    
    if (asana === null && youtrack === null) {
      return { status: 'untested', message: 'Connections not tested' };
    }
    
    if (asana && youtrack) {
      return { status: 'success', message: 'All connections successful' };
    }
    
    if (asana === false && youtrack === false) {
      return { status: 'error', message: 'All connections failed' };
    }
    
    const working = [];
    const failed = [];
    
    if (asana) working.push('Asana');
    else if (asana === false) failed.push('Asana');
    
    if (youtrack) working.push('YouTrack');
    else if (youtrack === false) failed.push('YouTrack');
    
    return { 
      status: 'partial', 
      message: `Working: ${working.join(', ')}${failed.length ? `, Failed: ${failed.join(', ')}` : ''}` 
    };
  };

  // Clear error message
  const clearError = () => {
    setError(null);
  };

  // Return hook interface
  return {
    // Settings data
    settings,
    originalSettings,
    hasUnsavedChanges,
    isConfigurationComplete: isConfigurationComplete(),
    
    // Projects data
    asanaProjects,
    youtrackProjects,
    
    // Connection status
    connectionStatus,
    connectionSummary: getConnectionSummary(),
    lastTestTime,
    
    // Loading states
    loading,
    saving,
    projectsLoading,
    
    // Error handling
    error,
    clearError,
    
    // Actions
    updateSetting,
    addCustomMapping,
    removeCustomMapping,
    updateCustomMapping,
    loadSettings,
    saveSettings,
    resetSettings,
    loadAsanaProjects,
    loadYoutrackProjects,
    testApiConnections
  };
};