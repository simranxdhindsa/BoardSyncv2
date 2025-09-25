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

// Initial state
const initialState = {
  user: null,
  isAuthenticated: false,
  loading: false,
  error: null,
  initializing: true,
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
        initializing: false,
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
        loading: false
      };
      
    case AUTH_ACTIONS.SET_INITIALIZING:
      return {
        ...state,
        initializing: action.payload.initializing,
        loading: false
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
        realtimeUpdates: [action.payload.message, ...state.realtimeUpdates.slice(0, 49)]
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
      console.log('🔍 Auth: Initializing authentication...');
      dispatch({ type: AUTH_ACTIONS.SET_INITIALIZING, payload: { initializing: true } });
      
      try {
        if (isAuthenticated()) {
          console.log('🔍 Auth: Token found, getting current user...');
          const response = await getCurrentUser();
          const user = response.data || response.user || response;
          
          console.log('🔍 Auth: Current user retrieved:', user);
          dispatch({ 
            type: AUTH_ACTIONS.SET_USER, 
            payload: { user: user } 
          });
          
          // Connect to WebSocket for real-time updates
          if (user.id) {
            connectWebSocket(user.id);
          }
        } else {
          console.log('🔍 Auth: No token found');
          dispatch({ 
            type: AUTH_ACTIONS.SET_USER, 
            payload: { user: null } 
          });
        }
      } catch (error) {
        console.error('🔍 Auth: Initialization failed:', error);
        clearAuth();
        dispatch({ 
          type: AUTH_ACTIONS.SET_USER, 
          payload: { user: null } 
        });
      } finally {
        console.log('🔍 Auth: Initialization complete');
        dispatch({ type: AUTH_ACTIONS.SET_INITIALIZING, payload: { initializing: false } });
      }
    };

    initAuth();
  }, []);

  // WebSocket connection handler
  const connectWebSocket = (userId) => {
    if (state.websocket || !userId) return;
    
    try {
      console.log('🔍 Auth: Connecting WebSocket for user:', userId);
      const ws = createWebSocketConnection(userId);
      
      ws.onopen = () => {
        console.log('🔍 Auth: WebSocket connected');
        dispatch({ 
          type: AUTH_ACTIONS.WEBSOCKET_CONNECT, 
          payload: { websocket: ws } 
        });
      };
      
      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('🔍 Auth: WebSocket message:', message);
          dispatch({ 
            type: AUTH_ACTIONS.WEBSOCKET_MESSAGE, 
            payload: { message } 
          });
        } catch (error) {
          console.error('🔍 Auth: Failed to parse WebSocket message:', error);
        }
      };
      
      ws.onclose = () => {
        console.log('🔍 Auth: WebSocket disconnected');
        dispatch({ type: AUTH_ACTIONS.WEBSOCKET_DISCONNECT });
        
        // Attempt reconnection after 3 seconds
        if (state.user) {
          setTimeout(() => connectWebSocket(userId), 3000);
        }
      };
      
      ws.onerror = (error) => {
        console.error('🔍 Auth: WebSocket error:', error);
        dispatch({ type: AUTH_ACTIONS.WEBSOCKET_DISCONNECT });
      };
      
    } catch (error) {
      console.error('🔍 Auth: Failed to create WebSocket connection:', error);
    }
  };

  // Login handler
  const login = async (credentials) => {
    console.log('🔍 Auth: Starting login process...');
    dispatch({ type: AUTH_ACTIONS.LOGIN_START });
    
    try {
      const { login: loginAPI } = await import('../services/api');
      const response = await loginAPI(credentials);
      
      console.log('🔍 Auth: Login API response:', response);
      
      // Handle different response structures
      const user = response.data?.user || response.user || {
        id: response.data?.id || response.id,
        username: response.data?.username || response.username,
        email: response.data?.email || response.email
      };
      
      console.log('🔍 Auth: Extracted user from login:', user);
      
      if (!user.id && !user.username) {
        throw new Error('Invalid login response format');
      }
      
      dispatch({ 
        type: AUTH_ACTIONS.LOGIN_SUCCESS, 
        payload: { user } 
      });
      
      // Connect WebSocket
      connectWebSocket(user.id);
      
      return response;
    } catch (error) {
      console.error('🔍 Auth: Login failed:', error);
      dispatch({ 
        type: AUTH_ACTIONS.LOGIN_FAILURE, 
        payload: { error: error.message } 
      });
      throw error;
    }
  };

  // Register handler
  const register = async (userData) => {
    console.log('🔍 Auth: Starting registration process...');
    dispatch({ type: AUTH_ACTIONS.SET_LOADING, payload: { loading: true } });
    
    try {
      const { register: registerAPI } = await import('../services/api');
      const response = await registerAPI(userData);
      
      console.log('🔍 Auth: Registration API response:', response);
      
      // If registration includes auto-login
      if (response.token && response.user) {
        const user = response.user || response.data;
        console.log('🔍 Auth: Auto-login after registration:', user);
        
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
      console.error('🔍 Auth: Registration failed:', error);
      dispatch({ 
        type: AUTH_ACTIONS.SET_ERROR, 
        payload: { error: error.message } 
      });
      throw error;
    }
  };

  // Logout handler
  const logout = async () => {
    console.log('🔍 Auth: Starting logout process...');
    dispatch({ type: AUTH_ACTIONS.SET_LOADING, payload: { loading: true } });
    
    try {
      // Close WebSocket connection
      if (state.websocket) {
        state.websocket.close();
      }
      
      const { logout: logoutAPI } = await import('../services/api');
      await logoutAPI();
      
      console.log('🔍 Auth: Logout completed');
    } catch (error) {
      console.error('🔍 Auth: Logout API failed:', error);
    } finally {
      dispatch({ type: AUTH_ACTIONS.LOGOUT });
    }
  };

  // Update user info
  const updateUser = (userData) => {
    console.log('🔍 Auth: Updating user data:', userData);
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
    loading: state.loading,
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