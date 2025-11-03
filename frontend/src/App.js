// Updated App.js - Matches your existing structure
import React, { useState } from 'react';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Dashboard from './components/Dashboard';
import AnalysisResults from './components/AnalysisResults';
import NavBar from './components/NavBar';
import LuxuryBackground from './components/Background';
import LoginForm from './components/auth/LoginForm';
import UserSettings from './components/settings/UserSettings';
import SyncHistory from './components/SyncHistory';
import AuditLogs from './components/AuditLogs';
import ReverseSyncPage from './components/ReverseSync/ReverseSyncPage';
import { analyzeTickets, syncSingleTicket, createSingleTicket, createMissingTickets } from './services/api';
import './styles/glass-theme.css';
import { Settings, History, FileText, RefreshCw } from 'lucide-react';


// Main App Component (wrapped in AuthProvider)
function AppContent() {
  const { isAuthenticated, user, initializing } = useAuth();
  
  // App state
  const [currentView, setCurrentView] = useState('dashboard');
  const [selectedColumn, setSelectedColumn] = useState('');
  const [analysisData, setAnalysisData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [navLeft, setNavLeft] = useState(null);
  const [navRight, setNavRight] = useState(null);

  // Show loading spinner ONLY while checking authentication (initializing)
  if (initializing) {
    return (
      <div className="min-h-screen flex items-center justify-center" style={{ zIndex: 2000, background: 'linear-gradient(135deg, #f8fafc 0%, #e2e8f0 50%, #cbd5e1 100%)' }}>
        <div className="flex items-center bg-white/80 backdrop-blur-md rounded-lg p-6 shadow-lg">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mr-3"></div>
          <span className="text-gray-600 font-medium">Initializing application...</span>
        </div>
      </div>
    );
  }

  // Show login form if not authenticated - WITH BACKGROUND
  if (!isAuthenticated) {
    return (
      <div className="App" style={{ 
        position: 'relative', 
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column'
      }}>
        {/* Background for login page */}
        <div className="luxury-background-container">
          <LuxuryBackground 
            currentView="login"
            analysisData={null}
            selectedColumn=""
            isLoading={false}
          />
        </div>
        
        <div className="luxury-canvas-blur-separator" />
        
        {/* Login form with proper layering */}
        <div className="luxury-canvas-content-layer" style={{ 
          flex: '1', 
          position: 'relative',
          zIndex: 1
        }}>
          <LoginForm onSuccess={() => setCurrentView('dashboard')} />
        </div>
      </div>
    );
  }

  // Handle column selection
  const handleColumnSelect = (column) => {
    setSelectedColumn(column);
  };

  // Handle analyze action
  const handleAnalyze = async () => {
    if (!selectedColumn) return;

    setLoading(true);
    
    try {
      const data = await analyzeTickets(selectedColumn);
      // Store both the analysis data and the column it was analyzed for
      setAnalysisData({
        ...data,
        analyzedColumn: selectedColumn
      });
      setCurrentView('results');
    } catch (error) {
      console.error('Analysis failed:', error);
      alert('Analysis failed: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // Handle re-analyze
  const handleReAnalyze = async (columnToAnalyze) => {
    setLoading(true);
    try {
      const data = await analyzeTickets(columnToAnalyze);
      setAnalysisData({
        ...data,
        analyzedColumn: columnToAnalyze
      });
      if (columnToAnalyze !== selectedColumn) {
        setSelectedColumn(columnToAnalyze);
      }
    } catch (error) {
      console.error('Re-analysis failed:', error);
      alert('Re-analysis failed: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // Handle back to dashboard
  const handleBackToDashboard = () => {
    setCurrentView('dashboard');
    setAnalysisData(null);
  };

  // Handle sync
  const handleSync = async (ticketId) => {
    setLoading(true);
    try {
      await syncSingleTicket(ticketId);
      await refreshAnalysis();
    } catch (error) {
      throw new Error('Sync failed: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // Handle create single
  const handleCreateSingle = async (taskId) => {
    setLoading(true);
    try {
      await createSingleTicket(taskId);
      await refreshAnalysis();
    } catch (error) {
      console.error('Create single failed:', error);
      throw new Error('Create failed: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // Handle create missing
  const handleCreateMissing = async () => {
  setLoading(true);
  try {
    // Pass the selectedColumn to the API call
    await createMissingTickets(selectedColumn);
    await refreshAnalysis();
  } catch (error) {
    throw new Error('Bulk create failed: ' + error.message);
  } finally {
    setLoading(false);
  }
};

  // Refresh analysis data
  const refreshAnalysis = async () => {
    try {
      const data = await analyzeTickets(selectedColumn);
      setAnalysisData({
        ...data,
        analyzedColumn: selectedColumn
      });
    } catch (error) {
      console.error('Failed to refresh analysis:', error);
    }
  };

  // Get navigation content based on current view
  const getNavigationContent = () => {
    if (currentView === 'settings') {
      return {
        left: (
          <div className="flex items-center">
            <Settings className="w-8 h-8 text-blue-600 mr-3" />
            <div className="text-xl font-semibold text-gray-900">Settings</div>
          </div>
        ),
        right: null
      };
    }

    if (currentView === 'sync-history') {
      return {
        left: (
          <div className="flex items-center">
            <History className="w-8 h-8 text-purple-600 mr-3" />
            <div className="text-xl font-semibold text-gray-900">Sync History</div>
          </div>
        ),
        right: null
      };
    }

    if (currentView === 'audit-logs') {
      return {
        left: (
          <div className="flex items-center">
            <FileText className="w-8 h-8 text-green-600 mr-3" />
            <div className="text-xl font-semibold text-gray-900">Audit Logs</div>
          </div>
        ),
        right: null
      };
    }

    if (currentView === 'reverse-sync') {
      return {
        left: (
          <div className="flex items-center">
            <RefreshCw className="w-8 h-8 text-blue-600 mr-3" />
            <div className="text-xl font-semibold text-gray-900">Reverse Sync</div>
          </div>
        ),
        right: null
      };
    }

    return {
      left: navLeft ?? (currentView === 'dashboard' ? (
        <div className="flex items-center">
          <img
            src="https://apyhub.com/logo.svg"
            alt="ApyHub"
            className="h-8 w-8 apyhub-logo"
          />
          <div className="ml-3 text-xl font-semibold text-gray-900">Asana-YouTrack Sync</div>
        </div>
      ) : null),
      right: navRight ?? (currentView === 'dashboard' ? (
        <div className="flex items-center space-x-6">
          <div className="flex items-center text-sm text-gray-500">
            <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
            <span>Connected</span>
          </div>
          <div className="flex items-center space-x-3">
            <div className="text-right">
              <div className="text-sm font-semibold text-gray-900">Welcome, {user?.username}</div>
              <div className="text-xs text-gray-500">Enhanced Dashboard</div>
            </div>
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setCurrentView('reverse-sync')}
                className="flex items-center justify-center h-10 w-10 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-lg shadow-sm text-white hover:shadow-md transition-shadow"
                title="Reverse Sync (YouTrack → Asana)"
              >
                <RefreshCw className="w-7 h-7" strokeWidth={2.5} />
              </button>
              <button
                onClick={() => setCurrentView('sync-history')}
                className="flex items-center justify-center h-10 w-10 bg-gradient-to-br from-purple-500 to-pink-600 rounded-lg shadow-sm text-white hover:shadow-md transition-shadow"
                title="Sync History"
              >
                <History className="w-7 h-7" strokeWidth={2.5} />
              </button>
              <button
                onClick={() => setCurrentView('audit-logs')}
                className="flex items-center justify-center h-10 w-10 bg-gradient-to-br from-green-500 to-teal-600 rounded-lg shadow-sm text-white hover:shadow-md transition-shadow"
                title="Audit Logs"
              >
                <FileText className="w-7 h-7" strokeWidth={2.5} />
              </button>
              <button
                onClick={() => setCurrentView('settings')}
                className="flex items-center justify-center h-10 w-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg shadow-sm text-white hover:shadow-md transition-shadow"
                title="Settings"
              >
                <Settings className="w-7 h-7" strokeWidth={2.5} />
              </button>
              <div className="flex items-center justify-center h-10 w-10 bg-gradient-to-br from-green-500 to-blue-600 rounded-lg shadow-sm text-white font-bold text-lg">
                {user?.username?.charAt(0).toUpperCase()}
              </div>
            </div>
          </div>
        </div>
      ) : null)
    };
  };

  const navContent = getNavigationContent();

  // ALWAYS show background for authenticated users (including settings)
  const showBackground = isAuthenticated;

  return (
    <div className="App" style={{ 
      position: 'relative', 
      minHeight: '100vh',
      display: 'flex',
      flexDirection: 'column'
    }}>
      {/* GLOBAL Background - shown on ALL pages for authenticated users */}
      {showBackground && (
        <>
          <div className="luxury-background-container">
            <LuxuryBackground 
              currentView={currentView}
              analysisData={analysisData}
              selectedColumn={selectedColumn}
              isLoading={loading}
            />
          </div>
          
          <div className="luxury-canvas-blur-separator" />
        </>
      )}
      
      {/* Main content layer with conditional styling */}
      <div 
        className={showBackground ? "luxury-canvas-content-layer" : ""}
        style={{ 
          flex: '1', 
          paddingBottom: showBackground ? '80px' : '0px',
          position: 'relative',
          zIndex: showBackground ? 1 : 'auto'
        }}
      >
        <NavBar
          title={currentView === 'dashboard' ? 'Dashboard' :
                 currentView === 'settings' ? 'Settings' :
                 currentView === 'sync-history' ? 'Sync History' :
                 currentView === 'audit-logs' ? 'Audit Logs' :
                 currentView === 'reverse-sync' ? 'Reverse Sync' : 'Analysis Results'}
          showBack={currentView !== 'dashboard'}
          onBack={currentView === 'settings' || currentView === 'sync-history' || currentView === 'audit-logs' || currentView === 'reverse-sync' ? () => setCurrentView('dashboard') : handleBackToDashboard}
          leftContent={navContent.left}
          rightContent={navContent.right}
        >
          {currentView === 'dashboard' ? (
            <Dashboard
              selectedColumn={selectedColumn}
              onColumnSelect={handleColumnSelect}
              onAnalyze={handleAnalyze}
              loading={loading}
            />
          ) : currentView === 'results' ? (
            <AnalysisResults
              analysisData={analysisData}
              selectedColumn={selectedColumn}
              onBack={handleBackToDashboard}
              onSync={handleSync}
              onCreateSingle={handleCreateSingle}
              onCreateMissing={handleCreateMissing}
              onReAnalyze={handleReAnalyze}
              loading={loading}
              setNavBarSlots={(left, right) => { setNavLeft(left); setNavRight(right); }}
            />
          ) : currentView === 'settings' ? (
            <UserSettings
              onBack={() => setCurrentView('dashboard')}
            />
          ) : currentView === 'sync-history' ? (
            <SyncHistory />
          ) : currentView === 'audit-logs' ? (
            <AuditLogs />
          ) : currentView === 'reverse-sync' ? (
            <ReverseSyncPage
              onBack={() => setCurrentView('dashboard')}
            />
          ) : null}
        </NavBar>
      </div>
      
      {/* Footer - shown for all authenticated views */}
      {showBackground && (
        <footer className="app-footer">
          <div className="credit-footer">
            Made with Frustration By Simran • Minor bugs included at no extra cost.
          </div>
        </footer>
      )}
    </div>
  );
}

// Root App with AuthProvider wrapper
function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

export default App;