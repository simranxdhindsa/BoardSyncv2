// frontend/src/components/ReverseSync/ReverseAnalysisResults.js
import React, { useState } from 'react';
import { ArrowLeft, CheckCircle, AlertCircle, PlusCircle, Loader2, Check, Calendar, User, Tag, FileText } from 'lucide-react';

const ReverseAnalysisResults = ({ analysisData, selectedCreator, onBack, onCreateTickets, loading }) => {
  const [selectedIssues, setSelectedIssues] = useState([]);
  const [viewMode, setViewMode] = useState('missing'); // 'missing' or 'matched'

  const { matched = [], missing_asana = [] } = analysisData;

  const toggleIssueSelection = (issueId) => {
    setSelectedIssues(prev =>
      prev.includes(issueId)
        ? prev.filter(id => id !== issueId)
        : [...prev, issueId]
    );
  };

  const toggleSelectAll = () => {
    if (selectedIssues.length === missing_asana.length) {
      setSelectedIssues([]);
    } else {
      setSelectedIssues(missing_asana.map(issue => issue.id));
    }
  };

  const handleCreate = () => {
    if (selectedIssues.length === 0 && missing_asana.length > 0) {
      // Create all if none selected
      onCreateTickets([]);
    } else {
      // Create selected
      onCreateTickets(selectedIssues);
    }
  };

  const formatDate = (timestamp) => {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  };

  return (
    <div>
      {/* Summary Cards */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
        gap: '16px',
        marginBottom: '24px'
      }}>
        {/* Matched Card */}
        <div style={{
          padding: '20px',
          background: 'rgba(240, 253, 244, 0.95)',
          borderLeft: '4px solid #10b981',
          borderRadius: '12px',
          boxShadow: '0 4px 12px rgba(0, 0, 0, 0.05)',
          cursor: 'pointer',
          transition: 'all 0.2s',
          border: viewMode === 'matched' ? '2px solid #10b981' : '1px solid rgba(16, 185, 129, 0.2)'
        }}
          onClick={() => setViewMode('matched')}
          onMouseEnter={(e) => e.currentTarget.style.transform = 'translateY(-2px)'}
          onMouseLeave={(e) => e.currentTarget.style.transform = 'translateY(0)'}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: '32px', fontWeight: '700', color: '#059669', marginBottom: '4px' }}>
                {matched.length}
              </div>
              <div style={{ fontSize: '14px', color: '#047857', fontWeight: '500' }}>
                Already in Asana
              </div>
            </div>
            <CheckCircle size={40} color="#10b981" />
          </div>
        </div>

        {/* Missing Card */}
        <div style={{
          padding: '20px',
          background: 'rgba(254, 243, 199, 0.95)',
          borderLeft: '4px solid #f59e0b',
          borderRadius: '12px',
          boxShadow: '0 4px 12px rgba(0, 0, 0, 0.05)',
          cursor: 'pointer',
          transition: 'all 0.2s',
          border: viewMode === 'missing' ? '2px solid #f59e0b' : '1px solid rgba(245, 158, 11, 0.2)'
        }}
          onClick={() => setViewMode('missing')}
          onMouseEnter={(e) => e.currentTarget.style.transform = 'translateY(-2px)'}
          onMouseLeave={(e) => e.currentTarget.style.transform = 'translateY(0)'}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: '32px', fontWeight: '700', color: '#d97706', marginBottom: '4px' }}>
                {missing_asana.length}
              </div>
              <div style={{ fontSize: '14px', color: '#b45309', fontWeight: '500' }}>
                Missing in Asana
              </div>
            </div>
            <AlertCircle size={40} color="#f59e0b" />
          </div>
        </div>
      </div>

      {/* Action Buttons (only for missing view) */}
      {viewMode === 'missing' && missing_asana.length > 0 && (
        <div style={{
          marginBottom: '24px',
          padding: '20px',
          background: 'rgba(255, 255, 255, 0.95)',
          borderRadius: '12px',
          boxShadow: '0 4px 12px rgba(0, 0, 0, 0.05)',
          border: '1px solid rgba(226, 232, 240, 0.8)'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '12px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <button
                onClick={toggleSelectAll}
                style={{
                  padding: '10px 20px',
                  background: 'white',
                  border: '1px solid rgba(226, 232, 240, 0.8)',
                  borderRadius: '8px',
                  fontSize: '14px',
                  fontWeight: '500',
                  color: '#475569',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  transition: 'all 0.2s'
                }}
                onMouseEnter={(e) => {
                  e.target.style.background = 'rgba(241, 245, 249, 0.8)';
                  e.target.style.borderColor = '#3b82f6';
                }}
                onMouseLeave={(e) => {
                  e.target.style.background = 'white';
                  e.target.style.borderColor = 'rgba(226, 232, 240, 0.8)';
                }}
              >
                <Check size={16} style={{ marginRight: '6px' }} />
                {selectedIssues.length === missing_asana.length ? 'Deselect All' : 'Select All'}
              </button>
              <div style={{ fontSize: '14px', color: '#64748b' }}>
                {selectedIssues.length > 0
                  ? `${selectedIssues.length} selected`
                  : 'No tickets selected'
                }
              </div>
            </div>

            <button
              onClick={handleCreate}
              disabled={loading}
              style={{
                padding: '12px 24px',
                background: loading
                  ? 'rgba(148, 163, 184, 0.5)'
                  : 'linear-gradient(135deg, #10b981 0%, #059669 100%)',
                color: 'white',
                border: 'none',
                borderRadius: '8px',
                fontSize: '15px',
                fontWeight: '600',
                cursor: loading ? 'not-allowed' : 'pointer',
                display: 'flex',
                alignItems: 'center',
                transition: 'all 0.2s',
                boxShadow: loading ? 'none' : '0 4px 12px rgba(16, 185, 129, 0.3)'
              }}
              onMouseEnter={(e) => {
                if (!loading) {
                  e.target.style.transform = 'translateY(-2px)';
                  e.target.style.boxShadow = '0 6px 20px rgba(16, 185, 129, 0.4)';
                }
              }}
              onMouseLeave={(e) => {
                if (!loading) {
                  e.target.style.transform = 'translateY(0)';
                  e.target.style.boxShadow = '0 4px 12px rgba(16, 185, 129, 0.3)';
                }
              }}
            >
              {loading ? (
                <>
                  <Loader2 size={18} className="animate-spin" style={{ marginRight: '8px' }} />
                  Creating...
                </>
              ) : (
                <>
                  <PlusCircle size={18} style={{ marginRight: '8px' }} />
                  Create {selectedIssues.length > 0 ? `${selectedIssues.length} Selected` : 'All'} Tickets
                </>
              )}
            </button>
          </div>
        </div>
      )}

      {/* Tickets List */}
      <div style={{
        background: 'rgba(255, 255, 255, 0.95)',
        borderRadius: '12px',
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.05)',
        border: '1px solid rgba(226, 232, 240, 0.8)',
        overflow: 'hidden'
      }}>
        {/* Header */}
        <div style={{
          padding: '16px 20px',
          background: 'rgba(241, 245, 249, 0.8)',
          borderBottom: '1px solid rgba(226, 232, 240, 0.8)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between'
        }}>
          <h3 style={{ fontSize: '16px', fontWeight: '600', color: '#1e293b', margin: 0 }}>
            {viewMode === 'matched' ? 'Matched Tickets' : 'Missing Tickets'}
          </h3>
          <button
            onClick={onBack}
            style={{
              padding: '8px 16px',
              background: 'white',
              border: '1px solid rgba(226, 232, 240, 0.8)',
              borderRadius: '6px',
              fontSize: '13px',
              fontWeight: '500',
              color: '#475569',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              transition: 'all 0.2s'
            }}
          >
            <ArrowLeft size={14} style={{ marginRight: '6px' }} />
            Back
          </button>
        </div>

        {/* Tickets */}
        <div style={{ maxHeight: '600px', overflowY: 'auto' }}>
          {viewMode === 'missing' && missing_asana.length === 0 && (
            <div style={{ padding: '48px', textAlign: 'center', color: '#64748b' }}>
              <CheckCircle size={48} style={{ margin: '0 auto 16px', color: '#10b981' }} />
              <div style={{ fontSize: '16px', fontWeight: '500', marginBottom: '8px' }}>
                All tickets are synced!
              </div>
              <div style={{ fontSize: '14px' }}>
                No missing tickets found for {selectedCreator === 'All' ? 'any user' : selectedCreator}
              </div>
            </div>
          )}

          {viewMode === 'matched' && matched.length === 0 && (
            <div style={{ padding: '48px', textAlign: 'center', color: '#64748b' }}>
              <AlertCircle size={48} style={{ margin: '0 auto 16px', color: '#f59e0b' }} />
              <div style={{ fontSize: '16px', fontWeight: '500', marginBottom: '8px' }}>
                No matched tickets
              </div>
              <div style={{ fontSize: '14px' }}>
                No tickets found that exist in both systems
              </div>
            </div>
          )}

          {/* Missing Tickets */}
          {viewMode === 'missing' && missing_asana.map((issue, index) => (
            <div
              key={issue.id}
              style={{
                padding: '16px 20px',
                borderBottom: index < missing_asana.length - 1 ? '1px solid rgba(226, 232, 240, 0.5)' : 'none',
                transition: 'background 0.2s'
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(241, 245, 249, 0.5)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <div style={{ display: 'flex', alignItems: 'start', gap: '12px' }}>
                {/* Checkbox */}
                <input
                  type="checkbox"
                  checked={selectedIssues.includes(issue.id)}
                  onChange={() => toggleIssueSelection(issue.id)}
                  style={{
                    marginTop: '4px',
                    width: '18px',
                    height: '18px',
                    cursor: 'pointer'
                  }}
                />

                {/* Ticket Info */}
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                    <span style={{
                      padding: '4px 10px',
                      background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                      color: 'white',
                      borderRadius: '6px',
                      fontSize: '13px',
                      fontWeight: '600'
                    }}>
                      {issue.id}
                    </span>
                    <span style={{
                      fontSize: '15px',
                      fontWeight: '600',
                      color: '#1e293b'
                    }}>
                      {issue.summary}
                    </span>
                  </div>

                  {/* Meta Info */}
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '16px', fontSize: '13px', color: '#64748b' }}>
                    {issue.state && (
                      <div style={{ display: 'flex', alignItems: 'center' }}>
                        <FileText size={14} style={{ marginRight: '4px' }} />
                        <span>{issue.state}</span>
                      </div>
                    )}
                    {issue.subsystem && (
                      <div style={{ display: 'flex', alignItems: 'center' }}>
                        <Tag size={14} style={{ marginRight: '4px' }} />
                        <span>{issue.subsystem}</span>
                      </div>
                    )}
                    {issue.created_by && (
                      <div style={{ display: 'flex', alignItems: 'center' }}>
                        <User size={14} style={{ marginRight: '4px' }} />
                        <span>{issue.created_by}</span>
                      </div>
                    )}
                    {issue.created && (
                      <div style={{ display: 'flex', alignItems: 'center' }}>
                        <Calendar size={14} style={{ marginRight: '4px' }} />
                        <span>{formatDate(issue.created)}</span>
                      </div>
                    )}
                  </div>

                  {/* Description Preview */}
                  {issue.description && (
                    <div style={{
                      marginTop: '8px',
                      padding: '8px 12px',
                      background: 'rgba(248, 250, 252, 0.8)',
                      borderRadius: '6px',
                      fontSize: '13px',
                      color: '#475569',
                      maxHeight: '60px',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis'
                    }}>
                      {issue.description.substring(0, 150)}{issue.description.length > 150 ? '...' : ''}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}

          {/* Matched Tickets */}
          {viewMode === 'matched' && matched.map((item, index) => (
            <div
              key={item.youtrack_issue.id}
              style={{
                padding: '16px 20px',
                borderBottom: index < matched.length - 1 ? '1px solid rgba(226, 232, 240, 0.5)' : 'none',
                transition: 'background 0.2s'
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(240, 253, 244, 0.3)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <CheckCircle size={20} color="#10b981" />
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                    <span style={{
                      padding: '4px 10px',
                      background: 'linear-gradient(135deg, #10b981 0%, #059669 100%)',
                      color: 'white',
                      borderRadius: '6px',
                      fontSize: '13px',
                      fontWeight: '600'
                    }}>
                      {item.youtrack_issue.id}
                    </span>
                    <span style={{
                      fontSize: '15px',
                      fontWeight: '600',
                      color: '#1e293b'
                    }}>
                      {item.youtrack_issue.summary}
                    </span>
                  </div>
                  <div style={{ fontSize: '13px', color: '#64748b' }}>
                    Already synced • Asana Task ID: {item.asana_task_id}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default ReverseAnalysisResults;
