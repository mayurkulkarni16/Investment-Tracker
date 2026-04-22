const fs = require('fs');
const { PDFParse } = require('pdf-parse');

const pdfPath = process.argv[2] || 'C:\\Users\\mayur\\Downloads\\Transactions_1776605217453.pdf';
const password = process.argv[3] || 'ELBPK1560F';

async function main() {
  const data = fs.readFileSync(pdfPath);
  const parser = new PDFParse({ data, password, verbosity: 0 });
  const result = await parser.getText();
  fs.writeFileSync('cas_output.txt', result.text);
  console.log('Pages:', result.total);
  console.log('Text length:', result.text.length);
  console.log('--- CONTENT START ---');
  console.log(result.text.substring(0, 8000));
  console.log('--- CONTENT END ---');
}

main().catch(err => console.error('Error:', err.message, err.stack));
