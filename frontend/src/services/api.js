// src/services/api.js
// MINIMAL CHANGES - Only fixing token retrieval issue

const API_BASE =
  process.env.NODE_ENV === 'production'
    ? process.env.REACT_APP_API_URL || 'https://boardsyncv2.onrender.com'
    : 'http://localhost:8080';

// FIXED: Token management - check both possible storage keys
let authToken = localStorage.getItem('auth_token') || localStorage.getItem('token');

const getAuthHeaders = () => {
  // FIXED: Re-check localStorage on every call in case token was updated
  authToken = localStorage.getItem('auth_token') || localStorage.getItem('token');
  
  const headers = { 'Content-Type': 'application/json' };
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }
  return headers;
};

const setAuthToken = (token) => {
  authToken = token;
  if (token) {
    // FIXED: Store in both locations for compatibility
    localStorage.setItem('auth_token', token);
    localStorage.setItem('token', token);
  } else {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('token');
  }
};

const handleAuthError = (response) => {
  if (response.status === 401) {
    setAuthToken(null);
    window.location.href = '/';
  }
  return response;
};

// ============================================================================
// AUTHENTICATION ENDPOINTS (unchanged)
// ============================================================================

export const register = async (userData) => {
  const response = await fetch(`${API_BASE}/api/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(userData),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Registration failed' }));
    throw new Error(error.message || 'Registration failed');
  }

  const result = await response.json();
  if (result.data?.token || result.token) {
    setAuthToken(result.data?.token || result.token);
  }
  return result;
};

export const login = async (credentials) => {
  const response = await fetch(`${API_BASE}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credentials),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Login failed' }));
    throw new Error(error.message || 'Invalid credentials');
  }

  const result = await response.json();
  const token = result.data?.token || result.token;
  if (token) {
    setAuthToken(token);
  }
  return result;
};

export const logout = async () => {
  try {
    if (authToken) {
      await fetch(`${API_BASE}/api/auth/logout`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
    }
  } catch (error) {
    // Continue with logout even if endpoint fails
  } finally {
    setAuthToken(null);
  }
};

export const getCurrentUser = async () => {
  if (!authToken) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE}/api/auth/me`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get user info');
  }

  return response.json();
};

export const changePassword = async (oldPassword, newPassword) => {
  if (!authToken) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE}/api/auth/change-password`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({
      old_password: oldPassword,
      new_password: newPassword
    }),
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Password change failed' }));
    throw new Error(error.message || 'Password change failed');
  }

  return response.json();
};

export const deleteAccount = async (deleteData) => {
  if (!authToken) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE}/api/auth/account/delete`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(deleteData),
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Account deletion failed' }));
    throw new Error(error.message || 'Account deletion failed');
  }

  const result = await response.json();
  setAuthToken(null);
  return result;
};

// ============================================================================
// SETTINGS ENDPOINTS (unchanged)
// ============================================================================

export const getUserSettings = async () => {
  if (!authToken) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE}/api/settings`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get user settings');
  }

  return response.json();
};

export const updateUserSettings = async (settings) => {
  const response = await fetch(`${API_BASE}/api/settings`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(settings),
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Settings update failed' }));
    throw new Error(error.message || 'Settings update failed');
  }

  return response.json();
};

export const getAsanaProjects = async () => {
  const response = await fetch(`${API_BASE}/api/settings/asana/projects`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get Asana projects');
  }

  return response.json();
};

export const getYouTrackProjects = async () => {
  const response = await fetch(`${API_BASE}/api/settings/youtrack/projects`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get YouTrack projects');
  }

  return response.json();
};

export const testConnections = async () => {
  const response = await fetch(`${API_BASE}/api/settings/test-connections`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Connection test failed');
  }

  return response.json();
};

// ============================================================================
// NEW: MAPPING ENDPOINTS
// ============================================================================

export const createMapping = async (asanaUrl, youtrackUrl) => {
  const response = await fetch(`${API_BASE}/api/mappings`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({
      asana_url: asanaUrl,
      youtrack_url: youtrackUrl
    }),
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Failed to create mapping' }));
    throw new Error(error.message || 'Failed to create mapping');
  }

  return response.json();
};

export const getAllMappings = async () => {
  const response = await fetch(`${API_BASE}/api/mappings`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get mappings');
  }

  return response.json();
};

