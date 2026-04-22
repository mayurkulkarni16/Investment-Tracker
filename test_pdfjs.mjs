import { readFileSync, writeFileSync } from 'fs';
import { getDocument } from 'pdfjs-dist/legacy/build/pdf.mjs';

const data = new Uint8Array(readFileSync('C:\\Users\\mayur\\Downloads\\Transactions_1776605217453.pdf'));
const pdf = await getDocument({ data, password: 'ELBPK1560F', useWorkerFetch: false, isEvalSupported: false, useSystemFonts: true }).promise;

const textParts = [];
for (let i = 1; i <= pdf.numPages; i++) {
  const page = await pdf.getPage(i);
  const content = await page.getTextContent();
  let lastY = null;
  for (const item of content.items) {
    if (!('str' in item)) continue;
    const y = item.transform[5];
    if (lastY !== null && Math.abs(y - lastY) > 5) {
      textParts.push('\n');
    } else if (lastY !== null) {
      textParts.push(' ');
    }
    textParts.push(item.str);
    lastY = y;
  }
  textParts.push('\n');
}

const text = textParts.join('');
writeFileSync('cas_pdfjs_output.txt', text);
console.log('Length:', text.length);
console.log('--- FIRST 3000 chars ---');
console.log(text.substring(0, 3000));
