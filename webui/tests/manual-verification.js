#!/usr/bin/env node

/**
 * Manual verification tests for the WebUI
 * This runs without requiring browser installation
 */

const fs = require('fs');
const path = require('path');
const http = require('http');

const tests = [];
let passCount = 0;
let failCount = 0;

function test(description, fn) {
  tests.push({ description, fn });
}

function pass(message) {
  console.log(`✓ ${message}`);
  passCount++;
}

function fail(message) {
  console.error(`✗ ${message}`);
  failCount++;
}

async function runTests() {
  console.log('\n=== WebUI Verification Tests ===\n');

  // Test 1: Check if dist folder exists
  test('Dist folder exists', () => {
    const distPath = path.join(__dirname, '../dist');
    if (fs.existsSync(distPath)) {
      pass('dist folder exists');
    } else {
      fail('dist folder does not exist');
    }
  });

  // Test 2: Check if index.html exists
  test('index.html exists', () => {
    const indexPath = path.join(__dirname, '../dist/index.html');
    if (fs.existsSync(indexPath)) {
      pass('index.html exists in dist');
    } else {
      fail('index.html does not exist');
    }
  });

  // Test 3: Check if logo.png exists
  test('logo.png exists', () => {
    const logoPath = path.join(__dirname, '../dist/logo.png');
    if (fs.existsSync(logoPath)) {
      const stats = fs.statSync(logoPath);
      if (stats.size > 0) {
        pass(`logo.png exists (${(stats.size / 1024 / 1024).toFixed(2)}MB)`);
      } else {
        fail('logo.png is empty');
      }
    } else {
      fail('logo.png does not exist');
    }
  });

  // Test 4: Check if CSS file exists
  test('CSS bundle exists', () => {
    const cssPath = path.join(__dirname, '../dist/assets');
    if (fs.existsSync(cssPath)) {
      const files = fs.readdirSync(cssPath);
      const cssFile = files.find((f) => f.endsWith('.css'));
      if (cssFile) {
        pass(`CSS bundle exists: ${cssFile}`);
      } else {
        fail('CSS file not found in assets');
      }
    } else {
      fail('assets folder does not exist');
    }
  });

  // Test 5: Check if JS bundle exists
  test('JavaScript bundle exists', () => {
    const jsPath = path.join(__dirname, '../dist/assets');
    if (fs.existsSync(jsPath)) {
      const files = fs.readdirSync(jsPath);
      const jsFile = files.find((f) => f.endsWith('.js'));
      if (jsFile) {
        pass(`JS bundle exists: ${jsFile}`);
      } else {
        fail('JS file not found in assets');
      }
    } else {
      fail('assets folder does not exist');
    }
  });

  // Test 6: Check HTML content
  test('index.html contains required elements', () => {
    const indexPath = path.join(__dirname, '../dist/index.html');
    const content = fs.readFileSync(indexPath, 'utf8');

    const checks = [
      { name: 'DOCTYPE', pattern: /<!DOCTYPE html/ },
      { name: 'app div', pattern: /<div id="app">/ },
      { name: 'meta charset', pattern: /charset="UTF-8"/ },
      { name: 'viewport meta', pattern: /viewport/ },
      { name: 'title', pattern: /<title>super-duper-bassoon<\/title>/ },
      { name: 'main.ts script', pattern: /main\.ts/ },
    ];

    let allPresent = true;
    checks.forEach(({ name, pattern }) => {
      if (!pattern.test(content)) {
        fail(`HTML missing ${name}`);
        allPresent = false;
      }
    });

    if (allPresent) {
      pass('index.html contains all required elements');
    }
  });

  // Test 7: Check TypeScript compilation
  test('TypeScript source files exist', () => {
    const srcPath = path.join(__dirname, '../src');
    if (fs.existsSync(srcPath)) {
      const tsFiles = [];
      const walk = (dir) => {
        fs.readdirSync(dir).forEach((file) => {
          const filePath = path.join(dir, file);
          if (fs.statSync(filePath).isDirectory()) {
            walk(filePath);
          } else if (file.endsWith('.ts')) {
            tsFiles.push(file);
          }
        });
      };
      walk(srcPath);

      if (tsFiles.length > 0) {
        pass(`Found ${tsFiles.length} TypeScript files`);
      } else {
        fail('No TypeScript files found');
      }
    } else {
      fail('src folder does not exist');
    }
  });

  // Test 8: Check directory structure
  test('Required directories exist', () => {
    const requiredDirs = [
      'src',
      'src/api',
      'src/types',
      'src/store',
      'src/components',
      'src/pages',
      'src/utils',
      'src/styles',
      'dist',
      'dist/assets',
    ];

    let allExist = true;
    requiredDirs.forEach((dir) => {
      const dirPath = path.join(__dirname, '..', dir);
      if (!fs.existsSync(dirPath)) {
        fail(`Directory missing: ${dir}`);
        allExist = false;
      }
    });

    if (allExist) {
      pass('All required directories exist');
    }
  });

  // Test 9: Check package.json
  test('package.json is valid', () => {
    const pkgPath = path.join(__dirname, '../package.json');
    try {
      const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
      const requiredScripts = ['dev', 'build', 'test', 'lint'];
      let allPresent = true;
      requiredScripts.forEach((script) => {
        if (!pkg.scripts[script]) {
          fail(`Missing script: ${script}`);
          allPresent = false;
        }
      });
      if (allPresent) {
        pass('package.json contains all required scripts');
      }
    } catch (e) {
      fail(`Failed to parse package.json: ${e.message}`);
    }
  });

  // Test 10: Check HTTP server
  test('Development server is running', (callback) => {
    http
      .get('http://localhost:5173/', (res) => {
        if (res.statusCode === 200) {
          pass('Dev server responding on port 5173');
        } else {
          fail(`Dev server responded with status ${res.statusCode}`);
        }
      })
      .on('error', () => {
        fail('Dev server not accessible on port 5173');
      });
  });

  // Run all tests
  for (const t of tests) {
    try {
      const result = t.fn();
      if (result && typeof result.then === 'function') {
        await result;
      }
    } catch (e) {
      fail(`Error in test "${t.description}": ${e.message}`);
    }
  }

  // Test 11: Verify TypeScript config
  test('TypeScript config is valid', () => {
    const tsconfigPath = path.join(__dirname, '../tsconfig.json');
    try {
      const tsconfig = JSON.parse(fs.readFileSync(tsconfigPath, 'utf8'));
      if (tsconfig.compilerOptions.strict === true) {
        pass('TypeScript strict mode is enabled');
      } else {
        fail('TypeScript strict mode is not enabled');
      }
    } catch (e) {
      fail(`Failed to validate tsconfig.json: ${e.message}`);
    }
  });

  // Test 12: Verify Vite config
  test('Vite config is valid', () => {
    const viteConfigPath = path.join(__dirname, '../vite.config.ts');
    if (fs.existsSync(viteConfigPath)) {
      const content = fs.readFileSync(viteConfigPath, 'utf8');
      const checks = [
        { name: 'root', pattern: /root:\s*['"']src['"']/ },
        { name: 'outDir', pattern: /outDir:\s*['"']\.\.\/dist['"']/ },
        { name: 'alias', pattern: /@:\s*resolve/ },
      ];

      let allPresent = true;
      checks.forEach(({ name, pattern }) => {
        if (!pattern.test(content)) {
          fail(`Vite config missing ${name}`);
          allPresent = false;
        }
      });

      if (allPresent) {
        pass('Vite config is properly configured');
      }
    } else {
      fail('vite.config.ts does not exist');
    }
  });

  // Summary
  console.log(`\n=== Test Summary ===`);
  console.log(`✓ Passed: ${passCount}`);
  console.log(`✗ Failed: ${failCount}`);
  console.log(`Total: ${passCount + failCount}\n`);

  process.exit(failCount > 0 ? 1 : 0);
}

runTests().catch((e) => {
  console.error('Fatal error:', e);
  process.exit(1);
});
