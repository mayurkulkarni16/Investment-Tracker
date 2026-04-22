import * as pdfjsLib from 'pdfjs-dist';
import pdfjsWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url';

pdfjsLib.GlobalWorkerOptions.workerSrc = pdfjsWorker;

export interface ParsedCASTransaction {
  date: string;
  type: string;
  amount: number;
  nav: number;
  units: number;
}

export interface ParsedCASFund {
  fund_name: string;
  category: string;
  folio_number: string;
  transactions: ParsedCASTransaction[];
}

const MONTHS: Record<string, string> = {
  Jan: '01', Feb: '02', Mar: '03', Apr: '04', May: '05', Jun: '06',
  Jul: '07', Aug: '08', Sep: '09', Oct: '10', Nov: '11', Dec: '12',
};

function parseIndianNumber(s: string): number {
  return parseFloat(s.replace(/,/g, ''));
}

function formatDateISO(day: string, month: string, year: string): string {
  const m = MONTHS[month] || '01';
  return `${year}-${m}-${day.padStart(2, '0')}`;
}

export async function extractTextFromPDF(file: File, password: string): Promise<string> {
  const arrayBuffer = await file.arrayBuffer();
  const pdf = await pdfjsLib.getDocument({ data: arrayBuffer, password }).promise;
  const textParts: string[] = [];

  for (let i = 1; i <= pdf.numPages; i++) {
    const page = await pdf.getPage(i);
    const content = await page.getTextContent();
    let lastY: number | null = null;
    for (const item of content.items) {
      if (!('str' in item)) continue;
      const textItem = item as { str: string; transform: number[] };
      const y = textItem.transform[5];
      if (lastY !== null && Math.abs(y - lastY) > 5) {
        textParts.push('\n');
      } else if (lastY !== null) {
        textParts.push(' ');
      }
      textParts.push(textItem.str);
      lastY = y;
    }
    textParts.push('\n');
  }

  return textParts.join('');
}

