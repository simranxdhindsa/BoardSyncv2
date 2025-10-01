// Complete API service - All endpoints wired for backend integration

const API_BASE =
  process.env.NODE_ENV === 'production'
    ? process.env.REACT_APP_API_URL || 'https://boardsyncv2.onrender.com'
    : 'http://localhost:8080';

// Token management
let authToken = localStorage.getItem('auth_token');

const getAuthHeaders = () => {
  const headers = { 'Content-Type': 'application/json' };
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }
  return headers;
};

const setAuthToken = (token) => {
  authToken = token;
  if (token) {
    localStorage.setItem('auth_token', token);
  } else {
    localStorage.removeItem('auth_token');
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
// AUTHENTICATION ENDPOINTS
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

export const refreshToken = async () => {
  if (!authToken) {
    throw new Error('No token to refresh');
  }

  const response = await fetch(`${API_BASE}/api/auth/refresh`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    setAuthToken(null);
    throw new Error('Token refresh failed');
  }

  const result = await response.json();
  const newToken = result.data?.token || result.token;
  if (newToken) {
    setAuthToken(newToken);
  }
  return result;
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

export const changePassword = async (passwordData) => {
  const response = await fetch(`${API_BASE}/api/auth/change-password`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(passwordData),
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Password change failed' }));
    throw new Error(error.message || 'Password change failed');
  }

  return response.json();
};

export const getAccountSummary = async () => {
  if (!authToken) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE}/api/auth/account/summary`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get account summary');
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
  // Clear auth after successful deletion
  setAuthToken(null);
  return result;
};

// ============================================================================
// SETTINGS ENDPOINTS
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
// NEW SYNC API ENDPOINTS
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

export const getSyncHistory = async (limit = 50) => {
  const response = await fetch(`${API_BASE}/api/sync/history?limit=${limit}`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get sync history');
  }

  return response.json();
};

export const rollbackSync = async (operationId) => {
  const response = await fetch(`${API_BASE}/api/sync/rollback/${operationId}`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to rollback sync');
  }

  return response.json();
};

// ============================================================================
// LEGACY API ENDPOINTS (All wired)
// ============================================================================

// Health and Status
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

// Analysis
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

// Ticket Creation
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

export const createByColumn = async (column) => {
  const response = await fetch(`${API_BASE}/create-by-column?column=${encodeURIComponent(column)}`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Create by column failed: ${response.status}`);
  }
  return response.json();
};

// Synchronization
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

export const getMismatchedTickets = async () => {
  const response = await fetch(`${API_BASE}/sync`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get mismatched tickets failed: ${response.status}`);
  }
  return response.json();
};

export const syncSingleTicket = async (ticketId) => {
  return syncTickets([{ ticket_id: ticketId, action: 'sync' }]);
};

export const syncByColumn = async (column) => {
  const response = await fetch(`${API_BASE}/sync-by-column?column=${encodeURIComponent(column)}`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Sync by column failed: ${response.status}`);
  }
  return response.json();
};

// Ticket Retrieval
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

export const getSyncableTickets = async () => {
  const response = await fetch(`${API_BASE}/syncable-tickets`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get syncable tickets failed: ${response.status}`);
  }
  return response.json();
};

// Statistics
export const getSyncStats = async () => {
  const response = await fetch(`${API_BASE}/sync-stats`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get sync stats failed: ${response.status}`);
  }
  return response.json();
};

// Deletion
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

export const getDeletionPreview = async (ticketIds, source) => {
  const params = new URLSearchParams();
  ticketIds.forEach(id => params.append('ticket_ids', id));
  params.append('source', source);
  
  const response = await fetch(`${API_BASE}/deletion-preview?${params}`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get deletion preview failed: ${response.status}`);
  }
  return response.json();
};

export const getSyncPreview = async (ticketIds) => {
  const params = new URLSearchParams();
  ticketIds.forEach(id => params.append('ticket_ids', id));
  
  const response = await fetch(`${API_BASE}/sync-preview?${params}`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get sync preview failed: ${response.status}`);
  }
  return response.json();
};

// Ignore Management
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

export const getIgnoredTickets = async () => {
  const response = await fetch(`${API_BASE}/ignore`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get ignored tickets failed: ${response.status}`);
  }
  return response.json();
};

// Auto-sync Management
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

// Auto-create Management
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

// ============================================================================
// WEBSOCKET CONNECTION
// ============================================================================

export const createWebSocketConnection = (userId) => {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsBase = API_BASE.replace(/^https?:/, wsProtocol);
  
  return new WebSocket(`${wsBase}/ws?user_id=${userId}`);
};

// ============================================================================
// API DOCUMENTATION
// ============================================================================

export const getApiDocs = async () => {
  const response = await fetch(`${API_BASE}/api/docs`);
  if (!response.ok) {
    throw new Error(`API docs failed: ${response.status}`);
  }
  return response.json();
};

// ============================================================================
// EXPORT GROUPED FUNCTIONS (for easier imports)
// ============================================================================

export const auth = {
  register,
  login,
  logout,
  refreshToken,
  getCurrentUser,
  changePassword,
  getAccountSummary,
  deleteAccount,
  isAuthenticated,
  getToken,
  clearAuth
};

export const settings = {
  getUserSettings,
  updateUserSettings,
  getAsanaProjects,
  getYouTrackProjects,
  testConnections
};

export const sync = {
  startSync,
  getSyncStatus,
  getSyncHistory,
  rollbackSync,
  syncTickets,
  syncSingleTicket,
  syncByColumn,
  getMismatchedTickets,
  getSyncableTickets,
  getSyncStats,
  getSyncPreview
};

export const tickets = {
  analyzeTickets,
  createMissingTickets,
  createSingleTicket,
  createByColumn,
  getTicketsByType,
  deleteTickets,
  getDeletionPreview
};

export const ignore = {
  ignoreTicket,
  unignoreTicket,
  getIgnoredTickets
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
  getApiDocs,
  createWebSocketConnection
};