// Enhanced API service with debug logging - Replace your frontend/src/services/api.js

const API_BASE =
  process.env.NODE_ENV === 'production'
    ? process.env.REACT_APP_API_URL || 'https://boardsyncapi.onrender.com'
    : 'http://localhost:8080';

// Token management with enhanced debugging
let authToken = localStorage.getItem('auth_token');

// Debug logging function
const debugLog = (message, data = null) => {
  if (process.env.NODE_ENV !== 'production') {
    console.log(`🔍 API Debug: ${message}`, data || '');
  }
};

debugLog('API Service initialized', { API_BASE, hasToken: !!authToken });

const getAuthHeaders = () => {
  const headers = { 'Content-Type': 'application/json' };
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
    debugLog('Adding auth header', { tokenLength: authToken.length });
  } else {
    debugLog('No auth token available');
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

  if (!authToken) {
    debugLog('No token to schedule refresh for');
    return;
  }

  try {
    // Decode JWT to get expiration (simple decode, not verified)
    const tokenParts = authToken.split('.');
    if (tokenParts.length === 3) {
      const payload = JSON.parse(atob(tokenParts[1]));
      const exp = payload.exp * 1000; // Convert to milliseconds
      const now = Date.now();
      const refreshTime = exp - now - 5 * 60 * 1000; // Refresh 5 minutes before expiry

      debugLog('Token refresh scheduled', { 
        expiresAt: new Date(exp), 
        refreshIn: Math.round(refreshTime / 1000 / 60) + ' minutes' 
      });

      if (refreshTime > 0) {
        refreshTimeout = setTimeout(async () => {
          try {
            debugLog('Auto-refreshing token');
            await refreshToken();
            scheduleTokenRefresh(); // Schedule next refresh
          } catch (error) {
            debugLog('Token refresh failed', error.message);
            setAuthToken(null);
          }
        }, refreshTime);
      }
    }
  } catch (error) {
    debugLog('Failed to schedule token refresh', error.message);
  }
};

const setAuthToken = (token) => {
  authToken = token;
  if (token) {
    localStorage.setItem('auth_token', token);
    debugLog('Token stored', { tokenLength: token.length });
  } else {
    localStorage.removeItem('auth_token');
    debugLog('Token removed');
  }
  scheduleTokenRefresh(); // Schedule refresh when token is set
};

const handleAuthError = (response) => {
  if (response.status === 401) {
    debugLog('Auth error detected, clearing token');
    setAuthToken(null);
    // Optionally redirect to login
    window.location.href = '/';
  }
  return response;
};

// ============================================================================
// AUTHENTICATION ENDPOINTS
// ============================================================================

export const register = async (userData) => {
  debugLog('Registering user', { username: userData.username });
  
  const response = await fetch(`${API_BASE}/api/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(userData),
  });

  debugLog('Register response', { status: response.status, ok: response.ok });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Registration failed' }));
    debugLog('Register error', error);
    throw new Error(error.message || 'Registration failed');
  }

  const result = await response.json();
  debugLog('Register success', { hasToken: !!result.token });
  
  // Store token if registration includes automatic login
  if (result.token) {
    setAuthToken(result.token);
  }
  
  return result;
};

export const login = async (credentials) => {
  debugLog('Logging in user', { username: credentials.username });
  
  const response = await fetch(`${API_BASE}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credentials),
  });

  debugLog('Login response', { status: response.status, ok: response.ok });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Login failed' }));
    debugLog('Login error', error);
    throw new Error(error.message || 'Invalid credentials');
  }

  const result = await response.json();
  debugLog('Login success', { hasToken: !!result.data?.token, hasUser: !!result.data?.user });
  
  // Handle different response structures
  const token = result.data?.token || result.token;
  const user = result.data?.user || result.user;
  
  if (token) {
    setAuthToken(token);
    debugLog('Token set after login', { tokenLength: token.length });
  } else {
    debugLog('No token in login response', result);
  }
  
  return { token, user, ...result };
};

