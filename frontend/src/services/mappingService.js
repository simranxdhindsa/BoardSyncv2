// ============================================
// FILE: frontend/src/services/mappingService.js
// ============================================
// API service for ticket mapping endpoints

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

/**
 * Get authentication token from localStorage
 */
const getAuthToken = () => {
  return localStorage.getItem('token') || localStorage.getItem('authToken');
};

/**
 * Ticket Mapping API Service
 */
const mappingService = {
  /**
   * Create a new manual mapping
   */
  createMapping: async (asanaUrl, youtrackUrl) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/mappings`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`
        },
        body: JSON.stringify({
          asana_url: asanaUrl,
          youtrack_url: youtrackUrl
        })
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Failed to create mapping');
      }

      return data;
    } catch (error) {
      console.error('Error creating mapping:', error);
      throw error;
    }
  },

  /**
   * Get all mappings
   */
  getAllMappings: async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/mappings`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${getAuthToken()}`
        }
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Failed to fetch mappings');
      }

      return data;
    } catch (error) {
      console.error('Error fetching mappings:', error);
      throw error;
    }
  },

  /**
   * Delete a mapping
   */
  deleteMapping: async (id) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/mappings/${id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${getAuthToken()}`
        }
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Failed to delete mapping');
      }

      return data;
    } catch (error) {
      console.error('Error deleting mapping:', error);
      throw error;
    }
  },

  /**
   * Find by Asana ID
   */
  findByAsanaId: async (taskId) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/mappings/asana/${taskId}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${getAuthToken()}`
        }
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Mapping not found');
      }

      return data;
    } catch (error) {
      console.error('Error finding mapping:', error);
      throw error;
    }
  },

  /**
   * Find by YouTrack ID
   */
  findByYouTrackId: async (issueId) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/mappings/youtrack/${issueId}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${getAuthToken()}`
        }
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Mapping not found');
      }

      return data;
    } catch (error) {
      console.error('Error finding mapping:', error);
      throw error;
    }
  }
};

export default mappingService;