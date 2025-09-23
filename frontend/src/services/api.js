// Enhanced API service with authentication support
// Preserves all existing endpoints while adding new auth features

const API_BASE =
  process.env.NODE_ENV === 'production'
    ? process.env.REACT_APP_API_URL || 'https://boardsyncapi.onrender.com'
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

// Auto-refresh token before expiration
let refreshTimeout;

const scheduleTokenRefresh = () => {
  // Clear existing timeout
  if (refreshTimeout) {
    clearTimeout(refreshTimeout);
  }

  if (!authToken) return;

  try {
    // Decode JWT to get expiration (simple decode, not verified)
    const tokenParts = authToken.split('.');
    if (tokenParts.length === 3) {
      const payload = JSON.parse(atob(tokenParts[1]));
      const exp = payload.exp * 1000; // Convert to milliseconds
      const now = Date.now();
      const refreshTime = exp - now - 5 * 60 * 1000; // Refresh 5 minutes before expiry

      if (refreshTime > 0) {
        refreshTimeout = setTimeout(async () => {
          try {
            await refreshToken();
            scheduleTokenRefresh(); // Schedule next refresh
          } catch (error) {
            console.warn('Token refresh failed:', error);
            setAuthToken(null);
          }
        }, refreshTime);
      }
    }
  } catch (error) {
    console.warn('Failed to schedule token refresh:', error);
  }
};

const setAuthToken = (token) => {
  authToken = token;
  if (token) {
    localStorage.setItem('auth_token', token);
  } else {
    localStorage.removeItem('auth_token');
  }
  scheduleTokenRefresh(); // Schedule refresh when token is set
};

const handleAuthError = (response) => {
  if (response.status === 401) {
    setAuthToken(null);
    // Optionally redirect to login
    window.location.href = '/login';
  }
  return response;
};

// ============================================================================
// EXISTING API ENDPOINTS (Preserved exactly as they were)
// ============================================================================

export const analyzeTickets = async (columnFilter = '') => {
  let url = `${API_BASE}/analyze`;
  if (columnFilter) {
    url += `?column=${encodeURIComponent(columnFilter)}`;
  }
  
  console.log('Analyzing tickets with column filter:', columnFilter);
  console.log('API URL:', url);
  
  const response = await fetch(url, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Analysis failed: ${response.status}`);
  }
  
  const result = await response.json();
  console.log('Analysis result:', result);
  return result;
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

export const createMissingTickets = async () => {
  const response = await fetch(`${API_BASE}/create`, {
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

export const getTicketsByType = async (type, column = '') => {
  const params = new URLSearchParams({ type });
  if (column) {
    params.append('column', column);
  }
  
  console.log('Getting tickets by type:', type, 'for column:', column);
  
  const response = await fetch(`${API_BASE}/tickets?${params}`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get tickets failed: ${response.status}`);
  }
  
  const result = await response.json();
  console.log('Get tickets result:', result);
  return result;
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

export const getHealth = async () => {
  const response = await fetch(`${API_BASE}/health`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
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

// ============================================================================
// NEW AUTHENTICATION ENDPOINTS
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
  
  // Store token if registration includes automatic login
  if (result.token) {
    setAuthToken(result.token);
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
  
  // Store the authentication token
  if (result.token) {
    setAuthToken(result.token);
  }
  
  return result;
};

export const logout = async () => {
  try {
    // Call logout endpoint if token exists
    if (authToken) {
      await fetch(`${API_BASE}/api/auth/logout`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
    }
  } catch (error) {
    console.warn('Logout endpoint failed:', error);
  } finally {
    // Always clear local token
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
  setAuthToken(result.token);
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
    const error = await response.json().catch(() => ({ message: 'Password change failed' }));
    throw new Error(error.message || 'Password change failed');
  }

  return response.json();
};

// ============================================================================
// NEW SETTINGS ENDPOINTS
// ============================================================================

export const getUserSettings = async () => {
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
// NEW SYNC HISTORY & ROLLBACK ENDPOINTS
// ============================================================================

export const getSyncHistory = async (limit = 50) => {
  const params = new URLSearchParams({ limit: limit.toString() });
  const response = await fetch(`${API_BASE}/api/sync/history?${params}`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get sync history');
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

export const rollbackSync = async (operationId) => {
  const response = await fetch(`${API_BASE}/api/sync/rollback/${operationId}`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Rollback failed');
  }

  return response.json();
};

export const startSyncOperation = async (syncRequest) => {
  const response = await fetch(`${API_BASE}/api/sync/start`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(syncRequest),
  });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to start sync operation');
  }

  return response.json();
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

// Initialize token refresh on load
if (authToken) {
  scheduleTokenRefresh();
}