export function parseCASText(rawText: string): ParsedCASFund[] {
  // Normalize: remove page headers/footers, trim lines
  let text = rawText;
  text = text.replace(/Need Help\?[\s\S]*?Page \d+ of \d+/g, '\n');

  // Normalize ₹ with optional space
  text = text.replace(/₹\s*/g, '₹');

  // Join "Purchase -\nLumpsum" across lines
  text = text.replace(/Purchase\s*-\s*\n\s*(Lumpsum|SIP)/g, 'Purchase - $1');

  // Split into trimmed non-empty lines
  const lines = text.split('\n').map(l => l.trim()).filter(Boolean);

  // Find start after header
  let startIdx = 0;
  for (let i = 0; i < lines.length; i++) {
    if (/^Date\*/.test(lines[i])) {
      startIdx = i + 1;
      break;
    }
  }

  // Rejoin all transaction text into one big string then re-split
  // This handles cases where data is on the same line as fund name or category
  const txText = lines.slice(startIdx).join('\n');

  // Master regex to find data portions: FOLIO TYPE UNITS ₹NAV ₹AMOUNT STATUS
  // This matches anywhere in text, even mid-line
  const dataRegex = /(\d{5,})\s+(Purchase\s*-\s*(?:SIP|Lumpsum)|Withdraw)\s+([\d,]+\.?\d*)\s+₹([\d,.]+)\s+₹([\d,.]+)\s+(Confirmed|Failed|Pending|Rejected)/g;

  interface RawMatch {
    fullMatch: string;
    startPos: number;
    endPos: number;
    folio: string;
    type: string;
    units: number;
    nav: number;
    amount: number;
  }

  // Find all data matches with positions
  const matches: RawMatch[] = [];
  let m;
  while ((m = dataRegex.exec(txText)) !== null) {
    matches.push({
      fullMatch: m[0],
      startPos: m.index,
      endPos: m.index + m[0].length,
      folio: m[1],
      type: m[2],
      units: parseIndianNumber(m[3]),
      nav: parseIndianNumber(m[4]),
      amount: parseIndianNumber(m[5]),
    });
  }

  // For each match, extract date/fund/category from the text BEFORE it
  // (between previous match end and current match start)
  const dateMonthRegex = /(\d{2})\s+([A-Z][a-z]{2})/;
  const yearRegex = /\b(20\d{2})\b/;
  const categoryRegex = /\(([^)]+)\)/;

  interface ParsedTx {
    day: string; month: string; year: string;
    fundName: string; category: string; folio: string;
    type: string; units: number; nav: number; amount: number;
  }

  const transactions: ParsedTx[] = [];

  for (let idx = 0; idx < matches.length; idx++) {
    const match = matches[idx];
    const prevEnd = idx > 0 ? matches[idx - 1].endPos : 0;
    let beforeText = txText.substring(prevEnd, match.startPos);

    // Clean up any leftover status words from previous match
    beforeText = beforeText.replace(/^\s*(Confirmed|Failed|Pending|Rejected)\s*/i, '');

    // Extract date (DD Mon)
    const dm = beforeText.match(dateMonthRegex);
    if (!dm) continue;

    // Extract year
    const yr = beforeText.match(yearRegex);
    if (!yr) continue;

    // Extract category
    const cat = beforeText.match(categoryRegex);
    const category = cat ? cat[1] : '';

    // Extract fund name: everything between year and category (or between year and end)
    // The fund name is typically after the year line and before category
    const yearPos = beforeText.indexOf(yr[0]);
    const catPos = cat ? beforeText.indexOf(cat[0]) : beforeText.length;

    let fundNameBlock: string;
    if (catPos > yearPos) {
      fundNameBlock = beforeText.substring(yearPos + yr[0].length, catPos);
    } else {
      // Category came before year somehow, grab text after year
      fundNameBlock = beforeText.substring(yearPos + yr[0].length);
    }

    // Clean fund name
    let fundName = fundNameBlock
      .replace(/\n/g, ' ')
      .replace(/\s+/g, ' ')
      .trim();

    // Remove any parenthesized category text e.g. "(Equity - Small Cap)"
    fundName = fundName.replace(/\([^)]*\)/g, '').trim();
    // Remove any trailing/leading folio numbers or data remnants
    fundName = fundName.replace(/\d{5,}.*$/, '').trim();
    // Remove trailing dashes
    fundName = fundName.replace(/\s*-\s*$/, '').trim();

    if (!fundName) {
      // Try to get fund name from matching a previous transaction with same folio
      const prev = transactions.find(t => t.folio === match.folio);
      fundName = prev ? prev.fundName : 'Unknown Fund';
    }

    transactions.push({
      day: dm[1],
      month: dm[2],
      year: yr[1],
      fundName,
      category,
      folio: match.folio,
      type: match.type,
      units: match.units,
      nav: match.nav,
      amount: match.amount,
    });
  }

  // Group by folio number
  const fundMap = new Map<string, ParsedCASFund>();

  for (const tx of transactions) {
    const key = tx.folio;
    if (!fundMap.has(key)) {
      fundMap.set(key, {
        fund_name: tx.fundName,
        category: tx.category,
        folio_number: tx.folio,
        transactions: [],
      });
    }
    // Use longest fund name for the fund
    const fund = fundMap.get(key)!;
    if (tx.fundName.length > fund.fund_name.length) {
      fund.fund_name = tx.fundName;
    }
    if (tx.category && !fund.category) {
      fund.category = tx.category;
    }

    fund.transactions.push({
      date: formatDateISO(tx.day, tx.month, tx.year),
      type: tx.type,
      amount: tx.amount,
      nav: tx.nav,
      units: tx.units,
    });
  }

  // Sort transactions within each fund by date
  for (const fund of fundMap.values()) {
    fund.transactions.sort((a, b) => a.date.localeCompare(b.date));
  }

  return Array.from(fundMap.values());
}
