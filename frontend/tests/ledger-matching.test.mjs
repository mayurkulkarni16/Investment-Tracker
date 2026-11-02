import test from 'node:test';
import assert from 'node:assert/strict';
import {
  findDuplicates, indexDuplicates, memberForAccount, normalizeBank, suggestAttribution,
} from '../src/utils/ledgerMatching.mjs';
import { parseSms } from '../src/utils/smsParser.mjs';

const members = [{ id: 'alice', name: 'Alice' }, { id: 'bob', name: 'Bob' }];
const account = (overrides = {}) => ({
  type: 'bank_account', id: 'a1', name: 'Savings', bank: 'HDFC Bank',
  last_four: '1234', status: 'active', member_ids: ['alice'], ...overrides,
});
const draft = (overrides = {}) => ({
  source_key: '', source: 'sms', status: 'pending', kind: 'expense', amount_paise: 12500,
  date: '2026-10-21', bank: 'HDFC', account: '1234', description: 'Purchase',
  profile_id: '', linked_account_type: '', linked_account_id: '',
  sms_received_at: '2026-10-21T10:00:00Z', ...overrides,
});
const entry = (id, overrides = {}) => ({
  ...draft(), id, source_key: id.repeat(64).slice(0, 64),
  created_at: '2026-10-21T10:00:00Z', updated_at: '2026-10-21T10:00:00Z', ...overrides,
});

test('bank aliases match without conflating different banks', () => {
  for (const [left, right] of [
    ['HDFC', 'HDFC Bank Ltd'], ['ICICI', 'ICICI BANK'], ['SBI', 'State Bank of India'],
    ['Axis', 'Axis Bank'], ['YES BANK', 'YESBNK'], ['OneCard', 'ONECRD'],
  ]) assert.equal(normalizeBank(left), normalizeBank(right));
  assert.notEqual(normalizeBank('HDFC'), normalizeBank('ICICI'));
});

test('unique account matches use its sole owner even on another member device', () => {
  const result = suggestAttribution(draft(), { members, accounts: [account()] }, 'bob');
  assert.equal(result.account.id, 'a1');
  assert.equal(result.profileId, 'alice');
  assert.match(result.message, /check/i);
});

test('joint and unassigned ownership use the device default conservatively', () => {
  const joint = account({ member_ids: ['alice', 'bob'] });
  assert.equal(memberForAccount(joint, 'bob'), 'bob');
  assert.equal(memberForAccount(joint, ''), '');
  assert.equal(memberForAccount(joint, 'other'), '');
  assert.equal(memberForAccount(account({ member_ids: [] }), 'alice'), 'alice');
  const result = suggestAttribution(draft(), { members, accounts: [joint] }, 'bob');
  assert.equal(result.profileId, 'bob');
  assert.match(result.message, /joint account/i);
});

test('ambiguous and mismatched accounts are never linked automatically', () => {
  const context = { members, accounts: [account(), account({ id: 'a2' })] };
  const ambiguous = suggestAttribution(draft(), context, 'bob');
  assert.equal(ambiguous.account, null);
  assert.equal(ambiguous.profileId, 'bob');
  assert.match(ambiguous.message, /several/i);
  const differentBank = suggestAttribution(draft(), { members, accounts: [account({ bank: 'ICICI' })] }, 'alice');
  assert.equal(differentBank.account, null);
  assert.match(differentBank.message, /no saved/i);
  const unknownBank = suggestAttribution(draft(), { members, accounts: [account(), account({ id: 'a2', bank: '' })] }, '');
  assert.equal(unknownBank.account, null);
  const invalidDefault = suggestAttribution(draft({ account: '' }), context, 'deleted');
  assert.equal(invalidDefault.profileId, '');
});

test('matching can identify a credit card without mixing up other issuers', () => {
  const card = account({ type: 'credit_card', id: 'c1', bank: 'ICICI Bank', member_ids: ['bob'] });
  const result = suggestAttribution(draft({ bank: 'ICICI' }), { members, accounts: [account(), card] }, 'alice');
  assert.equal(result.account.id, 'c1');
  assert.equal(result.profileId, 'bob');
});

test('SMS parsing retains receipt time for cross-device comparison', () => {
  const parsed = parseSms({ sender: 'VM-HDFCBK', body: 'INR 125.00 debited from A/c XX1234.', receivedAt: '2026-10-21T15:30:00+05:30' });
  assert.equal(parsed.draft.sms_received_at, '2026-10-21T10:00:00.000Z');
  assert.equal(parsed.draft.profile_id, '');
});

test('duplicate SMS detection uses amount, account, type, day and a five-minute window', () => {
  const original = entry('a');
  const near = entry('b', { sms_received_at: '2026-10-21T10:05:00Z' });
  const far = entry('c', { sms_received_at: '2026-10-21T10:05:00.001Z' });
  assert.equal(findDuplicates(original, [near])[0].reason, 'within_minutes');
  assert.deepEqual(findDuplicates(original, [far]), []);
  for (const overrides of [
    { date: '2026-10-22' }, { account: '4567' }, { bank: 'ICICI' },
    { amount_paise: 12501 }, { kind: 'income' }, { status: 'rejected' },
  ]) assert.deepEqual(findDuplicates(original, [entry('d', overrides)]), []);
});

test('same last four on two explicitly different saved accounts is not a match', () => {
  const first = entry('a', { linked_account_type: 'bank_account', linked_account_id: 'a1' });
  const otherBank = entry('b', { linked_account_type: 'bank_account', linked_account_id: 'a2' });
  const otherCard = entry('c', { linked_account_type: 'credit_card', linked_account_id: 'a1' });
  assert.deepEqual(findDuplicates(first, [otherBank, otherCard]), []);
  assert.equal(findDuplicates(first, [entry('d')])[0].reason, 'within_minutes');
});

test('legacy/manual entries without receipt time get a date-only warning, never automatic rejection', () => {
  const a = entry('a');
  const b = entry('b', { source: 'manual', sms_received_at: '', status: 'confirmed' });
  assert.equal(findDuplicates(a, [b])[0].reason, 'date_only');
  const index = indexDuplicates([a, b, entry('c', { status: 'rejected' })]);
  assert.equal(index.a.length, 1);
  assert.equal(index.b[0].entry.id, 'a');
  assert.equal(index.c, undefined);
  assert.equal(a.status, 'pending');
  assert.equal(b.status, 'confirmed');
});

test('identical fingerprint is surfaced even if the existing import was rejected', () => {
  const same = entry('a', { status: 'rejected' });
  const newDraft = draft({ source_key: same.source_key });
  assert.equal(findDuplicates(newDraft, [same])[0].reason, 'exact');
  assert.deepEqual(findDuplicates(same, [same]), []);
});
