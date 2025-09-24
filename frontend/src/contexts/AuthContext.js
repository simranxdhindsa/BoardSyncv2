// Fixed AuthContext - Replace frontend/src/contexts/AuthContext.js

import React, { createContext, useContext, useReducer, useEffect } from 'react';
import { getCurrentUser, isAuthenticated, clearAuth, createWebSocketConnection } from '../services/api';

// Auth context
const AuthContext = createContext();

// Auth states
const AUTH_ACTIONS = {
  LOGIN_START: 'LOGIN_START',
  LOGIN_SUCCESS: 'LOGIN_SUCCESS',
  LOGIN_FAILURE: 'LOGIN_FAILURE',
  LOGOUT: 'LOGOUT',
  SET_USER: 'SET_USER',
  SET_LOADING: 'SET_LOADING',
  SET_ERROR: 'SET_ERROR',
  CLEAR_ERROR: 'CLEAR_ERROR',
  SET_INITIALIZING: 'SET_INITIALIZING',
  WEBSOCKET_CONNECT: 'WEBSOCKET_CONNECT',
  WEBSOCKET_DISCONNECT: 'WEBSOCKET_DISCONNECT',
  WEBSOCKET_MESSAGE: 'WEBSOCKET_MESSAGE'
};

// Initial state - IMPORTANT: loading should be FALSE by default
const initialState = {
  user: null,
  isAuthenticated: false,
  loading: false, // Changed to false - this was causing the issue
  error: null,
  initializing: true, // Only this should be true initially
  websocket: null,
  websocketConnected: false,
  realtimeUpdates: []
};

// Auth reducer
const authReducer = (state, action) => {
  switch (action.type) {
    case AUTH_ACTIONS.LOGIN_START:
      return {
        ...state,
        loading: true,
        error: null
      };
      
    case AUTH_ACTIONS.LOGIN_SUCCESS:
      return {
        ...state,
        user: action.payload.user,
        isAuthenticated: true,
        loading: false,
        error: null,
        initializing: false
      };
      
    case AUTH_ACTIONS.LOGIN_FAILURE:
      return {
        ...state,
        user: null,
        isAuthenticated: false,
        loading: false,
        error: action.payload.error,
        initializing: false
      };
      
    case AUTH_ACTIONS.LOGOUT:
      return {
        ...state,
        user: null,
        isAuthenticated: false,
        loading: false,
        error: null,
        initializing: false, // Important: set to false after logout
        websocket: null,
        websocketConnected: false,
        realtimeUpdates: []
      };
      
    case AUTH_ACTIONS.SET_USER:
      return {
        ...state,
        user: action.payload.user,
        isAuthenticated: !!action.payload.user,
        initializing: false,
        loading: false // Ensure loading is false when setting user
      };
      
    case AUTH_ACTIONS.SET_INITIALIZING:
      return {
        ...state,
        initializing: action.payload.initializing,
        loading: false // Separate initializing from loading
      };
      
    case AUTH_ACTIONS.SET_LOADING:
      return {
        ...state,
        loading: action.payload.loading
      };
      
    case AUTH_ACTIONS.SET_ERROR:
      return {
        ...state,
        error: action.payload.error,
        loading: false
      };
      
    case AUTH_ACTIONS.CLEAR_ERROR:
      return {
        ...state,
        error: null
      };
      
    case AUTH_ACTIONS.WEBSOCKET_CONNECT:
      return {
        ...state,
        websocket: action.payload.websocket,
        websocketConnected: true
      };
      
    case AUTH_ACTIONS.WEBSOCKET_DISCONNECT:
      return {
        ...state,
        websocket: null,
        websocketConnected: false
      };
      
    case AUTH_ACTIONS.WEBSOCKET_MESSAGE:
      return {
        ...state,
        realtimeUpdates: [action.payload.message, ...state.realtimeUpdates.slice(0, 49)] // Keep last 50 updates
      };
      
    default:
      return state;
  }
};

