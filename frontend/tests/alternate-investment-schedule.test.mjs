import test from 'node:test';
import assert from 'node:assert/strict';
import { previewAltSchedule } from '../src/utils/alternateInvestmentSchedule.ts';

const isoDay = date => date.toISOString().slice(0, 10);
const terms = { asset_type: 'livestock', purchase_date: '2026-11-22', tenure_months: 12, units: 1, monthly_payout_per_unit: 20000 };

test('fixed livestock preview spans the full billing term, including a partial final month', () => {
  const { billingStart, payouts } = previewAltSchedule(terms);
  assert.equal(isoDay(billingStart), '2026-12-07');
  assert.equal(payouts.length, 13);
  assert.deepEqual([payouts[0], payouts[1], payouts.at(-1)].map(p => [isoDay(p.date), p.amount]), [
    ['2027-01-05', 16129.03], ['2027-02-05', 20000], ['2028-01-05', 3870.97],
  ]);
});

test('livestock proration uses the actual number of days in February', () => {
  const { payouts } = previewAltSchedule({ ...terms, purchase_date: '2028-02-14', tenure_months: 1 });
  assert.deepEqual(payouts.map(p => [isoDay(p.date), p.amount]), [['2028-03-05', 689.66], ['2028-04-05', 18064.52]]);
});

test('bikes and other assets do not pay in the month of purchase or prorate', () => {
  for (const asset_type of ['vehicle', 'other']) {
    const { billingStart, payouts } = previewAltSchedule({ ...terms, asset_type, purchase_date: '2026-11-02', tenure_months: 2 });
    assert.equal(billingStart, null);
    assert.deepEqual(payouts.map(p => [isoDay(p.date), p.amount]), [['2026-12-05', 20000], ['2027-01-05', 20000]]);
  }
});
