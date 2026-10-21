/**
 * EVENTRA API Client
 * Centralized request helper with credentials, JWT header fallback, and response normalization
 */

const API_BASE = window.location.origin.startsWith('file:')
  ? 'http://localhost:8080/api/v1'
  : `${window.location.origin}/api/v1`;

class APIClient {
  constructor() {
    this.tokenKey = 'eventra_jwt_token';
    this.userKey = 'eventra_user_data';
  }

  getToken() {
    return localStorage.getItem(this.tokenKey) || localStorage.getItem('evenrtra_jwt_token');
  }

  setToken(token) {
    if (token) {
      localStorage.setItem(this.tokenKey, token);
    } else {
      localStorage.removeItem(this.tokenKey);
      localStorage.removeItem('evenrtra_jwt_token');
    }
  }

  getUser() {
    try {
      const data = localStorage.getItem(this.userKey) || localStorage.getItem('evenrtra_user_data');
      return data ? JSON.parse(data) : null;
    } catch {
      return null;
    }
  }

  setUser(user) {
    if (user) {
      localStorage.setItem(this.userKey, JSON.stringify(user));
    } else {
      localStorage.removeItem(this.userKey);
      localStorage.removeItem('evenrtra_user_data');
    }
  }

  clearAuth() {
    localStorage.removeItem(this.tokenKey);
    localStorage.removeItem('evenrtra_jwt_token');
    localStorage.removeItem(this.userKey);
    localStorage.removeItem('evenrtra_user_data');
  }

  async request(endpoint, options = {}) {
    const url = endpoint.startsWith('http') ? endpoint : `${API_BASE}${endpoint}`;
    
    const headers = {
      'Content-Type': 'application/json',
      ...(options.headers || {})
    };

    const token = this.getToken();
    if (token && !headers['Authorization']) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const config = {
      ...options,
      headers,
      credentials: 'include' // Sends session cookies automatically
    };

    try {
      const res = await fetch(url, config);
      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        let errorMsg = data.error || `Request failed (${res.status})`;
        if (Array.isArray(data.data) && data.data.length > 0) {
          const details = data.data.map(d => `${d.field ? d.field.replace('_', ' ') + ': ' : ''}${d.message}`).join(', ');
          errorMsg = `${details}`;
        } else if (data.data && typeof data.data === 'string') {
          errorMsg = data.data;
        }
        
        // Handle unauthorized
        if (res.status === 401 && !endpoint.includes('/auth/login') && !endpoint.includes('/auth/firebase')) {
          this.clearAuth();
          // Dispatch auth expired event
          window.dispatchEvent(new CustomEvent('eventra:auth-expired'));
          window.dispatchEvent(new CustomEvent('evenrtra:auth-expired'));
        }

        const error = new Error(errorMsg);
        error.status = res.status;
        error.data = data;
        throw error;
      }

      return data;
    } catch (err) {
      console.error(`API Error on [${options.method || 'GET'}] ${endpoint}:`, err);
      throw err;
    }
  }

  get(endpoint, params = {}) {
    const query = new URLSearchParams(params).toString();
    const url = query ? `${endpoint}?${query}` : endpoint;
    return this.request(url, { method: 'GET' });
  }

  post(endpoint, body = {}) {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  }

  put(endpoint, body = {}) {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(body)
    });
  }

  patch(endpoint, body = {}) {
    return this.request(endpoint, {
      method: 'PATCH',
      body: JSON.stringify(body)
    });
  }

  delete(endpoint) {
    return this.request(endpoint, { method: 'DELETE' });
  }
}

// Global API instance
window.api = new APIClient();

// Toast notification helper
window.showToast = function(message, type = 'info', duration = 4000) {
  let container = document.getElementById('toast-container');
  if (!container) {
    container = document.createElement('div');
    container.id = 'toast-container';
    document.body.appendChild(container);
  }

  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  
  let icon = 'ℹ️';
  if (type === 'success') icon = '✅';
  if (type === 'error') icon = '⚠️';

  toast.innerHTML = `
    <span style="font-size:1.2rem;">${icon}</span>
    <div style="flex:1; font-size:0.9rem; color:var(--text-main); font-weight:500;">${message}</div>
  `;

  container.appendChild(toast);

  setTimeout(() => {
    toast.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(10px)';
    setTimeout(() => toast.remove(), 300);
  }, duration);
};
