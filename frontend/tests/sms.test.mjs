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

test('UPI "Sent" alerts are expenses dated from the message, with the payee', () => {
  const sent = { sender: 'JX-HDFCBK-S', body: 'Sent Rs.250.00\nFrom HDFC Bank A/C *1111\nTo Test Store\nOn 19/10/26\nRef 123456789012\nNot You?\nCall 1800000000\n', receivedAt: '2026-10-21T10:00:00Z' };
  const { draft } = parseSms(sent);
  assert.equal(draft.kind, 'expense');
  assert.equal(draft.amount_paise, 25000);
  assert.equal(draft.account, '1111');
  assert.equal(draft.date, '2026-10-19');
  assert.equal(draft.description, 'Test Store (HDFC)');
  assert.equal(parseSms({ ...sent, body: sent.body.replace('19/10/26', '25/10/26') }).draft.date, '2026-10-21');
  const spent = parseSms(message('Spent INR 900 \nusing Axis Bank Credit Card no. XX2222 on 20-10-26 06:17:24 IST at \nTESTSHOP. \nAvl limit: INR 5000.', 'AX-AXISBK-S')).draft;
  assert.deepEqual([spent.kind, spent.amount_paise, spent.date, spent.description], ['expense', 90000, '2026-10-20', 'TESTSHOP (Axis)']);
});

test('card due reminders with total due become bills; card payments read as transfers', () => {
  const bill = parseSms(message('Payment of Credit Card X3333 is due on 05/11/26. Min due Rs.100.00. Total Due Rs.1200.50. Pay before last date.', 'AX-YESBNK-S')).draft;
  assert.deepEqual([bill.kind, bill.amount_paise, bill.account, bill.due_date], ['bill', 120050, '3333', '2026-11-05']);
  const paid = parseSms(message('Payment of INR 500.00 has been received towards your Axis Bank Credit Card XX4444 on 21-10-26', 'AD-AXISBK-S')).draft;
  assert.deepEqual([paid.kind, paid.description], ['transfer', 'Axis card payment / transfer']);
});

test('bank, card, and UPI alerts are recognized outside the old issuer list', () => {
  const cases = [
    ['AX-KOTAKB-S', 'Your debit card ending in 4567 was used for INR 825.50 at Grocery Store on 20/10/26. Available limit INR 10,000.', 'expense', 82550, 'Kotak', '4567'],
    ['VM-RBLBNK-T', 'INR 825.50 was charged to credit card ****4567 at Grocery Store.', 'expense', 82550, 'RBL Bank', '4567'],
    ['VM-NEWBNK-S', 'A/c XX6789 debited by Rs 200/- via IMPS. UTR 987654321012. Available balance is Rs 2000.', 'expense', 20000, '', '6789'],
    ['VM-NEWBNK-S', 'NEFT transfer of INR 2,500.00 to your account **9876. UTR TEST123456789.', 'income', 250000, '', '9876'],
    ['VM-HDFCBK-S', 'RTGS transfer of INR 2,500.00 from your account XX1111 to own account XX2222.', 'transfer', 250000, 'HDFC', ''],
    ['AX-GPAY-S', 'UPI payment of ₹250.50 sent to Example Store. Ref 987654321012.', 'expense', 25050, '', ''],
    ['AX-NAVIUP-S', 'Paid Rs. 75 using UPI from Kotak A/c *1234. UPI Ref: 987654321012.', 'expense', 7500, 'Kotak', '1234'],
    ['VM-KIWI-S', 'Paid INR 755 using your credit card XX5432 via UPI.', 'expense', 75500, '', '5432'],
    ['VM-CREDUPI-S', 'Received INR 150 via UPI. Transaction ID: TEST123456789.', 'income', 15000, '', ''],
    ['AX-CRED-S', 'Payment of Rs. 1,550.25 was received towards your Axis Bank credit card XX6789.', 'transfer', 155025, 'Axis', '6789'],
  ];
  for (const [sender, body, kind, paise, bank, account] of cases) {
    const parsed = parseSms(message(body, sender));
    assert.ok(parsed.draft, `${sender}: ${parsed.reason}`);
    assert.deepEqual(
      [parsed.draft.kind, parsed.draft.amount_paise, parsed.draft.bank, parsed.draft.account, parsed.draft.status],
      [kind, paise, bank, account, 'pending'], sender,
    );
  }
});

