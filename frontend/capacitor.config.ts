import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.investmenttracker.app',
  appName: 'FinBoard',
  webDir: 'dist',
  android: { allowMixedContent: false },
  server: { androidScheme: 'https' },
};

export default config;
