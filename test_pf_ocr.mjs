import { readFileSync, writeFileSync } from 'fs';
import { getDocument } from 'pdfjs-dist/legacy/build/pdf.mjs';
import { createCanvas } from 'canvas';
import Tesseract from 'tesseract.js';

async function run() {
  const data = new Uint8Array(readFileSync('c:/Users/mayur/Downloads/1.pdf'));
  const doc = await getDocument({ data }).promise;

  let fullText = '';

  for (let i = 1; i <= doc.numPages; i++) {
    console.log(`Processing page ${i}/${doc.numPages}...`);
    const page = await doc.getPage(i);
    const viewport = page.getViewport({ scale: 2.0 }); // Higher scale for better OCR

    const canvas = createCanvas(viewport.width, viewport.height);
    const ctx = canvas.getContext('2d');

    await page.render({ canvasContext: ctx, viewport }).promise;

    // Save image for debugging
    const buffer = canvas.toBuffer('image/png');
    writeFileSync(`pf_page_${i}.png`, buffer);
    console.log(`  Saved pf_page_${i}.png (${viewport.width}x${viewport.height})`);

    // Run OCR
    const { data: ocrData } = await Tesseract.recognize(buffer, 'eng');
    console.log(`  OCR confidence: ${ocrData.confidence}%`);
    fullText += `--- PAGE ${i} ---\n${ocrData.text}\n\n`;
  }

  writeFileSync('pf_output.txt', fullText);
  console.log('\n=== FULL OCR TEXT ===');
  console.log(fullText);
}

run().catch(e => console.error(e));