export const deleteMapping = async (id) => {
  const response = await fetch(`${API_BASE}/api/mappings/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Failed to delete mapping' }));
    throw new Error(error.message || 'Failed to delete mapping');
  }

  return response.json();
};

export const findMappingByAsanaId = async (taskId) => {
  const response = await fetch(`${API_BASE}/api/mappings/asana/${taskId}`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Mapping not found');
  }

  return response.json();
};

export const findMappingByYouTrackId = async (issueId) => {
  const response = await fetch(`${API_BASE}/api/mappings/youtrack/${issueId}`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Mapping not found');
  }

  return response.json();
};

// ============================================================================
// SYNC API ENDPOINTS (unchanged)
// ============================================================================

export const startSync = async (syncData) => {
  const response = await fetch(`${API_BASE}/api/sync/start`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(syncData),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to start sync');
  }

  return response.json();
};

export const getSyncStatus = async (operationId) => {
  const response = await fetch(`${API_BASE}/api/sync/status/${operationId}`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get sync status');
  }

  return response.json();
};

// ============================================================================
// LEGACY API ENDPOINTS (unchanged - all existing functions)
// ============================================================================

export const getHealth = async () => {
  const response = await fetch(`${API_BASE}/health`);
  if (!response.ok) {
    throw new Error(`Health check failed: ${response.status}`);
  }
  return response.json();
};

export const getStatus = async () => {
  const response = await fetch(`${API_BASE}/status`, {
    headers: getAuthHeaders()
  });
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Status check failed: ${response.status}`);
  }
  return response.json();
};

export const analyzeTickets = async (columnFilter = '') => {
  let url = `${API_BASE}/analyze`;
  if (columnFilter) {
    url += `?column=${encodeURIComponent(columnFilter)}`;
  }
  
  const response = await fetch(url, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Analysis failed: ${response.status}`);
  }
  
  return response.json();
};

export const createMissingTickets = async (column = '') => {
  let url = `${API_BASE}/create`;
  if (column && column !== 'all_syncable') {
    url += `?column=${encodeURIComponent(column)}`;
  }
  
  const response = await fetch(url, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Create failed: ${response.status}`);
  }
  return response.json();
};

export const createSingleTicket = async (taskId) => {
  const response = await fetch(`${API_BASE}/create-single`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ task_id: taskId }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Single create failed: ${response.status}`);
  }
  return response.json();
};

export const syncTickets = async (tickets) => {
  const response = await fetch(`${API_BASE}/sync`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(tickets),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Sync failed: ${response.status}`);
  }
  return response.json();
};

export const syncSingleTicket = async (ticketId) => {
  return syncTickets([{ ticket_id: ticketId, action: 'sync' }]);
};

export const getTicketsByType = async (type, column = '') => {
  const params = new URLSearchParams({ type });
  if (column) {
    params.append('column', column);
  }
  
  const response = await fetch(`${API_BASE}/tickets?${params}`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get tickets failed: ${response.status}`);
  }
  
  return response.json();
};

export const deleteTickets = async (ticketIds, source) => {
  if (!Array.isArray(ticketIds) || ticketIds.length === 0) {
    throw new Error('ticketIds must be a non-empty array');
  }
  
  if (!['asana', 'youtrack', 'both'].includes(source)) {
    throw new Error('source must be one of: asana, youtrack, both');
  }

  const response = await fetch(`${API_BASE}/delete-tickets`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({
      ticket_ids: ticketIds,
      source: source
    }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    const errorData = await response.json().catch(() => null);
    throw new Error(
      errorData?.error || `Delete failed with status: ${response.status}`
    );
  }
  
  return response.json();
};

export const ignoreTicket = async (ticketId, type = 'forever') => {
  const response = await fetch(`${API_BASE}/ignore`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ 
      ticket_id: ticketId, 
      action: 'add', 
      type 
    }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Ignore ticket failed: ${response.status}`);
  }
  return response.json();
};

export const unignoreTicket = async (ticketId, type = 'forever') => {
  const response = await fetch(`${API_BASE}/ignore`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ 
      ticket_id: ticketId, 
      action: 'remove', 
      type 
    }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Unignore ticket failed: ${response.status}`);
  }
  return response.json();
};

export const getAutoSyncStatus = async () => {
  const response = await fetch(`${API_BASE}/auto-sync`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Auto-sync status failed: ${response.status}`);
  }
  return response.json();
};

