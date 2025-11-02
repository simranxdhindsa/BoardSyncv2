// frontend/src/components/ReverseSync/ReverseSyncPage.js
import React, { useState, useEffect } from 'react';
import { ArrowLeft, ArrowRight, Users, CheckCircle2, AlertCircle } from 'lucide-react';
import { getYouTrackUsers, reverseAnalyzeTickets, reverseCreateTickets } from '../../services/api';
import CreatorFilter from './CreatorFilter';
import ReverseAnalysisResults from './ReverseAnalysisResults';

const ReverseSyncPage = ({ onBack }) => {
  // State
  const [step, setStep] = useState(1); // 1 = Select Creator, 2 = Analysis Results
  const [users, setUsers] = useState([]);
  const [selectedCreator, setSelectedCreator] = useState('All');
  const [analysisData, setAnalysisData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [creatingTickets, setCreatingTickets] = useState(false);

  // Load YouTrack users on mount
  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    try {
      setLoading(true);
      setError(null);
      const usersData = await getYouTrackUsers();
      setUsers(usersData);
    } catch (err) {
      setError('Failed to load YouTrack users: ' + err.message);
      console.error('Error loading users:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleAnalyze = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await reverseAnalyzeTickets(selectedCreator);
      setAnalysisData(data);
      setStep(2);
    } catch (err) {
      setError('Analysis failed: ' + err.message);
      console.error('Error analyzing:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTickets = async (selectedIssueIDs = []) => {
    try {
      setCreatingTickets(true);
      setError(null);
      const result = await reverseCreateTickets(selectedIssueIDs);

      // Log success message to console
      console.log(`Successfully created ${result.data.success_count}/${result.data.total_tickets} tickets!`);

      // Re-analyze to refresh data
      const data = await reverseAnalyzeTickets(selectedCreator);
      setAnalysisData(data);
    } catch (err) {
      setError('Failed to create tickets: ' + err.message);
      console.error('Error creating tickets:', err);
    } finally {
      setCreatingTickets(false);
    }
  };

  const handleBackToCreatorSelection = () => {
    setStep(1);
    setAnalysisData(null);
  };

  return (
    <div className="reverse-sync-page" style={{ maxWidth: '1400px', margin: '0 auto', padding: '20px' }}>
      {/* Header */}
      <div className="glass-container" style={{
        marginBottom: '24px',
        padding: '20px 24px',
        background: 'rgba(255, 255, 255, 0.95)',
        backdropFilter: 'blur(10px)',
        borderRadius: '16px',
        border: '1px solid rgba(255, 255, 255, 0.8)',
        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1)'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <button
              onClick={onBack}
              className="glass-button"
              style={{
                padding: '8px 12px',
                marginRight: '16px',
                background: 'rgba(255, 255, 255, 0.8)',
                border: '1px solid rgba(226, 232, 240, 0.8)',
                borderRadius: '8px',
                display: 'flex',
                alignItems: 'center',
                cursor: 'pointer'
              }}
            >
              <ArrowLeft size={18} />
            </button>
            <div>
              <h1 style={{
                fontSize: '24px',
                fontWeight: '700',
                margin: 0,
                background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                WebkitBackgroundClip: 'text',
                WebkitTextFillColor: 'transparent'
              }}>
                Reverse Sync (YouTrack → Asana)
              </h1>
              <p style={{
                fontSize: '14px',
                color: '#64748b',
                margin: '4px 0 0 0'
              }}>
                Create Asana tickets from YouTrack issues
              </p>
            </div>
          </div>

          {/* Progress indicator */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              padding: '8px 16px',
              background: step === 1 ? 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)' : 'rgba(226, 232, 240, 0.5)',
              borderRadius: '20px',
              color: step === 1 ? 'white' : '#64748b',
              fontSize: '14px',
              fontWeight: '600'
            }}>
              <Users size={16} style={{ marginRight: '6px' }} />
              1. Select Creator
            </div>
            <ArrowRight size={20} color="#94a3b8" />
            <div style={{
              display: 'flex',
              alignItems: 'center',
              padding: '8px 16px',
              background: step === 2 ? 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)' : 'rgba(226, 232, 240, 0.5)',
              borderRadius: '20px',
              color: step === 2 ? 'white' : '#64748b',
              fontSize: '14px',
              fontWeight: '600'
            }}>
              <CheckCircle2 size={16} style={{ marginRight: '6px' }} />
              2. Review & Create
            </div>
          </div>
        </div>
      </div>

      {/* Error Alert */}
      {error && (
        <div style={{
          marginBottom: '24px',
          padding: '16px',
          background: 'rgba(254, 226, 226, 0.9)',
          borderLeft: '4px solid #ef4444',
          borderRadius: '8px',
          display: 'flex',
          alignItems: 'center'
        }}>
          <AlertCircle size={20} color="#ef4444" style={{ marginRight: '12px' }} />
          <span style={{ color: '#991b1b', fontSize: '14px' }}>{error}</span>
        </div>
      )}

      {/* Step 1: Creator Filter */}
      {step === 1 && (
        <CreatorFilter
          users={users}
          selectedCreator={selectedCreator}
          onCreatorChange={setSelectedCreator}
          onAnalyze={handleAnalyze}
          loading={loading}
        />
      )}

      {/* Step 2: Analysis Results */}
      {step === 2 && analysisData && (
        <ReverseAnalysisResults
          analysisData={analysisData}
          selectedCreator={selectedCreator}
          onBack={handleBackToCreatorSelection}
          onCreateTickets={handleCreateTickets}
          onReanalyze={handleAnalyze}
          loading={creatingTickets || loading}
        />
      )}
    </div>
  );
};

export default ReverseSyncPage;
