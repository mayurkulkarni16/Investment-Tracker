import * as pdfjsLib from 'pdfjs-dist';
import pdfjsWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url';
import Tesseract from 'tesseract.js';

pdfjsLib.GlobalWorkerOptions.workerSrc = pdfjsWorker;

export interface ParsedPFContribution {
  month: string;
  employee_contribution: number;
  employer_contribution: number;
  interest_earned: number;
}

export interface ParsedPFYear {
  financial_year: string;
  opening_balance: number;
  contributions: ParsedPFContribution[];
  interest_earned: number;
  closing_balance: number;
}

export interface ParsedPFData {
  account_number: string;
  employer_name: string;
  years: ParsedPFYear[];
}

export async function extractPFFromPDF(
  file: File,
  onProgress?: (msg: string) => void
): Promise<ParsedPFData> {
  const arrayBuffer = await file.arrayBuffer();
  const data = new Uint8Array(arrayBuffer);
  const doc = await pdfjsLib.getDocument({ data }).promise;

  let fullText = '';
  let totalTextItems = 0;

  // First attempt: direct text extraction (fast and accurate for digital PDFs)
  onProgress?.('Extracting text from PDF...');
  for (let i = 1; i <= doc.numPages; i++) {
    const page = await doc.getPage(i);
    const textContent = await page.getTextContent();
    const items = textContent.items as Array<{ str: string; transform: number[] }>;
    totalTextItems += items.length;

    if (items.length > 0) {
      // Group text items into lines by y-coordinate (PDF y-axis is bottom-up)
      const lines: Array<{ y: number; parts: Array<{ x: number; text: string }> }> = [];

      for (const item of items) {
        const x = item.transform[4];
        const y = item.transform[5];
        const text = item.str;
        if (!text.trim()) continue;

        const existing = lines.find(l => Math.abs(l.y - y) < 8);
        if (existing) {
          existing.parts.push({ x, text });
        } else {
          lines.push({ y, parts: [{ x, text }] });
        }
      }

      // Sort lines top-to-bottom (descending y), then sort parts left-to-right within each line
      lines.sort((a, b) => b.y - a.y);
      for (const line of lines) {
        line.parts.sort((a, b) => a.x - b.x);
        fullText += line.parts.map(p => p.text).join(' ') + '\n';
      }
    }
  }

  console.log(`=== TEXT EXTRACTION (${totalTextItems} items) ===\n`, fullText);

  // If direct extraction yielded meaningful text, use it
  if (totalTextItems > 20) {
    onProgress?.('Parsing extracted text...');
    return parsePFText(fullText);
  }

  // Fallback: OCR for scanned/image-based PDFs
  onProgress?.('PDF has no selectable text, falling back to OCR...');
  fullText = '';

  for (let i = 1; i <= doc.numPages; i++) {
    onProgress?.(`Rendering page ${i}/${doc.numPages}...`);

    const page = await doc.getPage(i);
    const viewport = page.getViewport({ scale: 3.0 });

    const canvas = document.createElement('canvas');
    canvas.width = viewport.width;
    canvas.height = viewport.height;
    const ctx = canvas.getContext('2d')!;
    ctx.fillStyle = '#ffffff';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    await page.render({ canvas, canvasContext: ctx, viewport } as any).promise;

    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    let nonWhitePixels = 0;
    for (let p = 0; p < imageData.data.length; p += 4) {
      if (imageData.data[p] < 250 || imageData.data[p + 1] < 250 || imageData.data[p + 2] < 250) {
        nonWhitePixels++;
      }
    }

    if (nonWhitePixels < 100) {
      canvas.remove();
      continue;
    }

    onProgress?.(`OCR page ${i}/${doc.numPages}...`);
    const { data: ocrData } = await Tesseract.recognize(canvas, 'eng', {
      logger: (m: { status: string; progress: number }) => {
        if (m.status === 'recognizing text') {
          onProgress?.(`OCR page ${i}: ${Math.round(m.progress * 100)}%`);
        }
      },
    });

    console.log(`Page ${i} OCR (confidence: ${ocrData.confidence}%):\n`, ocrData.text);
    fullText += ocrData.text + '\n';
    canvas.remove();
  }

  console.log('=== FULL OCR TEXT ===\n', fullText);
  onProgress?.('Parsing extracted text...');
  return parsePFText(fullText);
}

function parseNumber(s: string): number {
  if (!s) return 0;
  // Remove commas, spaces, and handle common OCR errors
  const cleaned = s.replace(/,/g, '').replace(/\s/g, '').replace(/[oO]/g, '0').replace(/[lI]/g, '1');
  const num = parseFloat(cleaned);
  return isNaN(num) ? 0 : num;
}

