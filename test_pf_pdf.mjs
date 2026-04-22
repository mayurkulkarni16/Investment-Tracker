import { readFileSync, writeFileSync } from 'fs';
import { getDocument } from 'pdfjs-dist/legacy/build/pdf.mjs';

async function run() {
  const data = new Uint8Array(readFileSync('c:/Users/mayur/Downloads/1.pdf'));
  const doc = await getDocument({ data }).promise;
  let text = '';
  for (let i = 1; i <= doc.numPages; i++) {
    const page = await doc.getPage(i);
    const content = await page.getTextContent();
    let lastY = null;
    for (const item of content.items) {
      if ('str' in item) {
        if (lastY !== null && Math.abs(item.transform[5] - lastY) > 5) text += '\n';
        else if (lastY !== null) text += '  ';
        text += item.str;
        lastY = item.transform[5];
      }
    }
    text += '\n---PAGE---\n';
  }
  writeFileSync('pf_output.txt', text);
  console.log(text.substring(0, 6000));
  console.log('\n\n=== TOTAL LENGTH:', text.length, '===');
}

run().catch(e => console.error(e));
