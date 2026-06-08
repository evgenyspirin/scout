import type { BBox, Prediction } from './types';

export interface RenderedRect {
  left: number;
  top: number;
  width: number;
  height: number;
}

/**
 * Maps a normalized [0,1] bounding box to pixel coordinates relative to the
 * ACTUAL rendered image size in the browser (not the original image size).
 * The same transform is used for gallery thumbnails and the full-size view.
 */
export function toRenderedRect(
  bbox: BBox,
  renderedWidth: number,
  renderedHeight: number,
): RenderedRect {
  return {
    left: bbox.xMin * renderedWidth,
    top: bbox.yMin * renderedHeight,
    width: (bbox.xMax - bbox.xMin) * renderedWidth,
    height: (bbox.yMax - bbox.yMin) * renderedHeight,
  };
}

/** Euclidean distance in meters between two greenhouse positions. */
export function distanceMeters(ax: number, ay: number, bx: number, by: number): number {
  return Math.hypot(ax - bx, ay - by);
}

/** Returns the highest-confidence prediction's class, or null when none. */
export function dominantClass(predictions: Prediction[]): string | null {
  if (predictions.length === 0) return null;
  return predictions.reduce((best, p) => (p.confidence > best.confidence ? p : best)).classId;
}
