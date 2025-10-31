// frontend/src/components/ReverseSync/CreatorFilter.js
import React from 'react';
import { Search, UserCheck, Loader2 } from 'lucide-react';

const CreatorFilter = ({ users, selectedCreator, onCreatorChange, onAnalyze, loading }) => {
  return (
    <div className="glass-container" style={{
      padding: '32px',
      background: 'rgba(255, 255, 255, 0.95)',
      backdropFilter: 'blur(10px)',
      borderRadius: '16px',
      border: '1px solid rgba(255, 255, 255, 0.8)',
      boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1)'
    }}>
      <div style={{ marginBottom: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: '12px' }}>
          <UserCheck size={24} style={{
            marginRight: '12px',
            background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent'
          }} />
          <h2 style={{
            fontSize: '20px',
            fontWeight: '600',
            margin: 0,
            color: '#1e293b'
          }}>
            Select Creator
          </h2>
        </div>
        <p style={{ fontSize: '14px', color: '#64748b', margin: 0 }}>
          Choose which YouTrack user's issues you want to create in Asana
        </p>
      </div>

      {/* Creator Dropdown */}
      <div style={{ marginBottom: '24px' }}>
        <label style={{
          display: 'block',
          fontSize: '14px',
          fontWeight: '600',
          color: '#475569',
          marginBottom: '8px'
        }}>
          Created By
        </label>
        <div style={{ position: 'relative' }}>
          <select
            value={selectedCreator}
            onChange={(e) => onCreatorChange(e.target.value)}
            disabled={loading}
            style={{
              width: '100%',
              padding: '12px 16px',
              fontSize: '15px',
              border: '1px solid rgba(226, 232, 240, 0.8)',
              borderRadius: '8px',
              background: 'white',
              color: '#1e293b',
              cursor: 'pointer',
              outline: 'none',
              transition: 'border 0.2s',
              appearance: 'none',
              paddingRight: '40px'
            }}
            onFocus={(e) => e.target.style.borderColor = '#3b82f6'}
            onBlur={(e) => e.target.style.borderColor = 'rgba(226, 232, 240, 0.8)'}
          >
            <option value="All">All Users</option>
            {users.map((user) => (
              <option key={user.id} value={user.fullName || user.login}>
                {user.fullName || user.login} {user.email ? `(${user.email})` : ''}
              </option>
            ))}
          </select>
          <div style={{
            position: 'absolute',
            right: '16px',
            top: '50%',
            transform: 'translateY(-50%)',
            pointerEvents: 'none',
            color: '#64748b'
          }}>
            ▼
          </div>
        </div>
        <p style={{ fontSize: '13px', color: '#94a3b8', margin: '8px 0 0 0' }}>
          {selectedCreator === 'All'
            ? `Analyzing all issues in the project`
            : `Analyzing issues created by ${selectedCreator}`
          }
        </p>
      </div>

      {/* Users Preview */}
      {users.length > 0 && (
        <div style={{
          padding: '16px',
          background: 'rgba(241, 245, 249, 0.5)',
          borderRadius: '8px',
          marginBottom: '24px'
        }}>
          <div style={{
            fontSize: '13px',
            color: '#64748b',
            marginBottom: '8px'
          }}>
            Available Users ({users.length})
          </div>
          <div style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '8px'
          }}>
            {users.slice(0, 10).map((user) => (
              <div
                key={user.id}
                onClick={() => onCreatorChange(user.fullName || user.login)}
                style={{
                  padding: '4px 12px',
                  background: selectedCreator === (user.fullName || user.login) ? 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)' : 'white',
                  color: selectedCreator === (user.fullName || user.login) ? 'white' : '#475569',
                  borderRadius: '16px',
                  fontSize: '13px',
                  border: selectedCreator === (user.fullName || user.login) ? '1px solid #3b82f6' : '1px solid rgba(226, 232, 240, 0.8)',
                  cursor: 'pointer',
                  transition: 'all 0.2s'
                }}
                onMouseEnter={(e) => {
                  if (selectedCreator !== (user.fullName || user.login)) {
                    e.target.style.background = 'rgba(59, 130, 246, 0.1)';
                    e.target.style.borderColor = '#3b82f6';
                  }
                }}
                onMouseLeave={(e) => {
                  if (selectedCreator !== (user.fullName || user.login)) {
                    e.target.style.background = 'white';
                    e.target.style.borderColor = 'rgba(226, 232, 240, 0.8)';
                  }
                }}
              >
                {user.fullName || user.login}
              </div>
            ))}
            {users.length > 10 && (
              <div style={{
                padding: '4px 12px',
                background: 'rgba(226, 232, 240, 0.3)',
                borderRadius: '16px',
                fontSize: '13px',
                color: '#64748b'
              }}>
                +{users.length - 10} more
              </div>
            )}
          </div>
        </div>
      )}

      {/* Analyze Button */}
      <button
        onClick={onAnalyze}
        disabled={loading}
        style={{
          width: '100%',
          padding: '14px 24px',
          background: loading
            ? 'rgba(148, 163, 184, 0.5)'
            : 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
          color: 'white',
          border: 'none',
          borderRadius: '8px',
          fontSize: '16px',
          fontWeight: '600',
          cursor: loading ? 'not-allowed' : 'pointer',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          transition: 'all 0.2s',
          boxShadow: loading ? 'none' : '0 4px 12px rgba(59, 130, 246, 0.3)'
        }}
        onMouseEnter={(e) => {
          if (!loading) {
            e.target.style.transform = 'translateY(-2px)';
            e.target.style.boxShadow = '0 6px 20px rgba(59, 130, 246, 0.4)';
          }
        }}
        onMouseLeave={(e) => {
          if (!loading) {
            e.target.style.transform = 'translateY(0)';
            e.target.style.boxShadow = '0 4px 12px rgba(59, 130, 246, 0.3)';
          }
        }}
      >
        {loading ? (
          <>
            <Loader2 size={20} className="animate-spin" style={{ marginRight: '8px' }} />
            Analyzing...
          </>
        ) : (
          <>
            <Search size={20} style={{ marginRight: '8px' }} />
            Analyze Tickets
          </>
        )}
      </button>

      {/* Info Box */}
      <div style={{
        marginTop: '24px',
        padding: '16px',
        background: 'rgba(239, 246, 255, 0.5)',
        borderLeft: '4px solid #3b82f6',
        borderRadius: '8px',
        fontSize: '14px',
        color: '#475569'
      }}>
        <strong style={{ color: '#1e40af', display: 'block', marginBottom: '8px' }}>
          How it works:
        </strong>
        <ul style={{ margin: 0, paddingLeft: '20px' }}>
          <li>Select a creator to filter YouTrack issues</li>
          <li>Analysis will show Matched (already in Asana) and Missing tickets</li>
          <li>You can create all missing tickets or select specific ones</li>
          <li>Ticket titles will keep YouTrack ID format (e.g., "ARD-123 Title")</li>
        </ul>
      </div>
    </div>
  );
};

export default CreatorFilter;
