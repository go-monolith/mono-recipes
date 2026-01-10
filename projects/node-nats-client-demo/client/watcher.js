/**
 * @typedef {Object} WatchEvent
 * @property {string} bucket - Bucket name
 * @property {string} name - File name
 * @property {boolean} deleted - Whether the file was deleted
 * @property {number} size - File size in bytes
 * @property {Date} timestamp - Event timestamp
 */

/**
 * @typedef {Object} WatchOptions
 * @property {boolean} [includeHistory] - Include historical entries
 * @property {boolean} [ignoreDeletes] - Ignore delete events
 */

/**
 * @callback WatchCallback
 * @param {WatchEvent} event - The watch event
 * @returns {Promise<void>}
 */

/**
 * BucketWatcher monitors JetStream object store for file changes
 */
export class BucketWatcher {
  /** @type {import('nats').NatsConnection} */
  nc;
  /** @type {string} */
  bucketName;
  /** @type {import('nats').ObjectStore | null} */
  os = null;
  /** @type {import('nats').ObjectStoreStatus | null} */
  watchSub = null;

  /**
   * @param {import('nats').NatsConnection} nc - NATS connection
   * @param {string} bucketName - Name of the object store bucket
   */
  constructor(nc, bucketName = 'user-settings') {
    this.nc = nc;
    this.bucketName = bucketName;
  }

  /**
   * Initialize the object store connection
   * @returns {Promise<void>}
   */
  async initialize() {
    const js = this.nc.jetstream();
    this.os = await js.views.os(this.bucketName);
    console.log(`Initialized object store: ${this.bucketName}`);
  }

  /**
   * Watch the bucket for changes
   * @param {WatchCallback} callback - Function to call for each change event
   * @param {WatchOptions} options - Watch options
   * @returns {Promise<void>}
   */
  async watch(callback, options = {}) {
    if (!this.os) {
      throw new Error('Watcher not initialized. Call initialize() first.');
    }

    this.watchSub = await this.os.watch({
      includeHistory: options.includeHistory ?? false,
      ignoreDeletes: options.ignoreDeletes ?? false,
    });

    console.log(`Watching bucket: ${this.bucketName}`);

    this.processEvents(callback);
  }

  /**
   * Process watch events in the background
   * @param {WatchCallback} callback - Function to call for each change event
   * @private
   */
  async processEvents(callback) {
    for await (const entry of this.watchSub) {
      if (!entry) {
        console.log('Initial watch updates complete');
        continue;
      }

      const event = {
        bucket: entry.bucket,
        name: entry.name,
        deleted: entry.deleted ?? false,
        size: entry.size,
        timestamp: new Date(),
      };
      await callback(event);
    }
  }

  /**
   * Stop watching
   * @returns {Promise<void>}
   */
  async stop() {
    if (!this.watchSub) return;
    await this.watchSub.stop();
    console.log('Watcher stopped');
  }
}
