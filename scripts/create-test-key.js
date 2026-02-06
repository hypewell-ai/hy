#!/usr/bin/env node
/**
 * Create a test API key for integration testing.
 * 
 * Usage:
 *   node scripts/create-test-key.js [workspace-id]
 * 
 * Requires:
 *   - GOOGLE_APPLICATION_CREDENTIALS env var set, OR
 *   - Running with `firebase` CLI context (gcloud auth)
 */

const { initializeApp, cert, applicationDefault } = require('firebase-admin/app');
const { getFirestore } = require('firebase-admin/firestore');
const crypto = require('crypto');
const path = require('path');
const fs = require('fs');

const PROJECT_ID = 'hypewell-prod';
const DEFAULT_WORKSPACE = 'ws_integration_test';

// Try service account key first, fall back to application default
let credential;
const keyPath = process.env.GOOGLE_APPLICATION_CREDENTIALS || 
  path.join(process.env.HOME, '.config/gcloud/hypewell-studio-key.json');

if (fs.existsSync(keyPath)) {
  credential = cert(keyPath);
  console.log(`Using service account: ${keyPath}\n`);
} else {
  credential = applicationDefault();
  console.log('Using application default credentials\n');
}

// Initialize Firebase Admin
initializeApp({
  credential: credential,
  projectId: PROJECT_ID,
});

const db = getFirestore();

function generateApiKey() {
  const bytes = crypto.randomBytes(24);
  return 'sk_live_' + bytes.toString('base64url');
}

function hashKey(key) {
  return crypto.createHash('sha256').update(key).digest('hex');
}

function generateKeyId() {
  const bytes = crypto.randomBytes(8);
  return 'key_' + bytes.toString('base64url').replace(/[_-]/g, '').slice(0, 12);
}

async function createApiKey(workspaceId) {
  const rawKey = generateApiKey();
  const keyId = generateKeyId();
  const keyHash = hashKey(rawKey);
  const keyPrefix = rawKey.slice(0, 12);
  const now = new Date();

  const keyDoc = {
    id: keyId,
    workspaceId: workspaceId,
    name: 'Integration Tests',
    keyHash: keyHash,
    keyPrefix: keyPrefix,
    scopes: [
      'productions:read',
      'productions:write',
      'assets:read',
      'assets:write',
      'thread:read',
      'thread:write',
      'workspace:read',
    ],
    createdAt: now,
    createdBy: 'bootstrap-script',
    lastUsedAt: null,
    expiresAt: null,
    revokedAt: null,
  };

  await db
    .collection('workspaces')
    .doc(workspaceId)
    .collection('apiKeys')
    .doc(keyId)
    .set(keyDoc);

  return { keyId, rawKey, keyPrefix };
}

async function main() {
  const workspaceId = process.argv[2] || DEFAULT_WORKSPACE;

  console.log(`Creating API key for workspace: ${workspaceId}\n`);

  try {
    const { keyId, rawKey, keyPrefix } = await createApiKey(workspaceId);

    console.log('✓ API key created\n');
    console.log(`ID:     ${keyId}`);
    console.log(`Prefix: ${keyPrefix}...`);
    console.log(`Key:    ${rawKey}\n`);
    console.log('⚠️  Save this key now. You won\'t be able to see it again.\n');
    console.log('To save for integration tests:');
    console.log(`  echo "${rawKey}" > ~/.config/hy/test-key\n`);
    console.log('To run integration tests:');
    console.log('  HY_TEST_API_KEY=$(cat ~/.config/hy/test-key) go test ./integration/... -v -tags=integration');

    process.exit(0);
  } catch (error) {
    console.error('Error creating API key:', error.message);
    process.exit(1);
  }
}

main();
