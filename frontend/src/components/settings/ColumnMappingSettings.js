// FILE: frontend/src/components/settings/ColumnMappingSettings.js
// Column Mapping Configuration Component

import React, { useState, useEffect } from 'react';
import {
  getAsanaSections,
  getYouTrackStates,
  getYouTrackBoards,
  updateUserSettings
} from '../../services/api';
import {
  Plus,
  X,
  RefreshCw,
  AlertCircle,
  ArrowRight,
  Eye,
  CheckCircle
} from 'lucide-react';

const ColumnMappingSettings = ({
  settings,
  onSettingsUpdate,
  onSuccess,
  onError
}) => {
  const [asanaSections, setAsanaSections] = useState([]);
  const [youtrackStates, setYoutrackStates] = useState([]);
  const [youtrackBoards, setYoutrackBoards] = useState([]);
  const [loading, setLoading] = useState({ sections: false, states: false, boards: false });
  const [columnMappings, setColumnMappings] = useState([]);
  const [initialColumnMappings, setInitialColumnMappings] = useState([]);
  const [hasChanges, setHasChanges] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    // Initialize column mappings from settings
    const mappings = settings?.column_mappings?.asana_to_youtrack || [];
    setColumnMappings(mappings);
    setInitialColumnMappings(JSON.parse(JSON.stringify(mappings))); // Deep copy
    setHasChanges(false);

    // Auto-load columns when component mounts (without showing success messages)
    const loadColumnsAutomatically = async () => {
      // Load Asana sections if credentials are configured and not already loaded
      if (settings?.asana_pat && settings?.asana_project_id && asanaSections.length === 0) {
        await loadAsanaSections(false);
      }

      // Load YouTrack states if credentials are configured and not already loaded
      if (settings?.youtrack_base_url && settings?.youtrack_token && settings?.youtrack_project_id && youtrackStates.length === 0) {
        await loadYouTrackStates(false);
      }
    };

    loadColumnsAutomatically();
  }, [settings]);

  // Track changes to enable/disable save button
  useEffect(() => {
    const mappingsChanged = JSON.stringify(columnMappings) !== JSON.stringify(initialColumnMappings);
    setHasChanges(mappingsChanged);
  }, [columnMappings, initialColumnMappings]);

  const loadAsanaSections = async (showMessages = true) => {
    setLoading(prev => ({ ...prev, sections: true }));
    try {
      const response = await getAsanaSections();
      setAsanaSections(response.data || response);
      if (showMessages) {
        onSuccess?.('Asana sections loaded successfully!');
      }
    } catch (err) {
      if (showMessages) {
        onError?.('Failed to load Asana sections: ' + err.message);
      }
      console.error('Failed to load Asana sections:', err);
    } finally {
      setLoading(prev => ({ ...prev, sections: false }));
    }
  };

  const loadYouTrackStates = async (showMessages = true) => {
    setLoading(prev => ({ ...prev, states: true }));
    try {
      const response = await getYouTrackStates();
      const states = response.data || response;
      setYoutrackStates(Array.isArray(states) ? states : []);
      if (showMessages) {
        onSuccess?.('YouTrack states loaded successfully!');
      }
    } catch (err) {
      if (showMessages) {
        onError?.('Failed to load YouTrack states: ' + err.message);
      }
      console.error('Failed to load YouTrack states:', err);
      setYoutrackStates([]);
    } finally {
      setLoading(prev => ({ ...prev, states: false }));
    }
  };

  const loadYouTrackBoards = async () => {
    setLoading(prev => ({ ...prev, boards: true }));
    try {
      const response = await getYouTrackBoards();
      setYoutrackBoards(response.data || response);
      onSuccess?.('YouTrack boards loaded successfully!');
    } catch (err) {
      onError?.('Failed to load YouTrack boards: ' + err.message);
    } finally {
      setLoading(prev => ({ ...prev, boards: false }));
    }
  };

  const addMapping = () => {
    setColumnMappings([
      ...columnMappings,
      { asana_column: '', youtrack_status: '', display_only: false }
    ]);
  };

  const removeMapping = (index) => {
    const newMappings = columnMappings.filter((_, i) => i !== index);
    setColumnMappings(newMappings);
  };

  const updateMapping = (index, field, value) => {
    const newMappings = [...columnMappings];
    newMappings[index] = {
      ...newMappings[index],
      [field]: value
    };
    setColumnMappings(newMappings);
  };

  const validateMappings = () => {
    // Check for empty mappings
    for (let i = 0; i < columnMappings.length; i++) {
      const mapping = columnMappings[i];
      if (!mapping.asana_column) {
        return { valid: false, error: `Mapping ${i + 1}: Please select an Asana column` };
      }
      if (!mapping.display_only && !mapping.youtrack_status) {
        return { valid: false, error: `Mapping ${i + 1}: Please select a YouTrack status or mark as display-only` };
      }
    }

    // Check for duplicate Asana columns
    const asanaColumns = columnMappings.map(m => m.asana_column);
    const duplicates = asanaColumns.filter((col, index) => asanaColumns.indexOf(col) !== index);
    if (duplicates.length > 0) {
      return { valid: false, error: `Duplicate Asana column: ${duplicates[0]}` };
    }

    return { valid: true };
  };

  const handleSave = async () => {
    const validation = validateMappings();
    if (!validation.valid) {
      onError?.(validation.error);
      return;
    }

    setSaving(true);
    try {
      const updatedSettings = {
        ...settings,
        column_mappings: {
          asana_to_youtrack: columnMappings,
          youtrack_to_asana: [] // For future bidirectional support
        }
      };

      await updateUserSettings(updatedSettings);

      // Update the initial state to match current (disable save button)
      setInitialColumnMappings(JSON.parse(JSON.stringify(columnMappings)));
      setHasChanges(false);

      onSettingsUpdate?.(updatedSettings);
      onSuccess?.('Column mappings saved successfully!');
    } catch (err) {
      onError?.('Failed to save column mappings: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h3 className="settings-section-header">Column Mapping Configuration</h3>
        <p className="settings-section-description">
          Configure how Asana columns map to YouTrack states. Display-only columns will be shown on the dashboard but tickets won't be synced.
        </p>
      </div>

      {/* Show info if columns not loaded */}
      {!loading.sections && !loading.states && (asanaSections.length === 0 || youtrackStates.length === 0) && (
        <div className="info-box">
          <p className="text-sm flex items-center">
            <AlertCircle className="w-4 h-4 mr-2" />
            Columns will be loaded automatically. Please ensure your API credentials are configured in the API Configuration tab.
          </p>
        </div>
      )}

      {/* Mappings Configuration */}
      <div className="settings-form-group">
        <div className="flex items-center justify-between mb-4">
          <h4 className="text-md font-medium text-gray-900">Configure Column Mappings</h4>
          <button
            onClick={addMapping}
            disabled={asanaSections.length === 0 || youtrackStates.length === 0}
            className="settings-button-secondary"
          >
            <Plus className="w-4 h-4 mr-2" />
            Add Mapping
          </button>
        </div>

        {columnMappings.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500 text-sm">No column mappings configured. Click "Add Mapping" to create one.</p>
          </div>
        ) : (
          <div className="settings-mapping-container">
            {columnMappings.map((mapping, index) => (
              <div key={index}>
                <div className="settings-form-row" style={{ marginBottom: index < columnMappings.length - 1 ? '12px' : '0' }}>
                  {/* Asana Column */}
                  <select
                    value={mapping.asana_column}
                    onChange={(e) => updateMapping(index, 'asana_column', e.target.value)}
                    className="settings-select flex-1"
                    disabled={asanaSections.length === 0}
                  >
                    <option value="">Select Asana Column</option>
                    {asanaSections.map(section => (
                      <option key={section.gid || section.id} value={section.name}>
                        {section.name}
                      </option>
                    ))}
                  </select>

                  {/* Arrow Icon */}
                  <div className="flex items-center justify-center self-center">
                    <ArrowRight className="w-7 h-7 text-gray-400" strokeWidth={2.5} />
                  </div>

                  {/* YouTrack Status */}
                  <select
                    value={mapping.youtrack_status}
                    onChange={(e) => updateMapping(index, 'youtrack_status', e.target.value)}
                    className="settings-select flex-1"
                    disabled={youtrackStates.length === 0 || mapping.display_only}
                  >
                    <option value="">Select YouTrack Status</option>
                    {youtrackStates.map(state => (
                      <option key={state.id} value={state.name}>
                        {state.name}
                      </option>
                    ))}
                  </select>

                  {/* Display Only Toggle */}
                  <label className="flex items-center space-x-2 px-3 py-2 bg-white bg-opacity-30 rounded-lg cursor-pointer border border-white border-opacity-40">
                    <input
                      type="checkbox"
                      checked={mapping.display_only}
                      onChange={(e) => updateMapping(index, 'display_only', e.target.checked)}
                      className="mapping-checkbox"
                    />
                    <Eye className="w-4 h-4 text-gray-600" />
                    <span className="text-sm font-medium text-gray-700">Display Only</span>
                  </label>

                  {/* Remove Button */}
                  <button
                    onClick={() => removeMapping(index)}
                    className="delete-button-table"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Save Button */}
      <div className="flex items-center justify-between pt-4 border-t border-white border-opacity-30">
        <div className="info-box flex-1 mr-4">
          <p className="text-xs">
            <strong>Note:</strong> Unmapped columns will be hidden from the dashboard and excluded from sync operations.
            Multiple Asana columns can map to the same YouTrack status.
          </p>
        </div>

        <button
          onClick={handleSave}
          disabled={saving || columnMappings.length === 0 || !hasChanges}
          className="settings-button-success"
        >
          {saving ? (
            <>
              <RefreshCw className="settings-spinner" />
              Saving...
            </>
          ) : (
            <>
              <CheckCircle className="w-4 h-4 mr-2" />
              {hasChanges ? 'Save Column Mappings' : 'No Changes to Save'}
            </>
          )}
        </button>
      </div>

      {/* Help Section */}
      <div className="info-box">
        <h4 className="text-sm font-medium mb-2 flex items-center">
          <AlertCircle className="w-4 h-4 mr-1" />
          How Column Mapping Works
        </h4>
        <ul className="text-xs space-y-1">
          <li>• <strong>Mapped Columns:</strong> Tickets will be synced based on your configuration</li>
          <li>• <strong>Display Only:</strong> Column is visible on dashboard but no sync operations</li>
          <li>• <strong>Unmapped Columns:</strong> Hidden from dashboard and excluded from all operations</li>
          <li>• <strong>Many-to-One:</strong> Multiple Asana columns can map to the same YouTrack status</li>
          <li>• <strong>Bidirectional:</strong> Future support for YouTrack → Asana mapping</li>
        </ul>
      </div>
    </div>
  );
};

export default ColumnMappingSettings;
