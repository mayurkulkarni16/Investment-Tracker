import { Capacitor, registerPlugin } from '@capacitor/core';
import type { SmsMessage } from '../utils/smsParser.mjs';

export interface SmsScanCursor {
  until: number;
  date: number;
  id: number;
}

interface BankSmsPlugin {
  scan(options: { days: number; sinceMs?: number; cursor?: SmsScanCursor }): Promise<{
    messages: SmsMessage[];
    scanned: number;
    nextCursor?: SmsScanCursor;
  }>;
  openSettings(): Promise<void>;
}
export const hasAndroidSms = Capacitor.getPlatform() === 'android';
export const BankSms = registerPlugin<BankSmsPlugin>('BankSms');
