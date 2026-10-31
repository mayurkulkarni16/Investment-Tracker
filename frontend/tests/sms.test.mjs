import test from 'node:test';
import assert from 'node:assert/strict';
import { toPaise, parseSms, smsFingerprint } from '../src/utils/smsParser.mjs';

const message = (body, sender = 'VM-HDFCBK') => ({ sender, body, receivedAt: '2026-10-21T10:00:00Z' });

test('exact INR minor units and Indian digit grouping', () => {
  assert.equal(toPaise('1,23,456.78'), 12345678);
  assert.equal(toPaise('1234.5'), 123450);
  for (const value of ['-10', '0', '1.001', '1,2', 'NaN', '10000000001', '1e3']) assert.throws(() => toPaise(value));
});

test('supported issuer debit and credit templates create pending drafts', () => {
  for (const sender of ['VM-HDFCBK', 'AX-ICICIB', 'AD-AXISBK', 'VM-SBIINB', 'VM-ONECRD', 'AX-YESBNK']) {
    const result = parseSms(message('INR 1,200.50 debited from A/c XX1234. Avl Bal INR 20,000.', sender));
    assert.equal(result.draft.amount_paise, 120050);
    assert.equal(result.draft.account, '1234');
    assert.equal(result.draft.kind, 'expense');
    assert.equal(result.draft.status, 'pending');
    assert.ok(result.warning);
  }
  assert.equal(parseSms(message('Account credited INR 50,000.00. Balance INR 60,000')).draft.kind, 'income');
});

test('card bill uses total due, not minimum or available limit', () => {
  const result = parseSms(message('Card XX1234 statement: Total amount due Rs. 10,500.00. Min due Rs. 500. Due on 25/10/2026.'));
  assert.equal(result.draft.kind, 'bill');
  assert.equal(result.draft.amount_paise, 1050000);
  assert.equal(result.draft.due_date, '2026-10-25');
  assert.equal(parseSms(message('Card bill: Minimum payment due Rs. 500.')).draft, undefined);
  assert.equal(parseSms(message('Card bill: Total due Rs. 1000. Minimum payment due Rs. 500.')).draft.amount_paise, 100000);
});

test('repayments are transfers, and ambiguous/OTP/promo messages do not import', () => {
  assert.equal(parseSms(message('Credit card payment of INR 500 received')).draft.kind, 'transfer');
  for (const body of [
    'Your OTP is 123456 for purchase INR 100',
    'Offer: spent INR 500 to win',
    'Avl balance INR 2000',
    'Debited INR 100 and credited INR 200',
    'Debited INR 100 and spent INR 200',
  ]) assert.equal(parseSms(message(body)).draft, undefined);
  assert.equal(parseSms(message('Debited INR 100', 'FRIEND')).draft, undefined);
});

test('fingerprints are stable for scan retries and distinguish timestamps', async () => {
  const input = message('Debited INR 100');
  assert.equal(await smsFingerprint(input), await smsFingerprint({ ...input, body: ' Debited   INR 100 ' }));
  assert.notEqual(await smsFingerprint(input), await smsFingerprint({ ...input, receivedAt: '2026-10-21T10:01:00Z' }));
});

test('refunds reduce expenses rather than becoming income', () => {
  assert.equal(parseSms(message('INR 250.00 refunded to card XX1234')).draft.kind, 'refund');
});

test('amount punctuation and ambiguity never produce partial values', () => {
  assert.equal(parseSms(message('Debited INR 12,345.67.')).draft.amount_paise, 1234567);
  assert.equal(parseSms(message('Debited INR 1,23,456.78.')).draft.amount_paise, 12345678);
  assert.equal(parseSms(message('Debited INR 12.345')).draft, undefined);
  assert.throws(() => parseSms(message('Debited INR 1,2')), /amount/);
  assert.equal(parseSms(message('Card statement: Total due INR 300. Min due INR 50. Due on 31/02/2026')).draft.due_date, '');
});
