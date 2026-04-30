import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

// Request interceptor: attach auth token and view-as-user header
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  const viewAsUser = localStorage.getItem('view_as_user');
  if (viewAsUser) {
    config.headers['X-View-As-User'] = viewAsUser;
  }
  return config;
});

api.interceptors.response.use(
  response => response,
  error => {
    const message = error.response?.data?.error || error.message || 'An error occurred';
    console.error(`API Error [${error.config?.method?.toUpperCase()} ${error.config?.url}]:`, message);
    // Redirect to login on 401
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('view_as_user');
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

export default api;