export function parsePFText(text: string): ParsedPFData {
  const lines = text.split('\n').map(l => l.trim()).filter(Boolean);

  const result: ParsedPFData = {
    account_number: '',
    employer_name: '',
    years: [],
  };

  // UAN / account number and employer name extraction
  for (const line of lines) {
    const uanMatch = line.match(/(?:UAN|Universal\s*Account\s*Number)[:\s]*(\d{10,})/i);
    if (uanMatch) {
      result.account_number = uanMatch[1];
    }
    // PF member number e.g. PY/BOM/0016573/2332906
    const pfNumMatch = line.match(/(?:PF\s*No|Member\s*Id)[.:\s]*([A-Z]{2}\/[A-Z]+\/\d+\/\d+)/i);
    if (pfNumMatch && !result.account_number) {
      result.account_number = pfNumMatch[1];
    }
    // Employer/establishment name – avoid matching table column headers ("Company", "Contribution Total", etc.)
    const empMatch = line.match(/(?:Establishment|Employer)[:\s]+(.+)/i);
    if (empMatch) {
      const candidate = empMatch[1].trim();
      // Reject if it contains digits (table data) or is too short
      if (candidate.length > 3 && !/\d/.test(candidate)) {
        result.employer_name = candidate;
      }
    }
    // Employee Name line – use as fallback employer context
    const empNameMatch = line.match(/(?:Employee\s*Name)[:\s]+(.+)/i);
    if (empNameMatch && !result.employer_name) {
      // don't use employee name as employer, but extract it for UAN fallback
    }
  }

  // Month patterns: plain "Apr" OR with year suffix "Apr-25" / "Apr/25" / "Apr 25"
  const monthWithYearPattern = /\b(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:tember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)[-\/\s](\d{2,4})\b/i;
  const plainMonthPattern    = /\b(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:tember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\b/i;

  // Financial year: "2024-25", "2025-2026", "2025 - 2026", "FY 2024-25"
  // The bare YYYY-YY pattern is intentionally NOT anchored to keywords here;
  // instead we validate the line context below.
  const fyPattern = /(\d{4})\s*[-–]\s*(\d{2,4})/;

  // A line is treated as a FY header only when it contains one of these signals:
  //   • a keyword like FY / YEAR / FINANCIAL / STATEMENT / ACCUMULATION
  //   • OR the YYYY-YY portion is the ONLY content on the line (standalone heading)
  const fyKeywordRe = /\b(?:FY|year|financial|statement|accumulation|passbook|provident)\b/i;

  let currentYear: ParsedPFYear | null = null;

  // Helper: grab numbers from a line, with optional lookahead if the line itself has none
  const getNumbers = (lineIdx: number): string[] => {
    const nums = lines[lineIdx].match(/[\d,]+\.?\d*/g);
    if (nums && nums.length > 0) return nums;
    // Numbers may be on the very next line (label cell / value cell at different y in pdfjs)
    const next = lines[lineIdx + 1] ?? '';
    const nextNums = next.match(/[\d,]+\.?\d*/g);
    return nextNums ?? [];
  };

  for (let li = 0; li < lines.length; li++) {
    const line = lines[li];

    // ── Financial year header detection ──────────────────────────────────
    const fyMatch = line.match(fyPattern);
    if (fyMatch) {
      const beforeFY = line.substring(0, line.indexOf(fyMatch[0]));
      const afterFY  = line.substring(line.indexOf(fyMatch[0]) + fyMatch[0].length);
      const restText = (beforeFY + afterFY).replace(/[\s\W]/g, '');

      const hasKeyword   = fyKeywordRe.test(line);
      const isStandalone = restText.length < 10;

      if (!plainMonthPattern.test(beforeFY) && (hasKeyword || isStandalone)) {
        const startYear = fyMatch[1];
        const rawEnd   = fyMatch[2];
        const endYear  = rawEnd.length === 2 ? startYear.substring(0, 2) + rawEnd : rawEnd;
        const fy = `${startYear}-${endYear.slice(-2)}`;

        const startYearNum = parseInt(startYear, 10);
        if (startYearNum < 1990 || startYearNum > 2040) continue;

        if (currentYear) result.years.push(currentYear);
        currentYear = {
          financial_year: fy,
          opening_balance: 0,
          contributions: [],
          interest_earned: 0,
          closing_balance: 0,
        };
      }
    }

    if (!currentYear) continue;

    // ── Opening balance ──────────────────────────────────────────────────
    if (/opening\s*balance/i.test(line)) {
      const nums = getNumbers(li);
      if (nums.length >= 6) {
        // 18-column Mphasis main-table row: MC is nums[0], CC is nums[5]; total = MC + CC
        const val = parseNumber(nums[0]) + parseNumber(nums[5]);
        if (val > 0) currentYear.opening_balance = val;
      } else if (nums.length >= 1) {
        const val = parseNumber(nums[nums.length - 1]);
        if (val > 0) currentYear.opening_balance = val;
      }
    }

    // ── Closing balance ──────────────────────────────────────────────────
    if (/closing\s*balance/i.test(line)) {
      const nums = getNumbers(li);
      if (nums.length >= 6) {
        const val = parseNumber(nums[0]) + parseNumber(nums[5]);
        if (val > 0) currentYear.closing_balance = val;
      } else if (nums.length >= 1) {
        const val = parseNumber(nums[nums.length - 1]);
        if (val > 0) currentYear.closing_balance = val;
      }
    }

    // ── Interest earned ──────────────────────────────────────────────────
    if (/interest\s*earned|add\s*:?\s*interest/i.test(line)) {
      const nums = getNumbers(li);
      if (nums.length >= 1) {
        const val = parseNumber(nums[nums.length - 1]);
        if (val > 0) currentYear.interest_earned = val;
      }
    }

    // ── "Total" row in 18-column Mphasis table ───────────────────────────
    // Columns: MC(0) VPF(1) TotalMC(2) NonTax(3) Tax(4) CC(5) ...
    //          Interest MC-NonTax(15) Interest MC-Tax(16) Interest Company(17)
    if (/^total\b/i.test(line) && currentYear.interest_earned === 0) {
      const nums = getNumbers(li);
      if (nums.length >= 18) {
        const interest = parseNumber(nums[15]) + parseNumber(nums[16]) + parseNumber(nums[17]);
        if (interest > 0) currentYear.interest_earned = interest;
      }
      // Also derive opening/closing from Total row contributions if still missing
      // Opening balance = not in Total row; skip
    }

    // ── Monthly contribution row ─────────────────────────────────────────
    // Try "Apr-25" / "Apr/25" first, then plain "Apr"
    const mwy = line.match(monthWithYearPattern);
    const mp  = mwy ? null : line.match(plainMonthPattern);
    const mMatch = mwy ?? mp;

    if (mMatch) {      // ── Infer correct FY from month+year suffix ───────────────────────
      // e.g. "Apr-25" → FY 2025-26, "Jan-26" → FY 2025-26
      if (mwy) {
        const mStr    = mwy[1].substring(0, 3).toLowerCase();
        const rawYr   = mwy[2];
        const yrSufx  = parseInt(rawYr.length === 2 ? rawYr : rawYr.slice(-2), 10);
        const janMar  = ['jan', 'feb', 'mar'];
        const inferredFY = janMar.includes(mStr)
          ? `20${String((yrSufx - 1 + 100) % 100).padStart(2, '0')}-${String(yrSufx).padStart(2, '0')}`
          : `20${String(yrSufx).padStart(2, '0')}-${String((yrSufx + 1) % 100).padStart(2, '0')}`;

        if (!currentYear) {
          currentYear = { financial_year: inferredFY, opening_balance: 0, contributions: [], interest_earned: 0, closing_balance: 0 };
        } else if (currentYear.financial_year !== inferredFY) {
          if (currentYear.contributions.length === 0) {
            // Wrong FY detected from title, no data yet – just rename it
            currentYear.financial_year = inferredFY;
          } else {
            // Genuine FY switch mid-document
            result.years.push(currentYear);
            currentYear = { financial_year: inferredFY, opening_balance: 0, contributions: [], interest_earned: 0, closing_balance: 0 };
          }
        }
      }
      const monthStr  = mMatch[1].substring(0, 3);
      const fullMonth = monthStr.charAt(0).toUpperCase() + monthStr.slice(1).toLowerCase();

      // afterMonth starts right after the full matched token (including year suffix if present)
      const matchedToken = mMatch[0];
      const startIdx     = line.indexOf(matchedToken) + matchedToken.length;
      const afterMonth   = line.substring(startIdx);

      const numbers = afterMonth.match(/[\d,]+\.?\d*/g) ?? [];

      if (numbers.length >= 2) {
        let empContrib: number;
        let erContrib: number;
        let monthInterest = 0;

        if (numbers.length >= 15) {
          // ── Mphasis / EPFO Annual Statement format (18 columns) ──────────
          // Columns after month: MC(0) VPF(1) TotalMC(2) NonTax(3) Tax(4) CC(5) ...
          // Interest on MRB: MC-NonTax(15) MC-Tax(16) Company(17)
          empContrib    = parseNumber(numbers[0]); // MC
          erContrib     = parseNumber(numbers[5]); // CC
          monthInterest = parseNumber(numbers[15]) + parseNumber(numbers[16]) + parseNumber(numbers[17]);
        } else if (numbers.length >= 3) {
          // Standard EPFO passbook: Wages(0) EmployeeShare(1) EmployerShare(2)
          empContrib = parseNumber(numbers[1]);
          erContrib  = parseNumber(numbers[2]);
        } else {
          // Simple two-column: Employee, Employer
          empContrib = parseNumber(numbers[0]);
          erContrib  = parseNumber(numbers[1]);
        }

        // Sanity check: must be positive and < 1 crore
        if (empContrib > 0 && empContrib < 10_000_000) {
          currentYear.contributions.push({
            month: fullMonth,
            employee_contribution: empContrib,
            employer_contribution: erContrib,
            interest_earned: monthInterest,
          });
        }
      }
    }
  }

  // Push last year
  if (currentYear) result.years.push(currentYear);

  return result;
}
