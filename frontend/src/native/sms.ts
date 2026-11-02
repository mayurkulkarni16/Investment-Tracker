import { Capacitor, registerPlugin } from '@capacitor/core';
import type { SmsMessage } from '../utils/smsParser.mjs';

interface BankSmsPlugin {
  scan(options: { days: number; sinceMs?: number }): Promise<{ messages: SmsMessage[]; truncated: boolean }>;
  openSettings(): Promise<void>;
}
export const hasAndroidSms = Capacitor.getPlatform() === 'android';
export const BankSms = registerPlugin<BankSmsPlugin>('BankSms');
