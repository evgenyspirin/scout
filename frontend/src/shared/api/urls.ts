import { THUMBNAIL_QUALITY, THUMBNAIL_WIDTHS } from '../constants';

const API = `${import.meta.env.VITE_BACKEND_URL}/api/v1`;

export function thumbnailUrl(id: string, width: number, token: string): string {
  return `${API}/photos/${id}/thumbnail?width=${width}&quality=${THUMBNAIL_QUALITY}&token=${encodeURIComponent(token)}`;
}

export function thumbnailSrcSet(id: string, token: string): string {
  return THUMBNAIL_WIDTHS.map((w) => `${thumbnailUrl(id, w, token)} ${w}w`).join(', ');
}

/** Appends a token query param to an absolute URL (for <img> requests). */
export function withToken(url: string, token: string): string {
  const sep = url.includes('?') ? '&' : '?';
  return `${url}${sep}token=${encodeURIComponent(token)}`;
}