export const startAutoSync = async (interval = 15) => {
  const response = await fetch(`${API_BASE}/auto-sync`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ action: 'start', interval }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Start auto-sync failed: ${response.status}`);
  }
  return response.json();
};

export const stopAutoSync = async () => {
  const response = await fetch(`${API_BASE}/auto-sync`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ action: 'stop' }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Stop auto-sync failed: ${response.status}`);
  }
  return response.json();
};

export const getAutoCreateStatus = async () => {
  const response = await fetch(`${API_BASE}/auto-create`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Auto-create status failed: ${response.status}`);
  }
  return response.json();
};

export const startAutoCreate = async (interval = 15) => {
  const response = await fetch(`${API_BASE}/auto-create`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ action: 'start', interval }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Start auto-create failed: ${response.status}`);
  }
  return response.json();
};

export const stopAutoCreate = async () => {
  const response = await fetch(`${API_BASE}/auto-create`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ action: 'stop' }),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Stop auto-create failed: ${response.status}`);
  }
  return response.json();
};

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

export const isAuthenticated = () => {
  return !!authToken;
};

export const getToken = () => {
  return authToken;
};

export const clearAuth = () => {
  setAuthToken(null);
};

export const createWebSocketConnection = (userId) => {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsBase = API_BASE.replace(/^https?:/, wsProtocol);
  
  return new WebSocket(`${wsBase}/ws?user_id=${userId}`);
};

// ============================================================================
// EXPORT GROUPED FUNCTIONS (for easier imports)
// ============================================================================

export const auth = {
  register,
  login,
  logout,
  getCurrentUser,
  changePassword,
  deleteAccount,
  isAuthenticated,
  getToken,
  clearAuth
};

export const getAsanaSections = async () => {
  const response = await fetch(`${API_BASE}/api/settings/columns/asana`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get Asana sections');
  }

  return response.json();
};

export const getYouTrackStates = async () => {
  const response = await fetch(`${API_BASE}/api/settings/columns/youtrack`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get YouTrack states');
  }

  return response.json();
};

export const getYouTrackBoards = async () => {
  const response = await fetch(`${API_BASE}/api/settings/youtrack/boards`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get YouTrack boards');
  }

  return response.json();
};

export const settings = {
  getUserSettings,
  updateUserSettings,
  getAsanaProjects,
  getYouTrackProjects,
  getAsanaSections,
  getYouTrackStates,
  getYouTrackBoards,
  testConnections
};

export const mapping = {
  createMapping,
  getAllMappings,
  deleteMapping,
  findMappingByAsanaId,
  findMappingByYouTrackId
};

export const sync = {
  startSync,
  getSyncStatus,
  syncTickets,
  syncSingleTicket
};

export const tickets = {
  analyzeTickets,
  createMissingTickets,
  createSingleTicket,
  getTicketsByType,
  deleteTickets
};

export const ignore = {
  ignoreTicket,
  unignoreTicket
};

export const autoSync = {
  getAutoSyncStatus,
  startAutoSync,
  stopAutoSync
};

export const autoCreate = {
  getAutoCreateStatus,
  startAutoCreate,
  stopAutoCreate
};

export const system = {
  getHealth,
  getStatus,
  createWebSocketConnection
};

// frontend/src/services/api.js - ADD these new functions to your existing api.js

// NEW: Get filter options for a column
export const getFilterOptions = async (column = '') => {
  let url = `${API_BASE}/filter-options`;
  if (column) {
    url += `?column=${encodeURIComponent(column)}`;
  }
  
  const response = await fetch(url, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get filter options: ${response.status}`);
  }
  
  return response.json();
};

// NEW: Get enhanced analysis with filters and sorting
export const getEnhancedAnalysis = async (requestBody) => {
  const response = await fetch(`${API_BASE}/analyze/enhanced`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(requestBody)
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Enhanced analysis failed: ${response.status}`);
  }
  
  return response.json();
};

// NEW: Get changed mappings
export const getChangedMappings = async () => {
  const response = await fetch(`${API_BASE}/changed-mappings`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get changed mappings: ${response.status}`);
  }
  
  return response.json();
};

