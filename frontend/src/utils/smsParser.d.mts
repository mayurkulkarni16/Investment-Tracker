import type { LedgerDraft } from '../api/ledger';

export interface SmsMessage {
  sender: string;
  body: string;
  receivedAt: number | string;
}
export function toPaise(value: string): number;
export function localDate(value: number | string): string;
export function parseSms(message: SmsMessage): { draft?: Omit<LedgerDraft, 'source_key'>; reason?: string; warning?: string };
export function smsFingerprint(message: SmsMessage): Promise<string>;
export function manualSourceKey(): Promise<string>;
