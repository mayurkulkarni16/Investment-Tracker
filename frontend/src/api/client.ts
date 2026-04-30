import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.response.use(
  response => response,
  error => {
    const message = error.response?.data?.error || error.message || 'An error occurred';
    console.error(`API Error [${error.config?.method?.toUpperCase()} ${error.config?.url}]:`, message);
    return Promise.reject(error);
  }
);

export default api;
