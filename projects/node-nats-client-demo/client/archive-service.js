/**
 * ArchiveService publishes to QueueGroupService for file archiving
 */
export class ArchiveService {
  /** @type {import('nats').NatsConnection} */
  nc;
  /** @type {string} */
  subject = 'services.fileops.archive';

  /**
   * @param {import('nats').NatsConnection} nc - NATS connection
   */
  constructor(nc) {
    this.nc = nc;
  }

  /**
   * Archive a file by its ID (fire-and-forget)
   * @param {string} fileId - File ID to archive
   * @returns {void}
   */
  archiveFile(fileId) {
    const request = { file_id: fileId };
    this.nc.publish(this.subject, JSON.stringify(request));
    console.log(`Archive request queued for file: ${fileId}`);
  }

  /**
   * Archive multiple files
   * @param {string[]} fileIds - Array of file IDs
   * @returns {number} Number of files queued
   */
  archiveMultiple(fileIds) {
    for (const fileId of fileIds) {
      this.archiveFile(fileId);
    }
    return fileIds.length;
  }
}
