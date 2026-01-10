import { NATSClient, FileService, ArchiveService, BucketWatcher } from './client/index.js';

// ANSI color codes
const RESET = '\x1b[0m';
const BRIGHT = '\x1b[1m';
const RED = '\x1b[31m';
const GREEN = '\x1b[32m';
const YELLOW = '\x1b[33m';
const BLUE = '\x1b[34m';
const MAGENTA = '\x1b[35m';
const CYAN = '\x1b[36m';

/**
 * Log a message with optional color
 * @param {string} message - Message to log
 * @param {string} color - ANSI color code
 */
function log(message, color = RESET) {
  console.log(`${color}${message}${RESET}`);
}

/**
 * Print a section header
 * @param {string} title - Header title
 */
function header(title) {
  console.log('');
  log('='.repeat(60), CYAN);
  log(title, BRIGHT + CYAN);
  log('='.repeat(60), CYAN);
}

/**
 * Wait for a specified number of milliseconds
 * @param {number} ms - Milliseconds to wait
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Handle watch events for the bucket watcher
 * @param {import('./client/watcher.js').WatchEvent} event - Watch event
 */
function handleWatchEvent(event) {
  if (event.deleted) {
    log(`  [trash] File deleted: ${event.name}`, YELLOW);
    return;
  }

  const extension = event.name.split('.').pop();
  const icon = extension === 'zip' ? '[archive]' : '[file]';
  const action = event.size > 0 ? 'created' : 'updated';
  log(`  ${icon} File ${action}: ${event.name} (${event.size} bytes)`, BLUE);
}

/**
 * Run the demo workflow
 * @param {string} natsUrl - NATS server URL
 */
async function runDemo(natsUrl) {
  const client = new NATSClient(natsUrl);
  let watcher;

  try {
    await client.connect();

    const nc = client.getConnection();
    const fileService = new FileService(nc);
    const archiveService = new ArchiveService(nc);
    const savedFiles = [];

    // Step 1: Start bucket watcher
    header('Step 1: Starting Bucket Watcher');
    watcher = new BucketWatcher(nc, 'user-settings');
    await watcher.initialize();
    await watcher.watch(handleWatchEvent);
    await sleep(1000);

    // Step 2: Save files
    header('Step 2: Saving User Settings Files');

    const users = [
      { id: '123', settings: { theme: 'dark', language: 'en', fontSize: 14 } },
      { id: '456', settings: { theme: 'light', language: 'es', fontSize: 16 } },
      { id: '789', settings: { theme: 'dark', language: 'fr', fontSize: 12 } },
    ];

    for (const user of users) {
      const result = await fileService.saveUserSettings(user.id, user.settings);
      savedFiles.push(result);
      log(`  [ok] Saved: ${result.filename}`, GREEN);
      log(`    File ID: ${result.file_id}`, CYAN);
      log(`    Size: ${result.size} bytes`, CYAN);
      log(`    Digest: ${result.digest.substring(0, 16)}...`, CYAN);
    }

    await sleep(2000);

    // Step 3: Archive files
    header('Step 3: Archiving Files');

    for (const file of savedFiles) {
      archiveService.archiveFile(file.file_id);
      log(`  [archive] Queued archive: ${file.filename}`, MAGENTA);
    }

    log('\n  Waiting for archive processing...', YELLOW);
    await sleep(5000);

    // Step 4: Summary
    header('Summary');
    log(`  Files saved: ${savedFiles.length}`, GREEN);
    log(`  Archive requests: ${savedFiles.length}`, MAGENTA);
    log(`  Watch events: Check output above`, BLUE);
    log('\n[ok] Demo completed successfully!\n', BRIGHT + GREEN);
  } finally {
    if (watcher) {
      await watcher.stop();
    }
    await client.disconnect();
  }
}

async function main() {
  const natsUrl = process.argv[2] || 'nats://localhost:4222';

  log('\n[rocket] Node.js NATS Client Demo', BRIGHT + GREEN);
  log(`Connecting to: ${natsUrl}\n`, YELLOW);

  try {
    await runDemo(natsUrl);
  } catch (error) {
    log(`\n[error] Error: ${error.message}`, BRIGHT + RED);
    if (error.message.includes('unavailable')) {
      log('Make sure the Go server is running: go run .', YELLOW);
    }
    process.exit(1);
  }
}

process.on('SIGINT', () => {
  log('\n\n[wave] Shutting down gracefully...', YELLOW);
  process.exit(0);
});

main();
