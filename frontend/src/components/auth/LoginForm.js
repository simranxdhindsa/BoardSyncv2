// Updated LoginForm Component - components/auth/LoginForm.js
import React, { useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { User, Mail, Lock, Eye, EyeOff, RefreshCw } from 'lucide-react';
import FluidText from '../FluidText';
import '../../styles/auth-glass-theme.css';

const LoginForm = ({ onSuccess }) => {
  const [isLogin, setIsLogin] = useState(true);
  const [showPassword, setShowPassword] = useState(false);
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: ''
  });
  const [formErrors, setFormErrors] = useState({});
  
  // LOCAL loading state instead of using auth context loading
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { login, register, error, clearError } = useAuth();

  const validateForm = () => {
    const errors = {};
    
    if (!formData.username.trim()) {
      errors.username = 'Username is required';
    } else if (formData.username.length < 3) {
      errors.username = 'Username must be at least 3 characters';
    }
    
    if (!isLogin && !formData.email.trim()) {
      errors.email = 'Email is required';
    } else if (!isLogin && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      errors.email = 'Please enter a valid email address';
    }
    
    if (!formData.password) {
      errors.password = 'Password is required';
    } else if (formData.password.length < 6) {
      errors.password = 'Password must be at least 6 characters';
    }
    
    if (!isLogin && formData.password !== formData.confirmPassword) {
      errors.confirmPassword = 'Passwords do not match';
    }
    
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    clearError();
    
    if (!validateForm()) return;
    
    setIsSubmitting(true);
    
    try {
      if (isLogin) {
        await login({
          username: formData.username,
          password: formData.password
        });
      } else {
        await register({
          username: formData.username,
          email: formData.email,
          password: formData.password
        });
      }
      
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      console.error('Authentication failed:', err);
      // Error is handled by the auth context
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleInputChange = (field) => (e) => {
    setFormData(prev => ({
      ...prev,
      [field]: e.target.value
    }));
    
    // Clear field error when user starts typing
    if (formErrors[field]) {
      setFormErrors(prev => ({
        ...prev,
        [field]: ''
      }));
    }
  };

  const toggleMode = () => {
    setIsLogin(!isLogin);
    setFormErrors({});
    clearError();
    setFormData({
      username: '',
      email: '',
      password: '',
      confirmPassword: ''
    });
  };

  return (
    <div className="auth-container">
      <div className="auth-glass-panel">
        {/* Header */}
        <div className="auth-header">
          <div className="auth-logo-container">
            <img 
              src="https://apyhub.com/logo.svg" 
              alt="ApyHub" 
              className="auth-logo"
            />
            <FluidText className="auth-title ml-3" sensitivity={1.5}>
              Asana-YouTrack Sync
            </FluidText>
          </div>
          <FluidText className="auth-title" sensitivity={1.2}>
            {isLogin ? 'Welcome Back' : 'Create Account'}
          </FluidText>
          <p className="auth-subtitle">
            {isLogin 
              ? 'Sign in to access your sync dashboard' 
              : 'Get started with personalized sync settings'
            }
          </p>
        </div>

          {/* Error Display */}
          {error && (
            <div className="auth-error">
              <p>{error}</p>
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="auth-form">
            {/* Username Field */}
            <div className="auth-form-group">
              <label htmlFor="username" className="auth-label">
                Username
              </label>
              <div className="auth-input-container">
                <User className="auth-input-icon" />
                <input
                  type="text"
                  id="username"
                  value={formData.username}
                  onChange={handleInputChange('username')}
                  className="auth-input auth-input-with-icon"
                  placeholder="Enter your username"
                  disabled={isSubmitting}
                />
              </div>
              {formErrors.username && (
                <p className="text-red-500 text-xs mt-1">{formErrors.username}</p>
              )}
            </div>

            {/* Email Field (Registration only) */}
            {!isLogin && (
              <div className="auth-form-group">
                <label htmlFor="email" className="auth-label">
                  Email Address
                </label>
                <div className="auth-input-container">
                  <Mail className="auth-input-icon" />
                  <input
                    type="email"
                    id="email"
                    value={formData.email}
                    onChange={handleInputChange('email')}
                    className="auth-input auth-input-with-icon"
                    placeholder="Enter your email"
                    disabled={isSubmitting}
                  />
                </div>
                {formErrors.email && (
                  <p className="text-red-500 text-xs mt-1">{formErrors.email}</p>
                )}
              </div>
            )}

            {/* Password Field */}
            <div className="auth-form-group">
              <label htmlFor="password" className="auth-label">
                Password
              </label>
              <div className="auth-input-container">
                <Lock className="auth-input-icon" />
                <input
                  type={showPassword ? 'text' : 'password'}
                  id="password"
                  value={formData.password}
                  onChange={handleInputChange('password')}
                  className="auth-input auth-input-with-icon"
                  style={{ paddingRight: '3rem' }}
                  placeholder="Enter your password"
                  disabled={isSubmitting}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="auth-input-toggle"
                  disabled={isSubmitting}
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {formErrors.password && (
                <p className="text-red-500 text-xs mt-1">{formErrors.password}</p>
              )}
            </div>

            {/* Confirm Password Field (Registration only) */}
            {!isLogin && (
              <div className="auth-form-group">
                <label htmlFor="confirmPassword" className="auth-label">
                  Confirm Password
                </label>
                <div className="auth-input-container">
                  <Lock className="auth-input-icon" />
                  <input
                    type={showPassword ? 'text' : 'password'}
                    id="confirmPassword"
                    value={formData.confirmPassword}
                    onChange={handleInputChange('confirmPassword')}
                    className="auth-input auth-input-with-icon"
                    placeholder="Confirm your password"
                    disabled={isSubmitting}
                  />
                </div>
                {formErrors.confirmPassword && (
                  <p className="text-red-500 text-xs mt-1">{formErrors.confirmPassword}</p>
                )}
              </div>
            )}

            {/* Submit Button */}
            <button
              type="submit"
              disabled={isSubmitting}
              className="auth-button"
            >
              {isSubmitting ? (
                <>
                  <RefreshCw className="auth-spinner" />
                  {isLogin ? 'Signing in...' : 'Creating account...'}
                </>
              ) : (
                <FluidText sensitivity={1}>
                  {isLogin ? 'Sign In' : 'Create Account'}
                </FluidText>
              )}
            </button>

            {/* Mode Toggle */}
            <div className="text-center mt-4">
              <button
                type="button"
                onClick={toggleMode}
                disabled={isSubmitting}
                className="auth-toggle-link"
              >
                {isLogin 
                  ? "Don't have an account? Sign up" 
                  : "Already have an account? Sign in"
                }
              </button>
            </div>
          </form>

          {/* Footer Info */}
          <div className="auth-footer">
            <p className="auth-footer-text">
              By {isLogin ? 'signing in' : 'creating an account'}, you can save your API configurations, 
              view sync history, and access advanced features like rollback and real-time updates.
            </p>
          </div>
        </div>
      </div>
    );
  };


export default LoginForm;