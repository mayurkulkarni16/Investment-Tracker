const banks = [
  ['HDFC', /\bHDFC(?:BK|BANK)?\b/i],
  ['ICICI', /\bICICI(?:B|BANK)?\b/i],
  ['Axis', /\bAXIS(?:BK|BANK)?\b/i],
  ['SBI', /\b(?:SBI(?:INB|BNK|CRD|PSB|UPI)?|SBICARD)\b/i],
  ['OneCard', /\b(?:ONECARD|ONECRD|ONCRD)\b/i],
  ['YES BANK', /\bYES(?:BNK|BANK)\b/i],
];
const amountToken = /(?:INR|Rs\.?|₹)\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)(?![0-9,]|\.[0-9])/gi;

export function toPaise(value) {
  const text = String(value).trim();
  if (!/^(?:\d+|\d{1,3}(?:,\d{3})+|\d{1,2}(?:,\d{2})*,\d{3})(?:\.\d{1,2})?$/.test(text)) {
    throw new Error('Enter an INR amount with at most two decimal places.');
  }
  const [whole, fraction = ''] = text.replaceAll(',', '').split('.');
  const result = Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
  if (!Number.isSafeInteger(result) || result <= 0 || result > 1_000_000_000_000) {
    throw new Error('Amount must be positive and at most INR 10 billion.');
  }
  return result;
}

export function localDate(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) throw new Error('Invalid message date.');
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function isoDate(day, month, year) {
  const date = `${year.length === 2 ? `20${year}` : year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`;
  const parsed = new Date(`${date}T12:00:00Z`);
  return !Number.isNaN(parsed.getTime()) && parsed.toISOString().slice(0, 10) === date ? date : '';
}

function dueDate(body) {
  const match = body.match(/due(?:\s+(?:date|on|by|is))*\s*[:\-]?\s*(\d{1,2})[/-](\d{1,2})[/-](\d{4}|\d{2})\b/i);
  return match ? isoDate(match[1], match[2], match[3]) : '';
}

// Uses the date printed in the alert ("on 01-11-26") when it is plausible, else the receipt date.
function transactionDate(body, received) {
  const match = body.match(/\bon\s+(\d{1,2})[/-](\d{1,2})[/-](\d{4}|\d{2})\b/i);
  const date = match ? isoDate(match[1], match[2], match[3]) : '';
  if (!date || date > received) return received;
  const ageDays = (Date.parse(`${received}T00:00:00Z`) - Date.parse(`${date}T00:00:00Z`)) / 86_400_000;
  return ageDays <= 31 ? date : received;
}

function merchant(body) {
  const match = body.match(/\b(?:to|at)\s+([^\n]{2,40}?)\s*(?:\.(?:\s|$)|\n|$)/i);
  const name = match?.[1].trim() ?? '';
  return /\b(?:a\/c|account|card|vpa)\b/i.test(name) ? '' : name;
}

export function parseSms({ sender, body, receivedAt }) {
  if (!sender?.trim() || !body?.trim()) throw new Error('Sender and message text are required.');
  if (body.length > 5000) throw new Error('Message is too long.');
  const bank = banks.find(([, pattern]) => pattern.test(sender.replaceAll('-', ' ')))?.[0];
  if (!bank) return { reason: 'Unsupported sender. Enter the transaction manually.' };
  if (/\b(?:OTP|one[- ]time password|verification code|offer|pre[- ]approved)\b/i.test(body)) {
    return { reason: 'OTP, promotional or verification message excluded.' };
  }
  const received = localDate(receivedAt);
  const bill = /\b(?:statement|bill|due\s+(?:on|date|by))\b/i.test(body) && /\b(?:total\s+(?:amount\s+|amt\.?\s+)?due|payment\s+due)\b/i.test(body);
  const debit = /\b(?:debited|spent|sent|purchase(?:d)?|withdrawn|charged)\b/i.test(body);
  const credit = /\b(?:credited|received|refund(?:ed)?|reversed)\b/i.test(body);
  const payment = /\b(?:payment|paid)\b/i.test(body) && /\b(?:credit\s*card|card|cc)\b/i.test(body) && !bill;
  const transfer = /\b(?:self[- ]transfer|own account|credit card payment|cc payment)\b/i.test(body) || payment;
  if (!bill && !debit && !credit && !payment) return { reason: 'No supported transaction or bill event detected.' };
  if (!bill && debit && credit && !transfer) return { reason: 'Both debit and credit appear. Enter manually to avoid choosing the wrong event.' };

  const amounts = [...body.matchAll(amountToken)];
  const selected = amounts.filter(match => {
    const before = body.slice(Math.max(0, match.index - 45), match.index);
    const after = body.slice(match.index + match[0].length, match.index + match[0].length + 50);
    if (bill) {
      if (/\bmin(?:imum)?\.?\s+(?:(?:amount|amt\.?)\s+)?(?:payment\s+)?due\s*[:\-]?\s*$/i.test(before)) return false;
      return /(?:total\s+(?:amount\s+|amt\.?\s+)?due|payment\s+due)\s*[:\-]?\s*$/i.test(before);
    }
    if (/\b(?:balance|bal|limit|minimum|min\.?\s*due|available)\s*[:.\-]?\s*$/i.test(before)) return false;
    return /(?:debited|spent|sent|purchase(?:d)?|withdrawn|charged|credited|received|refunded|reversed|paid|payment(?:\s+of)?|for|of)\s*[:\-]?\s*$/i.test(before)
      || /^\s*(?:has been\s+|is\s+)?(?:debited|credited|spent|paid|received|withdrawn|charged|refunded|reversed|was spent)/i.test(after);
  });
  if (selected.length !== 1) return { reason: 'Could not identify one unambiguous transaction amount. Enter manually.' };
  const amountPaise = toPaise(selected[0][1]);
  const account = body.match(/(?:a\/c|acct?|account|card)(?:\s+(?:no\.?|ending(?:\s+with)?|xx|number|with))?[\s:*xX.\-]*(\d{4})(?!\d)/i)?.[1] ?? '';
  const refund = /\b(?:refund(?:ed)?|reversed)\b/i.test(body);
  const kind = bill ? 'bill' : transfer ? 'transfer' : refund ? 'refund' : credit ? 'income' : 'expense';
  const payee = kind === 'expense' ? merchant(body) : '';
  return {
    draft: {
      source: 'sms', status: 'pending', kind, amount_paise: amountPaise,
      date: bill ? received : transactionDate(body, received), bank, account, due_date: bill ? dueDate(body) : '',
      profile_id: '', linked_account_type: '', linked_account_id: '',
      sms_received_at: new Date(receivedAt).toISOString(),
      description: payee ? `${payee} (${bank})` : `${bank} ${bill ? 'card bill' : transfer ? 'card payment / transfer' : credit ? 'credit / refund - review classification' : 'debit / card spend'}`,
      category: 'Uncategorized',
    },
    warning: 'Review amount, transaction date (taken from the message when present, otherwise the SMS receipt date), income vs refund/transfer, and existing records. SMS is not a bank statement.',
  };
}

export async function smsFingerprint(message) {
  const canonical = `${message.sender.trim().toUpperCase()}\n${new Date(message.receivedAt).toISOString()}\n${message.body.trim().replace(/\s+/g, ' ')}`;
  const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(canonical));
  return Array.from(new Uint8Array(hash), byte => byte.toString(16).padStart(2, '0')).join('');
}

export async function manualSourceKey() {
  const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(crypto.randomUUID()));
  return Array.from(new Uint8Array(hash), byte => byte.toString(16).padStart(2, '0')).join('');
}