test('SBI bare amounts, counterparty-credited UPI debits, and named-month dates parse', () => {
  const cases = [
    ['VM-SBIUPI-S', 'Dear UPI user A/C X4321 debited by 75.00 on date 20Oct26 trf to TESTSHOP Refno 612345678901', 'expense', 7500, 'SBI', '4321', '2026-10-20'],
    ['AX-ICICIT-S', 'ICICI Bank Acct XX123 debited for Rs 250.00 on 20-Oct-26; TESTSHOP credited. UPI:612345678901.', 'expense', 25000, 'ICICI', '', '2026-10-20'],
    ['VM-IPBMSG-S', 'Your a/c no. XXXXXXXX1234 is debited for Rs.89.00 on 20-10-2026 and credited to a/c no. XXXXXXXX5678 (UPI Ref no 612345678901)', 'expense', 8900, 'India Post Payments Bank', '1234', '2026-10-20'],
    ['AD-BOIIND-S', 'Rs.500.00 withdrawn at ATM from A/c XX3344 on 2026-10-20. Avl Bal Rs.9000', 'expense', 50000, 'Bank of India', '3344', '2026-10-20'],
    ['JD-HDFCBK-S', 'Txn Rs.1499.00 On HDFC Bank Card 1234 At TESTSHOP by UPI 612345678901 On 20-10-26', 'expense', 149900, 'HDFC', '1234', '2026-10-20'],
    ['BP-AXISBK-S', 'Spent Card no. XX1234 INR 899 on 20-10-26 10:12:13 TESTSHOP Avl Lmt INR 50000', 'expense', 89900, 'Axis', '1234', '2026-10-20'],
    ['VK-KOTAKB-S', 'Sent Rs.120.00 from Kotak Bank AC X5678 to abc@okaxis on 20-10-26.UPI Ref 432112345678.', 'expense', 12000, 'Kotak', '5678', '2026-10-20'],
  ];
  for (const [sender, body, kind, paise, bank, account, date] of cases) {
    const parsed = parseSms(message(body, sender));
    assert.ok(parsed.draft, `${sender}: ${parsed.reason}`);
    assert.deepEqual([parsed.draft.kind, parsed.draft.amount_paise, parsed.draft.bank, parsed.draft.account, parsed.draft.date],
      [kind, paise, bank, account, date], sender);
  }
  assert.equal(parseSms(message('ICICI Bank Acct XX123 debited for Rs 250.00; TESTSHOP credited.', 'AX-ICICIT-S')).draft.description, 'TESTSHOP (ICICI)');
  assert.equal(parseSms(message('INR 250.00 refunded to debit card XX1234')).draft.kind, 'refund');
});

test('broader coverage does not treat requests, declined spends, fees or balances as completed amounts', () => {
  for (const body of [
    'Payment request received for INR 125 via UPI. Ref 987654321012.',
    'UPI transaction failed. INR 125 was not debited from A/c XX4321.',
    'Card XX1234 was declined for INR 125.',
    'Account XX1234 was debited. Available balance as of INR 9999.',
    'Card XX1234 charged USD 10. Convenience fee of INR 25.',
    'Card XX1234 used for INR 250 and charged INR 200.',
    'NEFT completed. A/c XX1234 is credited INR 25 and debited INR 25.',
  ]) assert.equal(parseSms(message(body)).draft, undefined, body);
  assert.equal(parseSms(message('Sent INR 250. Call me later.', 'FRIEND')).draft, undefined);
});
