#!/usr/bin/env node

/**
 * Automatically bumps the patch version in the project.
 * Updates the root VERSION file and syncs all package.json files.
 * Called by lefthook on pre-commit when source files change.
 */

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const VERSION_FILE = path.join(ROOT, 'VERSION');
const PACKAGES = [
  path.join(ROOT, 'package.json'),
  path.join(ROOT, 'apps', 'web', 'package.json'),
  path.join(ROOT, 'apps', 'api', 'package.json'),
];

function bumpVersion(version) {
  const [major, minor, patch] = version.split('.').map(Number);
  
  if (isNaN(major) || isNaN(minor) || isNaN(patch)) {
    throw new Error(`Invalid version format: ${version}`);
  }
  
  // Bump patch version
  const newVersion = `${major}.${minor}.${patch + 1}`;
  return newVersion;
}

function updateVersionFile() {
  if (!fs.existsSync(VERSION_FILE)) {
    console.log('Creating VERSION file with initial version 0.0.1');
    fs.writeFileSync(VERSION_FILE, '0.0.1\n');
    return '0.0.1';
  }
  
  const content = fs.readFileSync(VERSION_FILE, 'utf8').trim();
  const newVersion = bumpVersion(content);
  
  fs.writeFileSync(VERSION_FILE, newVersion + '\n');
  console.log(`Bumped VERSION: ${content} → ${newVersion}`);
  return newVersion;
}

function syncPackage(pkgPath, version) {
  if (!fs.existsSync(pkgPath)) {
    console.log(`Skipping ${pkgPath} - file not found`);
    return;
  }
  
  const content = fs.readFileSync(pkgPath, 'utf8');
  const pkg = JSON.parse(content);
  
  const oldVersion = pkg.version;
  pkg.version = version;
  
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
  console.log(`Synced ${path.basename(path.dirname(pkgPath))}: ${oldVersion} → ${version}`);
}

// Main
const newVersion = updateVersionFile();
PACKAGES.forEach(pkgPath => syncPackage(pkgPath, newVersion));

console.log(`\n✅ Version bumped to ${newVersion}`);
