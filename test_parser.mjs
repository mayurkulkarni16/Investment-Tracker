import { readFileSync } from 'fs';

// Inline the parser logic for testing
const MONTHS = { Jan: '01', Feb: '02', Mar: '03', Apr: '04', May: '05', Jun: '06', Jul: '07', Aug: '08', Sep: '09', Oct: '10', Nov: '11', Dec: '12' };

function parseIndianNumber(s) { return parseFloat(s.replace(/,/g, '')); }
function formatDateISO(day, month, year) {
  const m = MONTHS[month] || '01';
  return `${year}-${m}-${day.padStart(2, '0')}`;
}

function parseCASText(rawText) {
  let text = rawText;
  text = text.replace(/Need Help\?[\s\S]*?Page \d+ of \d+/g, '\n');
  text = text.replace(/₹\s*/g, '₹');
  text = text.replace(/Purchase\s*-\s*\n\s*(Lumpsum|SIP)/g, 'Purchase - $1');

  const lines = text.split('\n').map(l => l.trim()).filter(Boolean);

  let startIdx = 0;
  for (let i = 0; i < lines.length; i++) {
    if (/^Date\*/.test(lines[i])) { startIdx = i + 1; break; }
  }

  const txText = lines.slice(startIdx).join('\n');
  const dataRegex = /(\d{5,})\s+(Purchase\s*-\s*(?:SIP|Lumpsum)|Withdraw)\s+([\d,]+\.?\d*)\s+₹([\d,.]+)\s+₹([\d,.]+)\s+(Confirmed|Failed|Pending|Rejected)/g;

  const matches = [];
  let m;
  while ((m = dataRegex.exec(txText)) !== null) {
    matches.push({
      fullMatch: m[0], startPos: m.index, endPos: m.index + m[0].length,
      folio: m[1], type: m[2], units: parseIndianNumber(m[3]),
      nav: parseIndianNumber(m[4]), amount: parseIndianNumber(m[5]),
    });
  }

  const dateMonthRegex = /(\d{2})\s+([A-Z][a-z]{2})/;
  const yearRegex = /\b(20\d{2})\b/;
  const categoryRegex = /\(([^)]+)\)/;

  const transactions = [];

  for (let idx = 0; idx < matches.length; idx++) {
    const match = matches[idx];
    const prevEnd = idx > 0 ? matches[idx - 1].endPos : 0;
    let beforeText = txText.substring(prevEnd, match.startPos);
    beforeText = beforeText.replace(/^\s*(Confirmed|Failed|Pending|Rejected)\s*/i, '');

    const dm = beforeText.match(dateMonthRegex);
    if (!dm) continue;

    const yr = beforeText.match(yearRegex);
    if (!yr) continue;

    const cat = beforeText.match(categoryRegex);
    const category = cat ? cat[1] : '';

    const yearPos = beforeText.indexOf(yr[0]);
    const catPos = cat ? beforeText.indexOf(cat[0]) : beforeText.length;

    let fundNameBlock;
    if (catPos > yearPos) {
      fundNameBlock = beforeText.substring(yearPos + yr[0].length, catPos);
    } else {
      fundNameBlock = beforeText.substring(yearPos + yr[0].length);
    }

    let fundName = fundNameBlock.replace(/\n/g, ' ').replace(/\s+/g, ' ').trim();
    fundName = fundName.replace(/\([^)]*\)/g, '').trim();
    fundName = fundName.replace(/\d{5,}.*$/, '').trim();
    fundName = fundName.replace(/\s*-\s*$/, '').trim();

    if (!fundName) {
      const prev = transactions.find(t => t.folio === match.folio);
      fundName = prev ? prev.fundName : 'Unknown Fund';
    }

    transactions.push({
      day: dm[1], month: dm[2], year: yr[1],
      fundName, category, folio: match.folio, type: match.type,
      units: match.units, nav: match.nav, amount: match.amount,
    });
  }

  // Group by folio
  const fundMap = new Map();
  for (const tx of transactions) {
    if (!fundMap.has(tx.folio)) {
      fundMap.set(tx.folio, { fund_name: tx.fundName, category: tx.category, folio_number: tx.folio, transactions: [] });
    }
    const fund = fundMap.get(tx.folio);
    if (tx.fundName.length > fund.fund_name.length) fund.fund_name = tx.fundName;
    if (tx.category && !fund.category) fund.category = tx.category;
    fund.transactions.push({ date: formatDateISO(tx.day, tx.month, tx.year), type: tx.type, amount: tx.amount, nav: tx.nav, units: tx.units });
  }

  for (const fund of fundMap.values()) {
    fund.transactions.sort((a, b) => a.date.localeCompare(b.date));
  }

  return Array.from(fundMap.values());
}

// Test with pdfjs output
const text = readFileSync('cas_pdfjs_output.txt', 'utf-8');
const funds = parseCASText(text);

console.log(`Found ${funds.length} funds:`);
for (const f of funds) {
  console.log(`  ${f.fund_name} (${f.folio_number}) - ${f.category} - ${f.transactions.length} txns`);
}
const totalTx = funds.reduce((s, f) => s + f.transactions.length, 0);
console.log(`Total transactions: ${totalTx}`);

// Show first fund details
if (funds.length > 0) {
  console.log('\nSample transactions from first fund:');
  for (const tx of funds[0].transactions.slice(0, 5)) {
    console.log(`  ${tx.date} ${tx.type} ${tx.units} units @ ₹${tx.nav} = ₹${tx.amount}`);
  }
}
