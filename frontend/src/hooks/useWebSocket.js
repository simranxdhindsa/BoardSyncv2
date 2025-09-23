import { useState, useEffect, useRef, useCallback } from 'react';
import { useAuth } from '../contexts/AuthContext';

export const useWebSocket = (autoConnect = true) => {
  const { user, isAuthenticated } = useAuth();
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState(null);
  const [messages, setMessages] = useState([]);
  const [lastMessage, setLastMessage] = useState(null);
  
  const wsRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const reconnectAttemptsRef = useRef(0);
  const maxReconnectAttempts = 5;
  const baseReconnectDelay = 1000;

  // Get WebSocket URL
  const getWebSocketUrl = useCallback(() => {
    const API_BASE = process.env.NODE_ENV === 'production'
      ? process.env.REACT_APP_API_URL || 'https://boardsyncapi.onrender.com'
      : 'http://localhost:8080';
    
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsBase = API_BASE.replace(/^https?:/, wsProtocol);
    
    return `${wsBase}/ws?user_id=${user?.id}`;
  }, [user?.id]);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (!isAuthenticated || !user?.id || wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    if (connecting) return;

    setConnecting(true);
    setError(null);

    try {
      const ws = new WebSocket(getWebSocketUrl());
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('WebSocket connected');
        setConnected(true);
        setConnecting(false);
        setError(null);
        reconnectAttemptsRef.current = 0;
        
        // Send heartbeat to maintain connection
        const heartbeatInterval = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'heartbeat' }));
          } else {
            clearInterval(heartbeatInterval);
          }
        }, 30000);
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('WebSocket message received:', message);
          
          // Update messages list
          setMessages(prev => [message, ...prev.slice(0, 99)]); // Keep last 100 messages
          setLastMessage(message);
          
          // Handle specific message types
          handleMessage(message);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason);
        setConnected(false);
        setConnecting(false);
        wsRef.current = null;

        // Attempt reconnection if it wasn't a clean close
        if (event.code !== 1000 && event.code !== 1001 && isAuthenticated) {
          scheduleReconnect();
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        setError('Connection failed');
        setConnecting(false);
      };

    } catch (err) {
      console.error('Failed to create WebSocket connection:', err);
      setError('Failed to connect');
      setConnecting(false);
    }
  }, [isAuthenticated, user?.id, connecting, getWebSocketUrl]);

  // Disconnect WebSocket
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close(1000, 'User disconnected');
      wsRef.current = null;
    }

    setConnected(false);
    setConnecting(false);
  }, []);

  // Schedule reconnection with exponential backoff
  const scheduleReconnect = useCallback(() => {
    if (reconnectAttemptsRef.current >= maxReconnectAttempts) {
      setError('Connection failed after multiple attempts');
      return;
    }

    const delay = baseReconnectDelay * Math.pow(2, reconnectAttemptsRef.current);
    reconnectAttemptsRef.current += 1;

    console.log(`Scheduling reconnect attempt ${reconnectAttemptsRef.current} in ${delay}ms`);

    reconnectTimeoutRef.current = setTimeout(() => {
      if (isAuthenticated && user?.id) {
        connect();
      }
    }, delay);
  }, [isAuthenticated, user?.id, connect]);

  // Send message through WebSocket
  const sendMessage = useCallback((message) => {
    if (!connected || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket not connected, cannot send message');
      return false;
    }

    try {
      wsRef.current.send(JSON.stringify(message));
      return true;
    } catch (err) {
      console.error('Failed to send WebSocket message:', err);
      return false;
    }
  }, [connected]);

  // Handle incoming messages
  const handleMessage = useCallback((message) => {
    switch (message.type) {
      case 'sync_start':
        console.log('Sync operation started:', message.data);
        break;
      case 'sync_progress':
        console.log('Sync progress:', message.data);
        break;
      case 'sync_complete':
        console.log('Sync completed:', message.data);
        break;
      case 'sync_error':
        console.error('Sync error:', message.data);
        break;
      case 'rollback':
        console.log('Rollback operation:', message.data);
        break;
      case 'notification':
        console.log('Notification:', message.data);
        break;
      case 'heartbeat':
        // Respond to server heartbeat
        sendMessage({ type: 'heartbeat', data: { status: 'alive' } });
        break;
      default:
        console.log('Unknown message type:', message.type);
    }
  }, [sendMessage]);

  // Get messages by type
  const getMessagesByType = useCallback((type) => {
    return messages.filter(msg => msg.type === type);
  }, [messages]);

  // Get latest message by type
  const getLatestMessageByType = useCallback((type) => {
    return messages.find(msg => msg.type === type);
  }, [messages]);

  // Clear messages
  const clearMessages = useCallback(() => {
    setMessages([]);
    setLastMessage(null);
  }, []);

  // Clear messages by type
  const clearMessagesByType = useCallback((type) => {
    setMessages(prev => prev.filter(msg => msg.type !== type));
    if (lastMessage?.type === type) {
      setLastMessage(messages.find(msg => msg.type !== type) || null);
    }
  }, [messages, lastMessage]);

  // Auto-connect effect
  useEffect(() => {
    if (autoConnect && isAuthenticated && user?.id && !connected && !connecting) {
      connect();
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [autoConnect, isAuthenticated, user?.id, connected, connecting, connect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  // Handle authentication changes
  useEffect(() => {
    if (!isAuthenticated) {
      disconnect();
    }
  }, [isAuthenticated, disconnect]);

  return {
    // Connection state
    connected,
    connecting,
    error,
    reconnectAttempts: reconnectAttemptsRef.current,
    
    // Messages
    messages,
    lastMessage,
    
    // Actions
    connect,
    disconnect,
    sendMessage,
    
    // Message utilities
    getMessagesByType,
    getLatestMessageByType,
    clearMessages,
    clearMessagesByType
  };
};