export const logout = async () => {
  debugLog('Logging out user');
  
  try {
    // Call logout endpoint if token exists
    if (authToken) {
      await fetch(`${API_BASE}/api/auth/logout`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
    }
  } catch (error) {
    debugLog('Logout endpoint failed', error.message);
  } finally {
    // Always clear local token
    setAuthToken(null);
    debugLog('Logout completed');
  }
};

export const refreshToken = async () => {
  debugLog('Refreshing token');
  
  if (!authToken) {
    throw new Error('No token to refresh');
  }

  const response = await fetch(`${API_BASE}/api/auth/refresh`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    debugLog('Token refresh failed', response.status);
    setAuthToken(null);
    throw new Error('Token refresh failed');
  }

  const result = await response.json();
  const newToken = result.data?.token || result.token;
  
  if (newToken) {
    setAuthToken(newToken);
    debugLog('Token refreshed successfully');
  }
  
  return result;
};

export const getCurrentUser = async () => {
  debugLog('Getting current user');
  
  if (!authToken) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE}/api/auth/me`, {
    headers: getAuthHeaders(),
  });

  debugLog('Get current user response', { status: response.status, ok: response.ok });

  if (!response.ok) {
    handleAuthError(response);
    throw new Error('Failed to get user info');
  }

  const result = await response.json();
  debugLog('Current user retrieved', { hasUser: !!(result.data || result.user) });
  return result;
};

// ============================================================================
// SETTINGS ENDPOINTS (Enhanced with debugging)
// ============================================================================

export const getUserSettings = async () => {
  debugLog('Getting user settings', { hasToken: !!authToken });
  
  if (!authToken) {
    debugLog('No token for settings request');
    throw new Error('Not authenticated');
  }
  
  const headers = getAuthHeaders();
  debugLog('Settings request headers', headers);

  const response = await fetch(`${API_BASE}/api/settings`, {
    headers: headers,
  });

  debugLog('Get settings response', { status: response.status, ok: response.ok });

  if (!response.ok) {
    if (response.status === 401) {
      debugLog('Settings request got 401 - token might be invalid');
    }
    handleAuthError(response);
    
    // Try to get response body for more details
    const errorText = await response.text().catch(() => 'No error details');
    debugLog('Settings error details', errorText);
    
    throw new Error('Failed to get user settings');
  }

  const result = await response.json();
  debugLog('Settings retrieved successfully');
  return result;
};

export const updateUserSettings = async (settings) => {
  debugLog('Updating user settings');
  
  const response = await fetch(`${API_BASE}/api/settings`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(settings),
  });

  if (!response.ok) {
    handleAuthError(response);
    const error = await response.json().catch(() => ({ message: 'Settings update failed' }));
    debugLog('Settings update failed', error);
    throw new Error(error.message || 'Settings update failed');
  }

  const result = await response.json();
  debugLog('Settings updated successfully');
  return result;
};

export const getAsanaProjects = async () => {
  debugLog('Getting Asana projects');
  
  const response = await fetch(`${API_BASE}/api/settings/asana/projects`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    debugLog('Get Asana projects failed', response.status);
    throw new Error('Failed to get Asana projects');
  }

  const result = await response.json();
  debugLog('Asana projects retrieved');
  return result;
};

export const getYouTrackProjects = async () => {
  debugLog('Getting YouTrack projects');
  
  const response = await fetch(`${API_BASE}/api/settings/youtrack/projects`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    debugLog('Get YouTrack projects failed', response.status);
    throw new Error('Failed to get YouTrack projects');
  }

  const result = await response.json();
  debugLog('YouTrack projects retrieved');
  return result;
};

export const testConnections = async () => {
  debugLog('Testing connections');
  
  const response = await fetch(`${API_BASE}/api/settings/test-connections`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    handleAuthError(response);
    debugLog('Test connections failed', response.status);
    throw new Error('Connection test failed');
  }

  const result = await response.json();
  debugLog('Connection test completed');
  return result;
};

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

export const isAuthenticated = () => {
  const authenticated = !!authToken;
  debugLog('Checking authentication', { authenticated, tokenLength: authToken?.length });
  return authenticated;
};

export const getToken = () => {
  return authToken;
};

export const clearAuth = () => {
  debugLog('Clearing authentication');
  setAuthToken(null);
};

// Initialize token refresh on load
if (authToken) {
  debugLog('Initializing token refresh on load');
  scheduleTokenRefresh();
}

// ============================================================================
// EXISTING API ENDPOINTS (preserved)
// ============================================================================

export const analyzeTickets = async (columnFilter = '') => {
  let url = `${API_BASE}/analyze`;
  if (columnFilter) {
    url += `?column=${encodeURIComponent(columnFilter)}`;
  }
  
  debugLog('Analyzing tickets', { columnFilter, url });
  
  const response = await fetch(url, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Analysis failed: ${response.status}`);
  }
  
  const result = await response.json();
  debugLog('Analysis completed');
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
  
  debugLog('Getting tickets by type', { type, column });
  
  const response = await fetch(`${API_BASE}/tickets?${params}`, {
    headers: getAuthHeaders()
  });
  
  if (!response.ok) {
    handleAuthError(response);
    throw new Error(`Get tickets failed: ${response.status}`);
  }
  
  const result = await response.json();
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
// WEBSOCKET CONNECTION
// ============================================================================

export const createWebSocketConnection = (userId) => {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsBase = API_BASE.replace(/^https?:/, wsProtocol);
  
  return new WebSocket(`${wsBase}/ws?user_id=${userId}`);
};