// Auth Provider Component
export const AuthProvider = ({ children }) => {
  const [state, dispatch] = useReducer(authReducer, initialState);

  // Initialize authentication on mount
  useEffect(() => {
    const initAuth = async () => {
      // Start initialization but don't set loading to true
      dispatch({ type: AUTH_ACTIONS.SET_INITIALIZING, payload: { initializing: true } });
      
      try {
        if (isAuthenticated()) {
          const user = await getCurrentUser();
          dispatch({ 
            type: AUTH_ACTIONS.SET_USER, 
            payload: { user: user.data || user } 
          });
          
          // Connect to WebSocket for real-time updates
          connectWebSocket(user.data?.id || user.id);
        } else {
          dispatch({ 
            type: AUTH_ACTIONS.SET_USER, 
            payload: { user: null } 
          });
        }
      } catch (error) {
        console.error('Auth initialization failed:', error);
        clearAuth();
        dispatch({ 
          type: AUTH_ACTIONS.SET_USER, 
          payload: { user: null } 
        });
      } finally {
        // Always set initializing to false when done
        dispatch({ type: AUTH_ACTIONS.SET_INITIALIZING, payload: { initializing: false } });
      }
    };

    initAuth();
  }, []);

  // WebSocket connection handler
  const connectWebSocket = (userId) => {
    if (state.websocket || !userId) return;
    
    try {
      const ws = createWebSocketConnection(userId);
      
      ws.onopen = () => {
        console.log('WebSocket connected');
        dispatch({ 
          type: AUTH_ACTIONS.WEBSOCKET_CONNECT, 
          payload: { websocket: ws } 
        });
      };
      
      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('WebSocket message:', message);
          dispatch({ 
            type: AUTH_ACTIONS.WEBSOCKET_MESSAGE, 
            payload: { message } 
          });
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
      
      ws.onclose = () => {
        console.log('WebSocket disconnected');
        dispatch({ type: AUTH_ACTIONS.WEBSOCKET_DISCONNECT });
        
        // Attempt reconnection after 3 seconds
        if (state.user) {
          setTimeout(() => connectWebSocket(userId), 3000);
        }
      };
      
      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        dispatch({ type: AUTH_ACTIONS.WEBSOCKET_DISCONNECT });
      };
      
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
    }
  };

  // Login handler - uses its own loading state
  const login = async (credentials) => {
    dispatch({ type: AUTH_ACTIONS.LOGIN_START });
    
    try {
      const { login: loginAPI } = await import('../services/api');
      const response = await loginAPI(credentials);
      
      const user = response.user || response.data;
      dispatch({ 
        type: AUTH_ACTIONS.LOGIN_SUCCESS, 
        payload: { user } 
      });
      
      // Connect WebSocket
      connectWebSocket(user.id);
      
      return response;
    } catch (error) {
      dispatch({ 
        type: AUTH_ACTIONS.LOGIN_FAILURE, 
        payload: { error: error.message } 
      });
      throw error;
    }
  };

  // Register handler - uses its own loading state
  const register = async (userData) => {
    dispatch({ type: AUTH_ACTIONS.SET_LOADING, payload: { loading: true } });
    
    try {
      const { register: registerAPI } = await import('../services/api');
      const response = await registerAPI(userData);
      
      // If registration includes auto-login
      if (response.token && response.user) {
        const user = response.user || response.data;
        dispatch({ 
          type: AUTH_ACTIONS.LOGIN_SUCCESS, 
          payload: { user } 
        });
        
        connectWebSocket(user.id);
      } else {
        dispatch({ type: AUTH_ACTIONS.SET_LOADING, payload: { loading: false } });
      }
      
      return response;
    } catch (error) {
      dispatch({ 
        type: AUTH_ACTIONS.SET_ERROR, 
        payload: { error: error.message } 
      });
      throw error;
    }
  };

  // Logout handler
  const logout = async () => {
    dispatch({ type: AUTH_ACTIONS.SET_LOADING, payload: { loading: true } });
    
    try {
      // Close WebSocket connection
      if (state.websocket) {
        state.websocket.close();
      }
      
      const { logout: logoutAPI } = await import('../services/api');
      await logoutAPI();
    } catch (error) {
      console.error('Logout API failed:', error);
    } finally {
      dispatch({ type: AUTH_ACTIONS.LOGOUT });
    }
  };

  // Update user info
  const updateUser = (userData) => {
    dispatch({ 
      type: AUTH_ACTIONS.SET_USER, 
      payload: { user: { ...state.user, ...userData } } 
    });
  };

  // Clear error
  const clearError = () => {
    dispatch({ type: AUTH_ACTIONS.CLEAR_ERROR });
  };

  // Get latest real-time updates
  const getRealtimeUpdates = (type = null) => {
    if (!type) return state.realtimeUpdates;
    return state.realtimeUpdates.filter(update => update.type === type);
  };

  // Clear real-time updates
  const clearRealtimeUpdates = () => {
    dispatch({ 
      type: AUTH_ACTIONS.WEBSOCKET_MESSAGE, 
      payload: { message: null } 
    });
  };

  const value = {
    // State
    user: state.user,
    isAuthenticated: state.isAuthenticated,
    loading: state.loading, // This should now be false by default
    error: state.error,
    initializing: state.initializing,
    websocketConnected: state.websocketConnected,
    realtimeUpdates: state.realtimeUpdates,
    
    // Actions
    login,
    register,
    logout,
    updateUser,
    clearError,
    getRealtimeUpdates,
    clearRealtimeUpdates
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to use auth context
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export default AuthContext;