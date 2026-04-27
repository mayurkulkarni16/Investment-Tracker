import api from './client';

export const exportCSV = (module?: string) => api.get('/export/csv', { params: { module }, responseType: 'blob' });
export const exportBackup = () => api.get('/backup', { responseType: 'blob' });
