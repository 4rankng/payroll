import { readFileSync, writeFileSync } from 'fs';
import { execSync } from 'child_process';

const root = '/Users/dev/Documents/projects/payroll/frontend/src';

// Get all files
const files = execSync(
  `grep -rln --include='*.ts' --include='*.tsx' -E 'toLocaleDateString|"d/M/yyyy"' ${root}`,
  { encoding: 'utf-8' }
).trim().split('\n')
  .filter(f => f && !f.includes('LoansPage/') && !f.includes('.test.') && !f.includes('.spec.'));

let totalReplacements = 0;
let filesModified = 0;
const filesNeedingAttention = [];

for (const file of files) {
  let content = readFileSync(file, 'utf-8');
  let original = content;
  let replacements = 0;
  const hadFormatImport = /import\s+.*\bformat\b.*from\s+['"]date-fns['"]/.test(content);
  const hadViImport = /import\s+.*\bvi\b.*from\s+['"]date-fns\/locale['"]/.test(content);

  // Step 1: Replace multiline .toLocaleDateString('vi-VN', { ... }) calls
  // Match: .toLocaleDateString('vi-VN' or "vi-VN", followed by optional { ... } spanning multiple lines
  content = content.replace(
    /\.toLocaleDateString\(\s*['"]vi-VN['"]\s*,\s*\{[^}]*\}\s*\)/g,
    (match) => {
      replacements++;
      return match.replace(
        /\.toLocaleDateString\(\s*['"]vi-VN['"]\s*,\s*\{[^}]*\}\s*\)/,
        ''
      );
    }
  );
  
  // Actually, let me do it differently. Replace the whole .toLocaleDateString(...) call
  // The call is always on some expression like `date.toLocaleDateString(...)` or `new Date(x).toLocaleDateString(...)`
  // We need to keep the expression before .toLocaleDateString and wrap it.
  
  // Let me restart with a cleaner approach
  content = original;

  // Replace multiline .toLocaleDateString("vi-VN", { ... }) with format wrapper
  // Pattern: expression.toLocaleDateString("vi-VN", { multi-line options })
  content = content.replace(
    /\.toLocaleDateString\(\s*["']vi-VN["']\s*,\s*\{[\s\S]*?\}\s*\)/g,
    () => {
      replacements++;
      // We can't easily extract the receiver here, so we'll do it in a second pass
      return '†FORMAT_DATE†';
    }
  );
  
  // Replace simple .toLocaleDateString("vi-VN") or .toLocaleDateString('vi-VN')
  content = content.replace(
    /\.toLocaleDateString\(\s*["']vi-VN["']\s*\)/g,
    () => {
      replacements++;
      return '†FORMAT_DATE†';
    }
  );
  
  // Replace format(date, "d/M/yyyy", { locale: vi }) → format(date, 'dd/MM/yyyy')
  content = content.replace(
    /format\(([^,]+),\s*["']d\/M\/yyyy["'],\s*\{[^}]*\}\s*\)/g,
    (_, dateExpr) => {
      replacements++;
      return `format(${dateExpr}, 'dd/MM/yyyy')`;
    }
  );
  
  // Now replace †FORMAT_DATE† with format() wrapper
  // The †FORMAT_DATE† appears after an expression like: new Date(x) or date
  // We need to find the expression before it. Let's use a different strategy:
  // Instead of marker, let me just replace directly.
  
  content = original;
  replacements = 0;
  
  // Strategy: replace all .toLocaleDateString('vi-VN'...) calls with format() equivalents
  // We need to capture the receiver (the expression before .toLocaleDateString)
  
  // Multiline version with options
  content = content.replace(
    /(\w[\w.]*\([^)]*\)|\w+(?:\.\w+)*)\.toLocaleDateString\(\s*["']vi-VN["']\s*,\s*\{[\s\S]*?\}\s*\)/g,
    (fullMatch, receiver) => {
      replacements++;
      return `format(${receiver}, 'dd/MM/yyyy')`;
    }
  );
  
  // Simple version without options
  content = content.replace(
    /(\w[\w.]*\([^)]*\)|\w+(?:\.\w+)*)\.toLocaleDateString\(\s*["']vi-VN["']\s*\)/g,
    (fullMatch, receiver) => {
      replacements++;
      return `format(${receiver}, 'dd/MM/yyyy')`;
    }
  );
  
  // format(date, "d/M/yyyy", { locale: vi })
  content = content.replace(
    /format\(([^,]+),\s*["']d\/M\/yyyy["'],\s*\{[^}]*\}\s*\)/g,
    (_, dateExpr) => {
      replacements++;
      return `format(${dateExpr.trim()}, 'dd/MM/yyyy')`;
    }
  );
  
  if (content !== original) {
    // Step 2: Add format import if needed
    if (!hadFormatImport && replacements > 0) {
      // Find first import to add before it, or add at top
      const importMatch = content.match(/^import\s/m);
      if (importMatch) {
        const idx = content.indexOf(importMatch[0]);
        content = content.slice(0, idx) + `import { format } from "date-fns";\n` + content.slice(idx);
      } else {
        content = `import { format } from "date-fns";\n` + content;
      }
    } else if (hadFormatImport && !/\bformat\b/.test(content.match(/import\s+{([^}]*)}\s+from\s+['"]date-fns['"]/)?.[1] || '')) {
      // format was already imported but maybe was removed - check
      // Actually if hadFormatImport, format was already there. Skip.
    }
    
    // Step 3: Remove unused vi import if it was only used with the old format
    // Check if 'vi' is still referenced in the file (besides the import)
    if (hadViImport) {
      const viImportLine = content.match(/import\s+\{[^}]*\bvi\b[^}]*\}\s+from\s+['"]date-fns\/locale['"];?\n?/);
      if (viImportLine) {
        // Check if vi is used elsewhere in the file (not in the import line)
        const contentWithoutImport = content.replace(viImportLine[0], '');
        if (!/\bvi\b/.test(contentWithoutImport)) {
          // vi is not used, remove the import
          content = contentWithoutImport;
        } else if (viImportLine[0].includes(',')) {
          // vi is imported with other things, just remove vi from the import
          const newImport = viImportLine[0].replace(/,\s*vi\b/, '').replace(/\bvi\s*,\s*/, '').replace(/\bvi\b/, '');
          if (newImport.match(/import\s+\{\s*\}\s+from/)) {
            // Empty import, remove entirely
            content = content.replace(viImportLine[0], '');
          } else {
            content = content.replace(viImportLine[0], newImport);
          }
        }
      }
    }
    
    if (content !== original) {
      writeFileSync(file, content, 'utf-8');
      filesModified++;
      totalReplacements += replacements;
      console.log(`✅ ${file}: ${replacements} replacements`);
    }
  }
}

console.log(`\n=== Summary ===`);
console.log(`Files modified: ${filesModified}`);
console.log(`Total replacements: ${totalReplacements}`);
