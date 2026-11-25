const banks = [
  ['HDFC', /\bHDFC(?:BK|BANK)?\b/i],
  ['ICICI', /\bICICI(?:B|T|O|BANK)?\b/i],
  ['Axis', /\bAXIS(?:BK|BANK)?\b/i],
  ['SBI', /\b(?:SBI(?:INB|BNK|CRD|PSB|UPI)?|SBICARD)\b/i],
  ['OneCard', /\b(?:ONECARD|ONECRD|ONCRD)\b/i],
  ['YES BANK', /\bYES(?:BNK|\s*BANK)\b/i],
  ['Kotak', /\bKOTAK(?:B|BK|BANK|\s+MAHINDRA)?\b/i],
  ['IDFC FIRST Bank', /\b(?:IDFCFB|IDFCFBK|IDFCBK|IDFC\s+FIRST)\b/i],
  ['IndusInd', /\b(?:INDUSB|INDUSIND(?:BK|BANK)?)\b/i],
  ['Indian Bank', /\b(?:INDBNK|INDIAN\s+BANK)\b/i],
  ['Bank of India', /\b(?:BOIIND|BOISMS)\b/i],
  ['IDBI Bank', /\b(?:IDBIBK|IDBIBANK|IDBI\s+BANK)\b/i],
  ['Central Bank of India', /\b(?:CBSSBI|CENTBK|CBOI)\b/i],
  ['Bank of Maharashtra', /\b(?:MAHABK|MAHABANK)\b/i],
  ['Indian Overseas Bank', /\b(?:IOBCHN|IOBBNK|IOBANK)\b/i],
  ['UCO Bank', /\b(?:UCOBNK|UCOBANK|UCO\s+BANK)\b/i],
  ['India Post Payments Bank', /\b(?:IPBMSG|IPPBNK|IPPB)\b/i],
  ['Paytm Payments Bank', /\b(?:PAYTMB|PYTMBK)\b/i],
  ['Airtel Payments Bank', /\b(?:AIRBNK|AIRTEL\s+PAYMENTS\s+BANK)\b/i],
  ['Equitas', /\b(?:EQUTAS|EQUITAS)\b/i],
  ['Ujjivan', /\b(?:UJJIVN|UJJIVAN)\b/i],
  ['DBS Bank', /\b(?:DBSBNK|DIGIBK)\b/i],
  ['AU Small Finance Bank', /\b(?:AUBANK|AUSFBK|AU\s+SMALL\s+FINANCE)\b/i],
  ['Bank of Baroda', /\b(?:BOBSMS|BARODA|BANK\s+OF\s+BARODA)\b/i],
  ['Punjab National Bank', /\b(?:PNBSMS|PNBBNK|PNB|PUNJAB\s+NATIONAL\s+BANK)\b/i],
  ['Canara Bank', /\b(?:CANBNK|CANARA(?:\s+BANK)?)\b/i],
  ['Union Bank', /\b(?:UNIONB|UNION\s+BANK)\b/i],
  ['Federal Bank', /\b(?:FEDBNK|FEDERAL\s+BANK)\b/i],
  ['RBL Bank', /\b(?:RBLBNK|RBLBANK|RBL\s+BANK)\b/i],
  ['Standard Chartered', /\b(?:SCBANK|STANCH|STANDARD\s+CHARTERED)\b/i],
  ['HSBC', /\bHSBC(?:BK|BANK)?\b/i],
];
const amountToken = /(?:\b(?:INR|Rs)\.?\s*|₹\s*)([0-9][0-9,]*(?:\.[0-9]{1,2})?)(?![0-9,]|\.[0-9])/gi;
const withdrawal = /\b(?:debited|debit(?!\s+card)|spent|sent|purchased|withdrawn|withdrawal|charged|used|paid)\b/i;
const deposit = /\b(?:credited|received|refund(?:ed)?|reversed|deposited)\b/i;
const rail = /\b(?:UPI|NEFT|IMPS|RTGS)\b/i;
const accountLabel = /\b(?:a\s*\/\s*c|ac|acct?|account|card)\b/i;
// Card alerts with no verb, e.g. HDFC "Txn Rs.1499 On HDFC Bank Card 1234 At AMAZON".
const cardTxn = /\b(?:txn|transaction)\s+(?:of\s+)?(?:INR|Rs\.?|₹)\s*[0-9][^;\n]{0,60}\bcard\b/i;
const excluded = /\b(?:OTP|one[- ]time password|verification code|offer|pre[- ]approved|cashback offer)\b/i;
const notCompleted = /\b(?:failed|declined|unsuccessful|cancelled|canceled|not\s+(?:debited|credited|completed|processed|successful)|pending|scheduled|requested|request\s+(?:to\s+pay|(?:was\s+)?sent|received)|(?:collect|payment)\s+request|payment\s+reminder)\b/i;
const repaymentLabel = /\b(?:credit\s*card|cc)\s+(?:bill\s+)?payment\b/i;
const repaymentTarget = /\b(?:towards?|to|for|on)\s+(?:your\s+)?(?:\w+\s+){0,4}(?:credit\s*card|cc)\b/i;
const transferFrom = /\b(?:UPI|NEFT|IMPS|RTGS)\b[^;\n]{0,90}\b(?:transfer(?:red)?|txn|transaction)\b[^;\n]{0,80}\bfrom\s+(?:your\s+)?(?:a\/c|account)\b/i;
const transferTo = /\b(?:UPI|NEFT|IMPS|RTGS)\b[^;\n]{0,90}\b(?:transfer(?:red)?|txn|transaction)\b[^;\n]{0,80}\bto\s+(?:your\s+)(?:a\/c|account)\b/i;
// Bare amount directly after the verb, e.g. SBI UPI "debited by 75.00".
const bareAmount = /\b(?:debited|credited)\s+(?:by|for|with)\s*[:\-]?\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)(?![0-9,]|\.[0-9])/gi;
// The other party of the same transaction, e.g. ICICI "debited for Rs 250; SWIGGY credited" or "and credited to a/c XX5678".
const counterpartyCredited = /;\s*([^;.\n\d₹]{2,40}?)\s+credited\b|\b(?:and|&)\s+credited\s+to\s+(?:a\s*\/\s*c|acct?|account|vpa|beneficiary)\b/i;
const counterpartyDebited = /;\s*([^;.\n\d₹]{2,40}?)\s+debited\b|\b(?:and|&)\s+debited\s+from\s+(?:a\s*\/\s*c|acct?|account|vpa|remitter)\b/i;
const months = { jan: 1, feb: 2, mar: 3, apr: 4, may: 5, jun: 6, jul: 7, aug: 8, sep: 9, oct: 10, nov: 11, dec: 12 };

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
  const numeric = body.match(/\bon\s*[:,-]?\s*(\d{1,2})[/-](\d{1,2})[/-](\d{4}|\d{2})\b/i);
  const named = body.match(/\bon\s*(?:date\s*)?[:,-]?\s*(\d{1,2})[\s/-]?(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*[\s/-]?(\d{4}|\d{2})\b/i);
  const iso = body.match(/\bon\s*[:,-]?\s*(\d{4})-(\d{2})-(\d{2})\b/i);
  const date = numeric ? isoDate(numeric[1], numeric[2], numeric[3])
    : named ? isoDate(named[1], String(months[named[2].toLowerCase()]), named[3])
      : iso ? isoDate(iso[3], iso[2], iso[1]) : '';
  if (!date || date > received) return received;
  const ageDays = (Date.parse(`${received}T00:00:00Z`) - Date.parse(`${date}T00:00:00Z`)) / 86_400_000;
  return ageDays <= 31 ? date : received;
}

