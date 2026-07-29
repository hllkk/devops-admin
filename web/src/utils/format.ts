/**
 * 文件大小/时间格式化工具
 */

/**
 * Format file size to human-readable string
 *
 * @param bytes - File size in bytes
 * @returns Formatted string like "1.5 MB"
 */
export function formatFileSize(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B';

  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  if (i === 0) {
    return `${bytes} B`;
  }

  const size = bytes / Math.pow(k, i);
  const decimals = i <= 2 ? 1 : 2;
  return `${size.toFixed(decimals)} ${units[i]}`;
}

/**
 * Format date string to "YYYY-MM-DD HH:mm:ss"
 *
 * @param dateStr - ISO date string or date string
 * @returns Formatted string like "2024-01-15 08:30:00"
 */
export function formatDateTime(dateStr: string | null | undefined): string {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return dateStr;
  const y = date.getFullYear();
  const mo = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  const h = String(date.getHours()).padStart(2, '0');
  const mi = String(date.getMinutes()).padStart(2, '0');
  const s = String(date.getSeconds()).padStart(2, '0');
  return `${y}-${mo}-${d} ${h}:${mi}:${s}`;
}
