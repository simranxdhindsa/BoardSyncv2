import React, { useState, useEffect } from 'react';

let notificationId = 0;
const notifications = [];
let notificationListeners = [];

// Global notification manager
export const showNotification = ({ type, message, duration = 4000 }) => {
  const notification = {
    id: ++notificationId,
    type,
    message,
    duration,
    timestamp: Date.now()
  };
  
  notifications.push(notification);
  
  // Notify all listeners
  notificationListeners.forEach(listener => listener([...notifications]));
  
  // Auto-remove after duration
  setTimeout(() => {
    removeNotification(notification.id);
  }, duration);
  
  return notification.id;
};

export const removeNotification = (id) => {
  const index = notifications.findIndex(n => n.id === id);
  if (index > -1) {
    notifications.splice(index, 1);
    notificationListeners.forEach(listener => listener([...notifications]));
  }
};

const Notification3D = () => {
  const [activeNotifications, setActiveNotifications] = useState([]);

  useEffect(() => {
    const listener = (updatedNotifications) => {
      setActiveNotifications(updatedNotifications);
    };
    
    notificationListeners.push(listener);
    
    return () => {
      const index = notificationListeners.indexOf(listener);
      if (index > -1) {
        notificationListeners.splice(index, 1);
      }
    };
  }, []);

  const getNotificationStyles = (type) => {
    const baseStyles = {
      position: 'fixed',
      top: '20px',
      right: '20px',
      minWidth: '300px',
      maxWidth: '400px',
      padding: '16px 20px',
      borderRadius: '16px',
      backdropFilter: 'blur(24px) saturate(180%)',
      WebkitBackdropFilter: 'blur(24px) saturate(180%)',
      border: '1px solid',
      boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1), inset 0 1px 0 rgba(255, 255, 255, 0.3)',
      color: '#1e293b',
      fontWeight: '500',
      fontSize: '14px',
      zIndex: 9999,
      animation: 'slideInRight 0.4s cubic-bezier(0.25, 0.46, 0.45, 0.94)',
      cursor: 'pointer',
      transition: 'all 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94)'
    };

    switch (type) {
      case 'success':
        return {
          ...baseStyles,
          background: 'rgba(16, 185, 129, 0.15)',
          borderColor: 'rgba(16, 185, 129, 0.3)',
          color: '#065f46'
        };
      case 'error':
        return {
          ...baseStyles,
          background: 'rgba(239, 68, 68, 0.15)',
          borderColor: 'rgba(239, 68, 68, 0.3)',
          color: '#991b1b'
        };
      case 'info':
        return {
          ...baseStyles,
          background: 'rgba(59, 130, 246, 0.15)',
          borderColor: 'rgba(59, 130, 246, 0.3)',
          color: '#1e40af'
        };
      case 'warning':
        return {
          ...baseStyles,
          background: 'rgba(245, 158, 11, 0.15)',
          borderColor: 'rgba(245, 158, 11, 0.3)',
          color: '#92400e'
        };
      default:
        return baseStyles;
    }
  };

  const getNotificationIcon = (type) => {
    switch (type) {
      case 'success':
        return '✅';
      case 'error':
        return '❌';
      case 'info':
        return 'ℹ️';
      case 'warning':
        return '⚠️';
      default:
        return '📝';
    }
  };

  return (
    <>
      <style>
        {`
          @keyframes slideInRight {
            from {
              transform: translateX(100%) scale(0.95);
              opacity: 0;
            }
            to {
              transform: translateX(0) scale(1);
              opacity: 1;
            }
          }
          
          @keyframes slideOutRight {
            from {
              transform: translateX(0) scale(1);
              opacity: 1;
            }
            to {
              transform: translateX(100%) scale(0.95);
              opacity: 0;
            }
          }
          
          .notification-3d:hover {
            transform: translateY(-2px) scale(1.02);
            box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15), inset 0 1px 0 rgba(255, 255, 255, 0.4);
          }
          
          .notification-exit {
            animation: slideOutRight 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94) forwards;
          }
        `}
      </style>
      
      <div style={{ position: 'fixed', top: 0, right: 0, zIndex: 9999, pointerEvents: 'none' }}>
        {activeNotifications.map((notification, index) => (
          <div
            key={notification.id}
            className="notification-3d"
            style={{
              ...getNotificationStyles(notification.type),
              top: `${20 + index * 80}px`,
              pointerEvents: 'auto'
            }}
            onClick={() => removeNotification(notification.id)}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <span style={{ fontSize: '18px' }}>
                {getNotificationIcon(notification.type)}
              </span>
              <div style={{ flex: 1 }}>
                <div style={{ 
                  fontWeight: '600', 
                  marginBottom: notification.message.length > 50 ? '4px' : '0' 
                }}>
                  {notification.type === 'success' && 'Success'}
                  {notification.type === 'error' && 'Error'}
                  {notification.type === 'info' && 'Information'}
                  {notification.type === 'warning' && 'Warning'}
                </div>
                <div style={{ opacity: 0.9 }}>
                  {notification.message}
                </div>
              </div>
              <div style={{ 
                fontSize: '12px', 
                opacity: 0.6, 
                fontWeight: '400',
                marginLeft: '8px'
              }}>
                Click to dismiss
              </div>
            </div>
          </div>
        ))}
      </div>
    </>
  );
};

export default Notification3D;