// frontend/src/components/ReverseSync/ReverseTicketDetailView.js
import React, { useState, useEffect } from 'react';
import {
  ArrowLeft, CheckCircle, AlertCircle, Plus, RefreshCw,
  Calendar, User, Tag, FileText, Copy, Eye, EyeOff
} from 'lucide-react';
import { reverseIgnoreAction, getUserSettings } from '../../services/api';

const ReverseTicketDetailView = ({
  type,
  analysisData,
  selectedCreator,
  onBack,
  onCreateTickets,
  loading,
  onRefreshAnalysis
}) => {
  const [createAllLoading, setCreateAllLoading] = useState(false);
  const [copiedId, setCopiedId] = useState(null);
  const [ignoringTicket, setIgnoringTicket] = useState(null);
  const [creating, setCreating] = useState({});
  const [youtrackUrl, setYoutrackUrl] = useState('loop.youtrack.cloud'); // fallback default

  // Load YouTrack URL from user settings on mount
  useEffect(() => {
    const loadYouTrackUrl = async () => {
      try {
        const response = await getUserSettings();
        const settings = response.data || response;
        if (settings.youtrack_base_url) {
          // Remove protocol if present (we'll add it when opening the link)
          const cleanUrl = settings.youtrack_base_url.replace(/^https?:\/\//, '');
          setYoutrackUrl(cleanUrl);
        }
      } catch (error) {
        console.error('Failed to load YouTrack URL:', error);
      }
    };
    loadYouTrackUrl();
  }, []);

  const { matched = [], missing_asana = [], ignored = [] } = analysisData || {};
  const tickets = type === 'matched' ? matched : type === 'ignored' ? ignored : missing_asana;

  const handleCreateAll = async () => {
    setCreateAllLoading(true);
    const ticketIds = missing_asana.map(t => t.id);
    await onCreateTickets(ticketIds);
    setCreateAllLoading(false);
  };

  const handleCreateTicket = async (ticketId) => {
    setCreating(prev => ({ ...prev, [ticketId]: true }));
    await onCreateTickets([ticketId]);
    setCreating(prev => ({ ...prev, [ticketId]: false }));
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const handleCopyTicketTitle = (ticketName, ticketId) => {
    // Remove ticket ID prefix (e.g., "ARD-123 ")
    const titleWithoutId = ticketName.replace(/^[A-Z]+-\d+\s+/, '');
    navigator.clipboard.writeText(titleWithoutId).then(() => {
      setCopiedId(ticketId);
      setTimeout(() => setCopiedId(null), 2000);
    }).catch(err => {
      console.error('Failed to copy:', err);
    });
  };

  const handleOpenAsanaSearch = (ticketId, ticketName) => {
    // Search in Asana with ticket ID
    const searchQuery = encodeURIComponent(`${ticketId} ${ticketName}`);
    const asanaUrl = `https://app.asana.com/0/search?q=${searchQuery}`;
    window.open(asanaUrl, '_blank', 'noopener,noreferrer');
  };

  const handleOpenYouTrackLink = (ticketId) => {
    // Use the YouTrack URL from user settings
    const url = `https://${youtrackUrl}/issue/${ticketId}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  const handleIgnoreTicket = async (ticketId, ignoreType) => {
    setIgnoringTicket(ticketId);
    try {
      await reverseIgnoreAction(ticketId, 'add', ignoreType);
      if (onRefreshAnalysis) {
        await onRefreshAnalysis();
      }
    } catch (error) {
      console.error('Failed to ignore ticket:', error);
      alert('Failed to ignore ticket. Please try again.');
    } finally {
      setIgnoringTicket(null);
    }
  };

  const handleUnignoreTicket = async (ticketId, ignoreType) => {
    setIgnoringTicket(ticketId);
    try {
      await reverseIgnoreAction(ticketId, 'remove', ignoreType);
      if (onRefreshAnalysis) {
        await onRefreshAnalysis();
      }
    } catch (error) {
      console.error('Failed to unignore ticket:', error);
      alert('Failed to unignore ticket. Please try again.');
    } finally {
      setIgnoringTicket(null);
    }
  };

  const getTypeInfo = () => {
    const typeConfig = {
      matched: {
        title: 'Already in Asana',
        description: 'Tickets that exist in both YouTrack and Asana',
        icon: CheckCircle,
        color: 'green'
      },
      missing: {
        title: 'Missing in Asana',
        description: 'Tickets that need to be created in Asana',
        icon: AlertCircle,
        color: 'amber'
      },
      ignored: {
        title: 'Ignored Tickets',
        description: 'Tickets that are permanently ignored and won\'t be synced',
        icon: EyeOff,
        color: 'purple'
      }
    };
    return typeConfig[type] || typeConfig.missing;
  };

  const typeInfo = getTypeInfo();
  const TypeIcon = typeInfo.icon;

  return (
    <div className="min-h-screen">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="glass-panel border border-gray-200 rounded-lg p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center">
              <button
                onClick={onBack}
                className="glass-button mr-4"
              >
                <ArrowLeft className="w-5 h-5" />
              </button>
              <div>
                <div className="flex items-center">
                  <TypeIcon className={`w-6 h-6 mr-3 text-${typeInfo.color}-600`} />
                  <h1 className="text-2xl font-bold text-gray-900">{typeInfo.title}</h1>
                </div>
                <p className="text-gray-600 mt-1">{typeInfo.description}</p>
              </div>
            </div>

            {type === 'missing' && missing_asana.length > 0 && (
              <button
                onClick={handleCreateAll}
                disabled={createAllLoading}
                className="glass-button bg-gradient-to-r from-blue-500 to-purple-600 text-white"
              >
                {createAllLoading ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    Creating All...
                  </>
                ) : (
                  <>
                    <Plus className="w-4 h-4 mr-2" />
                    Create All ({missing_asana.length})
                  </>
                )}
              </button>
            )}
          </div>

          <div className="flex items-center text-sm text-gray-600">
            <FileText className="w-4 h-4 mr-2" />
            <span>Showing {tickets.length} ticket{tickets.length !== 1 ? 's' : ''}</span>
            {selectedCreator !== 'All' && (
              <>
                <span className="mx-2">•</span>
                <User className="w-4 h-4 mr-1" />
                <span>Creator: {selectedCreator}</span>
              </>
            )}
          </div>
        </div>

        {/* Tickets List */}
        <div className="space-y-4">
          {type === 'missing' && missing_asana.map((issue) => (
            <div
              key={issue.id}
              className="glass-panel border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-all"
            >
              {/* Header with Action Buttons */}
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-3 flex-1">
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-bold bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-sm">
                    {issue.id}
                  </span>
                  <h3 className="text-lg font-semibold text-gray-900 flex-1">{issue.summary}</h3>
                </div>

                {/* Action Buttons */}
                <div className="flex items-center gap-2 ml-4">
                  <button
                    onClick={() => handleCopyTicketTitle(issue.summary, issue.id)}
                    className={`glass-button p-2 ${copiedId === issue.id ? 'bg-green-100' : ''}`}
                    title="Copy ticket title"
                  >
                    <Copy className={`w-4 h-4 ${copiedId === issue.id ? 'text-green-600' : 'text-gray-600'}`} />
                  </button>
                  <button
                    onClick={() => handleOpenAsanaSearch(issue.id, issue.summary)}
                    className="glass-button p-2"
                    title="Search in Asana"
                  >
                    <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <circle cx="16" cy="9" r="5" fill="#F06A6A"/>
                      <circle cx="9" cy="20" r="5" fill="#F06A6A"/>
                      <circle cx="23" cy="20" r="5" fill="#F06A6A"/>
                    </svg>
                  </button>
                  <button
                    onClick={() => handleOpenYouTrackLink(issue.id)}
                    className="glass-button p-2"
                    title="Open in YouTrack"
                  >
                    <svg className="h-5 w-auto" viewBox="0 0 43.788 42.787" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <g clipPath="url(#clip0)">
                        <path fill="url(#gradient0)" d="M0.816462 32.0278C0.755912 31.9832 0.735994 31.9016 0.768806 31.834L5.54107 22.0055L0.0363835 15.4316C-0.0190853 15.3656 -0.0101009 15.2668 0.0563054 15.2118L16.1117 1.83243C18.9684 -0.548815 23.1044 -0.616002 26.0391 1.66955C28.9727 3.95509 29.9102 7.97463 28.2872 11.3172L26.5376 14.9215C27.22 14.6926 27.8954 14.4844 28.5633 14.2973L36.4844 12.0219C36.5707 11.9969 36.6606 12.05 36.6801 12.1379L39.9961 26.8785C40.0164 26.9696 39.9548 27.0586 39.8618 27.0703C38.8106 27.2028 33.0758 28.0266 25.936 31.0262C17.8466 34.4238 12.5082 39.2528 11.752 39.959C11.6961 40.0113 11.6141 40.0133 11.5528 39.968L0.816462 32.0278Z"/>
                        <path fill="black" d="M32.5 7.5H7.5V32.5H32.5V7.5Z"/>
                        <path fill="white" d="M13.4164 16.4828L9.98633 10.6211H11.9559L14.0797 14.366L14.3278 14.902L14.5758 14.3594L16.6457 10.6211H18.5816L15.2051 16.4695V20H13.4164V16.4828Z"/>
                        <path fill="white" d="M20.625 27.5H10.625V29.375H20.625V27.5Z"/>
                        <path fill="white" d="M26.4531 10.6211H18.9301L18.9297 12.2692H21.7836V20H23.6125V12.2692H26.4531V10.6211Z"/>
                      </g>
                      <defs>
                        <linearGradient id="gradient0" x1="-0.0640069" y1="20" x2="40.0332" y2="20" gradientUnits="userSpaceOnUse">
                          <stop stopColor="#FB43FF"/>
                          <stop offset="0.97" stopColor="#FB406D"/>
                        </linearGradient>
                        <clipPath id="clip0">
                          <rect fill="white" width="163.125" height="40"/>
                        </clipPath>
                      </defs>
                    </svg>
                  </button>

                  {/* Ignore Button */}
                  <button
                    onClick={() => handleIgnoreTicket(issue.id, 'forever')}
                    disabled={ignoringTicket === issue.id}
                    className="glass-button bg-red-50 text-red-700 hover:bg-red-100"
                    title="Ignore Permanently"
                  >
                    <EyeOff className="w-4 h-4 mr-1" />
                    {ignoringTicket === issue.id ? 'Ignoring...' : 'Ignore'}
                  </button>

                  {/* Create Button */}
                  <button
                    onClick={() => handleCreateTicket(issue.id)}
                    disabled={creating[issue.id]}
                    className="glass-button bg-gradient-to-r from-blue-500 to-purple-600 text-white"
                  >
                    {creating[issue.id] ? (
                      <>
                        <RefreshCw className="w-4 h-4 mr-1 animate-spin" />
                        Creating...
                      </>
                    ) : (
                      <>
                        <Plus className="w-4 h-4 mr-1" />
                        Create
                      </>
                    )}
                  </button>
                </div>
              </div>

              {/* Metadata */}
              <div className="flex flex-wrap gap-4 mb-3">
                {issue.state && (
                  <div className="flex items-center text-sm">
                    <FileText className="w-4 h-4 mr-1.5 text-blue-600" />
                    <span className="text-gray-700">State:</span>
                    <span className="ml-1 px-2 py-0.5 rounded bg-blue-100 text-blue-800 font-medium">
                      {issue.state}
                    </span>
                  </div>
                )}
                {issue.subsystem && (
                  <div className="flex items-center text-sm">
                    <Tag className="w-4 h-4 mr-1.5 text-purple-600" />
                    <span className="text-gray-700">Subsystem:</span>
                    <span className="ml-1 px-2 py-0.5 rounded bg-purple-100 text-purple-800 font-medium">
                      {issue.subsystem}
                    </span>
                  </div>
                )}
                {issue.created_by && (
                  <div className="flex items-center text-sm">
                    <User className="w-4 h-4 mr-1.5 text-gray-600" />
                    <span className="text-gray-700">Creator:</span>
                    <span className="ml-1 font-medium text-gray-900">{issue.created_by}</span>
                  </div>
                )}
                {issue.created && (
                  <div className="flex items-center text-sm">
                    <Calendar className="w-4 h-4 mr-1.5 text-gray-600" />
                    <span className="text-gray-700">Created:</span>
                    <span className="ml-1 text-gray-900">{formatDate(issue.created)}</span>
                  </div>
                )}
              </div>

              {/* Description */}
              {issue.description && (
                <div className="glass-panel border border-gray-200 rounded-lg p-3 bg-gray-50">
                  <p className="text-sm text-gray-700 line-clamp-3">
                    {issue.description}
                  </p>
                </div>
              )}
            </div>
          ))}

          {type === 'matched' && matched.map((item) => (
            <div
              key={item.youtrack_issue.id}
              className="glass-panel border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-all"
            >
              {/* Header with Action Buttons */}
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-3 flex-1">
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-bold bg-gradient-to-r from-green-500 to-emerald-600 text-white shadow-sm">
                    {item.youtrack_issue.id}
                  </span>
                  <h3 className="text-lg font-semibold text-gray-900 flex-1">
                    {item.youtrack_issue.summary}
                  </h3>
                  <CheckCircle className="w-5 h-5 text-green-600" />
                </div>

                {/* Action Buttons */}
                <div className="flex items-center gap-2 ml-4">
                  <button
                    onClick={() => handleCopyTicketTitle(item.youtrack_issue.summary, item.youtrack_issue.id)}
                    className={`glass-button p-2 ${copiedId === item.youtrack_issue.id ? 'bg-green-100' : ''}`}
                    title="Copy ticket title"
                  >
                    <Copy className={`w-4 h-4 ${copiedId === item.youtrack_issue.id ? 'text-green-600' : 'text-gray-600'}`} />
                  </button>
                  <button
                    onClick={() => handleOpenAsanaSearch(item.youtrack_issue.id, item.youtrack_issue.summary)}
                    className="glass-button p-2"
                    title="Search in Asana"
                  >
                    <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <circle cx="16" cy="9" r="5" fill="#F06A6A"/>
                      <circle cx="9" cy="20" r="5" fill="#F06A6A"/>
                      <circle cx="23" cy="20" r="5" fill="#F06A6A"/>
                    </svg>
                  </button>
                  <button
                    onClick={() => handleOpenYouTrackLink(item.youtrack_issue.id)}
                    className="glass-button p-2"
                    title="Open in YouTrack"
                  >
                    <svg className="h-5 w-auto" viewBox="0 0 43.788 42.787" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <g clipPath="url(#clip0)">
                        <path fill="url(#gradient0)" d="M0.816462 32.0278C0.755912 31.9832 0.735994 31.9016 0.768806 31.834L5.54107 22.0055L0.0363835 15.4316C-0.0190853 15.3656 -0.0101009 15.2668 0.0563054 15.2118L16.1117 1.83243C18.9684 -0.548815 23.1044 -0.616002 26.0391 1.66955C28.9727 3.95509 29.9102 7.97463 28.2872 11.3172L26.5376 14.9215C27.22 14.6926 27.8954 14.4844 28.5633 14.2973L36.4844 12.0219C36.5707 11.9969 36.6606 12.05 36.6801 12.1379L39.9961 26.8785C40.0164 26.9696 39.9548 27.0586 39.8618 27.0703C38.8106 27.2028 33.0758 28.0266 25.936 31.0262C17.8466 34.4238 12.5082 39.2528 11.752 39.959C11.6961 40.0113 11.6141 40.0133 11.5528 39.968L0.816462 32.0278Z"/>
                        <path fill="black" d="M32.5 7.5H7.5V32.5H32.5V7.5Z"/>
                        <path fill="white" d="M13.4164 16.4828L9.98633 10.6211H11.9559L14.0797 14.366L14.3278 14.902L14.5758 14.3594L16.6457 10.6211H18.5816L15.2051 16.4695V20H13.4164V16.4828Z"/>
                        <path fill="white" d="M20.625 27.5H10.625V29.375H20.625V27.5Z"/>
                        <path fill="white" d="M26.4531 10.6211H18.9301L18.9297 12.2692H21.7836V20H23.6125V12.2692H26.4531V10.6211Z"/>
                      </g>
                      <defs>
                        <linearGradient id="gradient0" x1="-0.0640069" y1="20" x2="40.0332" y2="20" gradientUnits="userSpaceOnUse">
                          <stop stopColor="#FB43FF"/>
                          <stop offset="0.97" stopColor="#FB406D"/>
                        </linearGradient>
                        <clipPath id="clip0">
                          <rect fill="white" width="163.125" height="40"/>
                        </clipPath>
                      </defs>
                    </svg>
                  </button>
                </div>
              </div>

              {/* Asana Task Info */}
              <div className="flex items-center space-x-4 text-sm mb-3">
                <div className="flex items-center">
                  <svg className="w-4 h-4 mr-1.5" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <circle cx="16" cy="9" r="4" fill="#F06A6A"/>
                    <circle cx="9" cy="20" r="4" fill="#F06A6A"/>
                    <circle cx="23" cy="20" r="4" fill="#F06A6A"/>
                  </svg>
                  <span className="text-gray-700">Asana Task ID:</span>
                  <span className="ml-1 font-mono text-gray-900">{item.asana_task.gid}</span>
                </div>
              </div>

              {/* Metadata */}
              <div className="flex flex-wrap gap-4">
                {item.youtrack_issue.state && (
                  <div className="flex items-center text-sm">
                    <FileText className="w-4 h-4 mr-1.5 text-blue-600" />
                    <span className="text-gray-700">State:</span>
                    <span className="ml-1 px-2 py-0.5 rounded bg-blue-100 text-blue-800 font-medium">
                      {item.youtrack_issue.state}
                    </span>
                  </div>
                )}
                {item.youtrack_issue.created_by && (
                  <div className="flex items-center text-sm">
                    <User className="w-4 h-4 mr-1.5 text-gray-600" />
                    <span className="text-gray-700">Creator:</span>
                    <span className="ml-1 font-medium text-gray-900">{item.youtrack_issue.created_by}</span>
                  </div>
                )}
              </div>
            </div>
          ))}

          {type === 'ignored' && ignored.map((issue) => (
            <div
              key={issue.id}
              className="glass-panel border border-purple-200 rounded-lg p-6 hover:shadow-lg transition-all bg-purple-50"
            >
              {/* Header with Action Buttons */}
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-3 flex-1">
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-bold bg-gradient-to-r from-purple-500 to-pink-600 text-white shadow-sm">
                    {issue.id}
                  </span>
                  <h3 className="text-lg font-semibold text-gray-900 flex-1">{issue.summary}</h3>
                </div>

                {/* Action Buttons */}
                <div className="flex items-center gap-2 ml-4">
                  <button
                    onClick={() => handleCopyTicketTitle(issue.summary, issue.id)}
                    className={`glass-button p-2 ${copiedId === issue.id ? 'bg-green-100' : ''}`}
                    title="Copy ticket title"
                  >
                    <Copy className={`w-4 h-4 ${copiedId === issue.id ? 'text-green-600' : 'text-gray-600'}`} />
                  </button>
                  <button
                    onClick={() => handleOpenAsanaSearch(issue.id, issue.summary)}
                    className="glass-button p-2"
                    title="Search in Asana"
                  >
                    <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <circle cx="16" cy="9" r="5" fill="#F06A6A"/>
                      <circle cx="9" cy="20" r="5" fill="#F06A6A"/>
                      <circle cx="23" cy="20" r="5" fill="#F06A6A"/>
                    </svg>
                  </button>
                  <button
                    onClick={() => handleOpenYouTrackLink(issue.id)}
                    className="glass-button p-2"
                    title="Open in YouTrack"
                  >
                    <svg className="h-5 w-auto" viewBox="0 0 43.788 42.787" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <g clipPath="url(#clip0)">
                        <path fill="url(#gradient0)" d="M0.816462 32.0278C0.755912 31.9832 0.735994 31.9016 0.768806 31.834L5.54107 22.0055L0.0363835 15.4316C-0.0190853 15.3656 -0.0101009 15.2668 0.0563054 15.2118L16.1117 1.83243C18.9684 -0.548815 23.1044 -0.616002 26.0391 1.66955C28.9727 3.95509 29.9102 7.97463 28.2872 11.3172L26.5376 14.9215C27.22 14.6926 27.8954 14.4844 28.5633 14.2973L36.4844 12.0219C36.5707 11.9969 36.6606 12.05 36.6801 12.1379L39.9961 26.8785C40.0164 26.9696 39.9548 27.0586 39.8618 27.0703C38.8106 27.2028 33.0758 28.0266 25.936 31.0262C17.8466 34.4238 12.5082 39.2528 11.752 39.959C11.6961 40.0113 11.6141 40.0133 11.5528 39.968L0.816462 32.0278Z"/>
                        <path fill="black" d="M32.5 7.5H7.5V32.5H32.5V7.5Z"/>
                        <path fill="white" d="M13.4164 16.4828L9.98633 10.6211H11.9559L14.0797 14.366L14.3278 14.902L14.5758 14.3594L16.6457 10.6211H18.5816L15.2051 16.4695V20H13.4164V16.4828Z"/>
                        <path fill="white" d="M20.625 27.5H10.625V29.375H20.625V27.5Z"/>
                        <path fill="white" d="M26.4531 10.6211H18.9301L18.9297 12.2692H21.7836V20H23.6125V12.2692H26.4531V10.6211Z"/>
                      </g>
                      <defs>
                        <linearGradient id="gradient0" x1="-0.0640069" y1="20" x2="40.0332" y2="20" gradientUnits="userSpaceOnUse">
                          <stop stopColor="#FB43FF"/>
                          <stop offset="0.97" stopColor="#FB406D"/>
                        </linearGradient>
                        <clipPath id="clip0">
                          <rect fill="white" width="163.125" height="40"/>
                        </clipPath>
                      </defs>
                    </svg>
                  </button>

                  {/* Unignore Button */}
                  <button
                    onClick={() => handleUnignoreTicket(issue.id, 'forever')}
                    disabled={ignoringTicket === issue.id}
                    className="glass-button bg-green-50 text-green-700 hover:bg-green-100"
                    title="Unignore Ticket"
                  >
                    <Eye className="w-4 h-4 mr-1" />
                    {ignoringTicket === issue.id ? 'Unignoring...' : 'Unignore'}
                  </button>
                </div>
              </div>

              {/* Metadata */}
              <div className="flex flex-wrap gap-4 mb-3">
                {issue.state && (
                  <div className="flex items-center text-sm">
                    <FileText className="w-4 h-4 mr-1.5 text-blue-600" />
                    <span className="text-gray-700">State:</span>
                    <span className="ml-1 px-2 py-0.5 rounded bg-blue-100 text-blue-800 font-medium">
                      {issue.state}
                    </span>
                  </div>
                )}
                {issue.subsystem && (
                  <div className="flex items-center text-sm">
                    <Tag className="w-4 h-4 mr-1.5 text-purple-600" />
                    <span className="text-gray-700">Subsystem:</span>
                    <span className="ml-1 px-2 py-0.5 rounded bg-purple-100 text-purple-800 font-medium">
                      {issue.subsystem}
                    </span>
                  </div>
                )}
                {issue.created_by && (
                  <div className="flex items-center text-sm">
                    <User className="w-4 h-4 mr-1.5 text-gray-600" />
                    <span className="text-gray-700">Creator:</span>
                    <span className="ml-1 font-medium text-gray-900">{issue.created_by}</span>
                  </div>
                )}
                {issue.created && (
                  <div className="flex items-center text-sm">
                    <Calendar className="w-4 h-4 mr-1.5 text-gray-600" />
                    <span className="text-gray-700">Created:</span>
                    <span className="ml-1 text-gray-900">{formatDate(issue.created)}</span>
                  </div>
                )}
              </div>

              {/* Description */}
              {issue.description && (
                <div className="glass-panel border border-gray-200 rounded-lg p-3 bg-purple-100">
                  <p className="text-sm text-gray-700 line-clamp-3">
                    {issue.description}
                  </p>
                </div>
              )}

              {/* Ignored Badge */}
              <div className="mt-3 flex items-center text-sm text-purple-700 bg-purple-100 rounded px-3 py-2">
                <EyeOff className="w-4 h-4 mr-2" />
                <span className="font-medium">This ticket is permanently ignored</span>
              </div>
            </div>
          ))}
        </div>

        {/* Empty State */}
        {tickets.length === 0 && (
          <div className="glass-panel border border-gray-200 rounded-lg p-12 text-center">
            <TypeIcon className={`w-16 h-16 mx-auto mb-4 text-${typeInfo.color}-600 opacity-50`} />
            <h3 className="text-xl font-semibold text-gray-900 mb-2">
              No Tickets Found
            </h3>
            <p className="text-gray-600">
              {type === 'missing'
                ? 'All YouTrack tickets are already in Asana!'
                : type === 'ignored'
                ? 'No ignored tickets found.'
                : 'No matched tickets found.'}
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ReverseTicketDetailView;
