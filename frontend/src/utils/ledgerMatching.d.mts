import type { LedgerAccount, LedgerContext, LedgerDraft, LedgerEntry } from '../api/ledger';

export type DuplicateReason = 'exact' | 'within_minutes' | 'date_only';
export interface DuplicateMatch {
  entry: LedgerEntry;
  reason: DuplicateReason;
}
export interface AttributionSuggestion {
  account: LedgerAccount | null;
  profileId: string;
  message: string;
}

export function normalizeBank(value: string): string;
export function accountKey(account: LedgerAccount): string;
export function memberForAccount(account: LedgerAccount, deviceMemberId: string): string;
export function suggestAttribution(draft: LedgerDraft, context: LedgerContext, deviceMemberId: string): AttributionSuggestion;
export function findDuplicates(entry: LedgerDraft & { id?: string }, entries: LedgerEntry[]): DuplicateMatch[];
export function indexDuplicates(entries: LedgerEntry[]): Record<string, DuplicateMatch[]>;
