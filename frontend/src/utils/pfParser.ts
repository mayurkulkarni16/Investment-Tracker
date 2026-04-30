import * as pdfjsLib from 'pdfjs-dist';
import pdfjsWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url';
import Tesseract from 'tesseract.js';

pdfjsLib.GlobalWorkerOptions.workerSrc = pdfjsWorker;

export interface ParsedPFContribution {
  month: string;
  employee_contribution: number;
  employer_contribution: number;
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

  for (let i = 1; i <= doc.numPages; i++) {
    onProgress?.(`Rendering page ${i}/${doc.numPages}...`);

    const page = await doc.getPage(i);
    const viewport = page.getViewport({ scale: 3.0 });

    const canvas = document.createElement('canvas');
    canvas.width = viewport.width;
    canvas.height = viewport.height;
    const ctx = canvas.getContext('2d')!;

    // Fill white background first
    ctx.fillStyle = '#ffffff';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    await page.render({ canvas, canvasContext: ctx, viewport } as any).promise;

    // Debug: check if canvas has content
    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    let nonWhitePixels = 0;
    for (let p = 0; p < imageData.data.length; p += 4) {
      if (imageData.data[p] < 250 || imageData.data[p+1] < 250 || imageData.data[p+2] < 250) {
        nonWhitePixels++;
      }
    }
    console.log(`Page ${i}: ${canvas.width}x${canvas.height}, non-white pixels: ${nonWhitePixels}`);

    if (nonWhitePixels < 100) {
      onProgress?.(`Page ${i} appears blank after rendering, skipping OCR...`);
      canvas.remove();
      continue;
    }

    onProgress?.(`OCR processing page ${i}/${doc.numPages}... (this may take 30-60s)`);

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

  // Try to extract account number (UAN or member ID patterns)
  for (const line of lines) {
    // UAN pattern: 10+ digit number
    const uanMatch = line.match(/(?:UAN|Member\s*Id)[:\s]*(\d{10,})/i);
    if (uanMatch) {
      result.account_number = uanMatch[1];
    }
    // Employer/establishment name
    const empMatch = line.match(/(?:Establishment|Employer|Company)[:\s]*(.+)/i);
    if (empMatch) {
      result.employer_name = empMatch[1].trim();
    }
  }

  // EPF passbook format typically has rows with:
  // Month | Wages | Employee Share | Employer Share | Pension | ...
  // Or: Date | Particulars | Employee Share | Employer Share | ...
  // Try to find table rows with monthly contribution data



  const monthPattern = /\b(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:tember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\b/i;

  // Look for financial year headers like "2024-25" or "FY 2024-25"
  const fyPattern = /(?:FY|Financial\s*Year)?[\s:]*(\d{4})[-–](\d{2,4})/i;

  let currentYear: ParsedPFYear | null = null;

  for (const line of lines) {
    // Check for financial year header
    const fyMatch = line.match(fyPattern);
    if (fyMatch && !monthPattern.test(line.substring(0, line.indexOf(fyMatch[0])))) {
      const startYear = fyMatch[1];
      const endYear = fyMatch[2].length === 2 ? startYear.substring(0, 2) + fyMatch[2] : fyMatch[2];
      const fy = `${startYear}-${endYear.slice(-2)}`;

      if (currentYear) {
        result.years.push(currentYear);
      }
      currentYear = {
        financial_year: fy,
        opening_balance: 0,
        contributions: [],
        interest_earned: 0,
        closing_balance: 0,
      };
    }

    // Check for opening balance
    const obMatch = line.match(/opening\s*balance[:\s]*([\d,]+\.?\d*)/i);
    if (obMatch && currentYear) {
      currentYear.opening_balance = parseNumber(obMatch[1]);
    }

    // Check for closing balance
    const cbMatch = line.match(/closing\s*balance[:\s]*([\d,]+\.?\d*)/i);
    if (cbMatch && currentYear) {
      currentYear.closing_balance = parseNumber(cbMatch[1]);
    }

    // Check for interest
    const intMatch = line.match(/interest[:\s]*([\d,]+\.?\d*)/i);
    if (intMatch && currentYear) {
      currentYear.interest_earned = parseNumber(intMatch[1]);
    }

    // Try to match monthly contribution rows
    // Common formats:
    // "Apr 2024  15000  1800  1800  ..."
    // "04/2024  15000  1800  1800  ..."
    // Match: month followed by multiple numbers
    const mMatch = line.match(monthPattern);
    if (mMatch && currentYear) {
      const monthStr = mMatch[1].substring(0, 3).toLowerCase();
      const fullMonth = monthStr.charAt(0).toUpperCase() + monthStr.slice(1);

      // Extract all numbers from the line after the month
      const afterMonth = line.substring(line.indexOf(mMatch[0]) + mMatch[0].length);
      const numbers = afterMonth.match(/[\d,]+\.?\d*/g);

      if (numbers && numbers.length >= 2) {
        // Typical EPF format: Wages, Employee Share, Employer Share, ...
        // Or just: Employee Share, Employer Share
        let empContrib: number;
        let erContrib: number;

        if (numbers.length >= 3) {
          // Wages, Employee, Employer (skip wages)
          empContrib = parseNumber(numbers[1]);
          erContrib = parseNumber(numbers[2]);
        } else {
          // Employee, Employer directly
          empContrib = parseNumber(numbers[0]);
          erContrib = parseNumber(numbers[1]);
        }

        // Sanity check - contributions should be reasonable (< 1 crore)
        if (empContrib > 0 && empContrib < 10000000) {
          currentYear.contributions.push({
            month: fullMonth,
            employee_contribution: empContrib,
            employer_contribution: erContrib,
          });
        }
      }
    }
  }

  // Push last year
  if (currentYear) {
    result.years.push(currentYear);
  }

  return result;
}
