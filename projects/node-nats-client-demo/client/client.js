import { connect } from 'nats';

/**
 * NATSClient manages connection to NATS server
 */
export class NATSClient {
  /** @type {string} */
  url;
  /** @type {import('nats').NatsConnection | null} */
  nc = null;

  /**
   * @param {string} url - NATS server URL
   */
  constructor(url = 'nats://localhost:4222') {
    this.url = url;
  }

  /**
   * Connect to NATS server
   * @returns {Promise<import('nats').NatsConnection>}
   */
  async connect() {
    this.nc = await connect({ servers: this.url });
    console.log(`Connected to NATS at ${this.url}`);
    return this.nc;
  }

  /**
   * Disconnect from NATS server
   * @returns {Promise<void>}
   */
  async disconnect() {
    if (!this.nc) return;
    await this.nc.drain();
    console.log('Disconnected from NATS');
  }

  /**
   * Get the active NATS connection
   * @returns {import('nats').NatsConnection}
   * @throws {Error} If not connected
   */
  getConnection() {
    if (!this.nc) {
      throw new Error('Not connected. Call connect() first.');
    }
    return this.nc;
  }
}