function merchant(body) {
  const match = body.match(/\b(?:to|at)[\s:]+([^;\n]{2,40}?)\s*(?=\s+(?:on|via|using|ref(?:no)?|txn|utr|upi|avl|bal(?:ance)?)\b|\.(?:\s|$)|[;\n]|$)/i);
  return cleanName(match?.[1]);
}

function cleanName(value) {
  const name = value?.trim() ?? '';
  return !name || accountLabel.test(name) || !/^[\p{L}\p{M}0-9 &'()_-]+$/u.test(name) || /\d{5,}|\bvpa\b/i.test(name) ? '' : name;
}

export function parseSms({ sender, body, receivedAt }) {
  if (!sender?.trim() || !body?.trim()) throw new Error('Sender and message text are required.');
  if (body.length > 5000) throw new Error('Message is too long.');
  const senderId = sender.trim().replace(/^[a-z0-9]{2}-/i, '').replace(/-[a-z]$/i, '');
  const issuer = banks.find(([, pattern]) => pattern.test(senderId))?.[0];
  if (excluded.test(body)) {
    return { reason: 'OTP, promotional or verification message excluded.' };
  }
  const bill = /\b(?:statement|bill|due\s+(?:on|date|by))\b/i.test(body) && /\b(?:total\s+(?:amount\s+|amt\.?\s+)?due|payment\s+due)\b/i.test(body);
  if (!bill && notCompleted.test(body)) return { reason: 'Transaction is not clearly completed.' };
  let debit = withdrawal.test(body) || transferFrom.test(body) || (cardTxn.test(body) && !deposit.test(body));
  let credit = deposit.test(body) || (!debit && transferTo.test(body));
  const refund = /\b(?:refund(?:ed)?|reversed)\b/i.test(body);
  const repayment = repaymentLabel.test(body)
    || (/\b(?:payment|paid)\b/i.test(body) && repaymentTarget.test(body))
    || /\bpayment\b[^;\n]{0,100}\breceived\s+(?:on|for|towards?)\b[^;\n]{0,60}\bcard\b/i.test(body);
  const transfer = /\b(?:self[- ]transfer|own\s+(?:bank\s+)?account|between your accounts)\b/i.test(body) || repayment;
  const accountIds = new Set([...body.matchAll(/\b(?:a\s*\/\s*c|ac|acct?|account|card)\s*(?:(?:no\.?|ending(?:\s+(?:in|with))?|number|with)\s*)?[:\-]?\s*[*xX.\s]*(\d{4})(?!\d)/gi)].map(match => match[1]));
  const reference = /\b(?:ref(?:erence)?(?:\s+(?:no\.?|id))?|refno|utr|txn\s+id)\s*[:#-]?\s*[A-Z0-9]{6,}\b|\bUPI\s*[:#-]?\s*\d{6,}\b/i.test(body);
  const brandedApp = /^(?:GPAY|GOOGLE\s*PAY|NAVIFIN|NAVI(?:UPI)?|KIWI(?:UPI)?|CRED(?:UPI|PAY|CLUB)?)$/i.test(senderId);
  if (!issuer && !(accountLabel.test(body) && (accountIds.size > 0 || reference))
    && !(rail.test(body) && (reference || brandedApp))) {
    return { reason: 'Not enough evidence of a bank or payment alert.' };
  }
  if (!bill && !debit && !credit && !repayment) return { reason: 'No supported transaction or bill event detected.' };
  let counterparty = null;
  if (!bill && debit && credit && !transfer) {
    const debitAt = body.search(withdrawal);
    const creditAt = body.search(deposit);
    if (debitAt >= 0 && creditAt > debitAt && (counterparty = body.slice(debitAt).match(counterpartyCredited))) credit = false;
    else if (creditAt >= 0 && debitAt > creditAt && (counterparty = body.slice(creditAt).match(counterpartyDebited))) debit = false;
    else return { reason: 'Both debit and credit appear. Enter manually to avoid choosing the wrong event.' };
  }
  const account = accountIds.size === 1 || (counterparty && accountIds.size) ? [...accountIds][0] : '';

  const amounts = [...body.matchAll(amountToken)];
  const selected = amounts.filter((match, index) => {
    const previous = amounts[index - 1];
    const before = body.slice(Math.max(match.index - 120, previous ? previous.index + previous[0].length : 0), match.index);
    const after = body.slice(match.index + match[0].length, match.index + match[0].length + 80);
    if (bill) {
      if (/\bmin(?:imum)?\.?\s+(?:(?:amount|amt\.?)\s+)?(?:payment\s+)?due\s*[:\-]?\s*$/i.test(before)) return false;
      return /(?:total\s+(?:amount\s+|amt\.?\s+)?due|payment\s+due)\s*[:\-]?\s*$/i.test(before);
    }
    if (/\b(?:balance|bal|limit|minimum|min\.?\s*due|available)\.?\s*(?:(?:is|of|as\s+of)\s+)?[:=\-]?\s*$/i.test(before)
      || /^\s*(?:available|bal(?:ance)?|(?:credit\s+)?limit)\b/i.test(after)) return false;
    return /\b(?:debit(?:ed)?|spent|sent|purchase(?:d)?|withdraw(?:al|n)|charged|credit(?:ed)?|received|deposit(?:ed)?|refund(?:ed)?|reversed|paid|payment|transfer(?:red)?|txn|transaction)\s*(?:(?:for|of|by|with|amount|amt\.?)\s*)?[:=\-]?\s*$/i.test(before)
      || /\b(?:used|charged|spent|debited|credited|paid)\b[^.!?;\n]{0,60}\s+(?:for|by|of)\s*[:=\-]?\s*$/i.test(before)
      || /\b(?:spent|debited|charged|credited)\s+(?:(?:on|from|to)\s+)?(?:your\s+)?(?:[\w-]+\s+){0,3}(?:card|a\s*\/\s*c|ac|acct?|account)\s*(?:no\.?)?\s*[:\-]?\s*[*xX.]*\d{3,4}\s*[:,\-]?\s*$/i.test(before)
      || /^\s*(?:\/-)?\s*(?:(?:has been|was|is|successfully)\s+)*(?:debited|credited|spent|paid|received|sent|deposited|withdrawn|charged|used|refunded|reversed|transferred|remitted)\b/i.test(after);
  });
  const bare = !bill && !selected.length ? [...body.matchAll(bareAmount)] : [];
  const chosen = selected.length ? selected : bare;
  if (chosen.length !== 1) return { reason: 'Could not identify one unambiguous transaction amount. Enter manually.' };
  const amountPaise = toPaise(chosen[0][1]);
  const received = localDate(receivedAt);
  const bodyBanks = banks.filter(([, pattern]) => pattern.test(body));
  const bank = issuer || (bodyBanks.length === 1 ? bodyBanks[0][0] : '');
  const kind = bill ? 'bill' : refund ? 'refund' : transfer ? 'transfer' : credit ? 'income' : 'expense';
  const payee = kind === 'expense' ? merchant(body) || cleanName(counterparty?.[1]) : '';
  const label = bill ? 'card bill' : transfer ? 'card payment / transfer' : credit ? 'credit / refund - review classification' : 'debit / card spend';
  return {
    draft: {
      source: 'sms', status: 'pending', kind, amount_paise: amountPaise,
      date: bill ? received : transactionDate(body, received), bank, account, due_date: bill ? dueDate(body) : '',
      profile_id: '', linked_account_type: '', linked_account_id: '',
      sms_received_at: new Date(receivedAt).toISOString(),
      description: payee ? (bank ? `${payee} (${bank})` : payee) : `${bank || 'SMS'} ${label}`,
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
