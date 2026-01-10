const SERVICE_UNAVAILABLE_CODE = '503';

/**
 * @typedef {Object} SaveFileResult
 * @property {string} file_id - Unique file identifier
 * @property {string} filename - Name of the saved file
 * @property {number} size - File size in bytes
 * @property {string} digest - File content digest
 */

/**
 * FileService wraps RequestReplyService for file.save operations
 */
export class FileService {
  /** @type {import('nats').NatsConnection} */
  nc;
  /** @type {string} */
  subject = 'services.fileops.save';

  /**
   * @param {import('nats').NatsConnection} nc - NATS connection
   */
  constructor(nc) {
    this.nc = nc;
  }

  /**
   * Save a JSON file to the bucket
   * @param {string} filename - Name of the file
   * @param {object} content - JSON content to save
   * @param {number} timeout - Request timeout in milliseconds
   * @returns {Promise<SaveFileResult>}
   */
  async saveFile(filename, content, timeout = 5000) {
    const request = { filename, content };
    return this.request(request, timeout);
  }

  /**
   * Convenience method to save user settings
   * @param {string} userId - User ID
   * @param {object} settings - Settings object
   * @returns {Promise<SaveFileResult>}
   */
  async saveUserSettings(userId, settings) {
    const filename = `user-${userId}-settings.json`;
    return this.saveFile(filename, settings);
  }

  /**
   * Send request to the file service
   * @param {object} request - Request payload
   * @param {number} timeout - Request timeout in milliseconds
   * @returns {Promise<SaveFileResult>}
   * @private
   */
  async request(request, timeout) {
    let response;
    try {
      response = await this.nc.request(
        this.subject,
        JSON.stringify(request),
        { timeout }
      );
    } catch (error) {
      if (error.code === SERVICE_UNAVAILABLE_CODE) {
        throw new Error('Service unavailable. Make sure Go server is running.');
      }
      throw error;
    }

    const result = JSON.parse(new TextDecoder().decode(response.data));

    if (result.error) {
      throw new Error(result.error);
    }

    return result;
  }
}