// NEW: Sync enhanced tickets (with title/description updates)
export const syncEnhancedTickets = async (column = '', body = {}) => {
  let url = `${API_BASE}/sync/enhanced`;
  if (column) {
    url += `?column=${encodeURIComponent(column)}`;
  }
  
  const response = await fetch(url, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(body)
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Enhanced sync failed: ${response.status}`);
  }
  
  return response.json();
};

// NEW: Get detailed auto-sync status
export const getAutoSyncDetailed = async () => {
  const response = await fetch(`${API_BASE}/auto-sync/detailed`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get auto-sync details: ${response.status}`);
  }

  return response.json();
};

// ============================================================================
// ROLLBACK & AUDIT LOG ENDPOINTS
// ============================================================================

// Get sync history (last 15 operations)
export const getSyncHistory = async (limit = 15) => {
  const response = await fetch(`${API_BASE}/api/sync/history?limit=${limit}`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get sync history: ${response.status}`);
  }

  return response.json();
};

// Rollback a sync operation
export const rollbackSync = async (operationId) => {
  const response = await fetch(`${API_BASE}/api/sync/rollback/${operationId}`, {
    method: 'POST',
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ error: 'Rollback failed' }));
    throw new Error(error.error || `Rollback failed: ${response.status}`);
  }

  return response.json();
};

// Get snapshot summary for an operation
export const getSnapshotSummary = async (operationId) => {
  const response = await fetch(`${API_BASE}/api/sync/snapshot/${operationId}`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get snapshot summary: ${response.status}`);
  }

  return response.json();
};

// Get operation audit logs
export const getOperationAuditLogs = async (operationId) => {
  const response = await fetch(`${API_BASE}/api/sync/operation/${operationId}/logs`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get operation logs: ${response.status}`);
  }

  return response.json();
};

// Get audit logs with filtering
export const getAuditLogs = async (filters = {}) => {
  const params = new URLSearchParams();
  if (filters.userEmail) params.append('user_email', filters.userEmail);
  if (filters.ticketId) params.append('ticket_id', filters.ticketId);
  if (filters.platform) params.append('platform', filters.platform);
  if (filters.actionType) params.append('action_type', filters.actionType);
  if (filters.startDate) params.append('start_date', filters.startDate);
  if (filters.endDate) params.append('end_date', filters.endDate);
  if (filters.limit) params.append('limit', filters.limit);

  const response = await fetch(`${API_BASE}/api/audit/logs?${params.toString()}`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get audit logs: ${response.status}`);
  }

  return response.json();
};

// Export audit logs as CSV
export const exportAuditLogsCSV = async (filters = {}) => {
  const params = new URLSearchParams();
  if (filters.platform) params.append('platform', filters.platform);
  if (filters.actionType) params.append('action_type', filters.actionType);
  if (filters.startDate) params.append('start_date', filters.startDate);
  if (filters.endDate) params.append('end_date', filters.endDate);

  const response = await fetch(`${API_BASE}/api/audit/logs/export?${params.toString()}`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to export audit logs: ${response.status}`);
  }

  // Return blob for CSV download
  const blob = await response.blob();
  return blob;
};

// Get ticket history
export const getTicketHistory = async (ticketId) => {
  const response = await fetch(`${API_BASE}/api/audit/ticket/${ticketId}/history`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get ticket history: ${response.status}`);
  }

  return response.json();
};

// ============================================================================
// REVERSE SYNC ENDPOINTS (YouTrack → Asana)
// ============================================================================

// Get YouTrack users for creator dropdown
export const getYouTrackUsers = async () => {
  const response = await fetch(`${API_BASE}/reverse-sync/users`, {
    headers: getAuthHeaders()
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Failed to get YouTrack users: ${response.status}`);
  }

  const result = await response.json();
  return result.data || result;
};

// Perform reverse analysis (YouTrack → Asana)
export const reverseAnalyzeTickets = async (creatorFilter) => {
  const response = await fetch(`${API_BASE}/reverse-sync/analyze`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({
      creator_filter: creatorFilter
    })
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Reverse analysis failed: ${response.status}`);
  }

  const result = await response.json();
  return result.data || result;
};

// Create tickets from YouTrack to Asana
export const reverseCreateTickets = async (selectedIssueIDs = []) => {
  const response = await fetch(`${API_BASE}/reverse-sync/create`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({
      selected_issue_ids: selectedIssueIDs
    })
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Create failed' }));
    throw new Error(error.message || 'Failed to create tickets');
  }

  return response.json();
};