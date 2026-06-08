import { describe, it, expect } from 'vitest';
import { toRenderedRect, distanceMeters, dominantClass } from './bbox';
import type { BBox, Prediction } from './types';

describe('toRenderedRect', () => {
  it('scales a normalized box to the rendered image size', () => {
    const bbox: BBox = { xMin: 0.1, yMin: 0.2, xMax: 0.6, yMax: 0.8 };
    const rect = toRenderedRect(bbox, 1000, 500);
    expect(rect.left).toBeCloseTo(100);
    expect(rect.top).toBeCloseTo(100);
    expect(rect.width).toBeCloseTo(500);
    expect(rect.height).toBeCloseTo(300);
  });

  it('produces a different result for a different rendered size (DPR/responsive)', () => {
    const bbox: BBox = { xMin: 0, yMin: 0, xMax: 0.5, yMax: 0.5 };
    const small = toRenderedRect(bbox, 320, 180);
    const large = toRenderedRect(bbox, 1280, 720);
    expect(small.width).toBeCloseTo(160);
    expect(large.width).toBeCloseTo(640);
    expect(large.width).toBeGreaterThan(small.width);
  });

  it('handles a full-frame box', () => {
    const rect = toRenderedRect({ xMin: 0, yMin: 0, xMax: 1, yMax: 1 }, 800, 450);
    expect(rect).toEqual({ left: 0, top: 0, width: 800, height: 450 });
  });
});

describe('distanceMeters', () => {
  it('computes euclidean distance', () => {
    expect(distanceMeters(0, 0, 3, 4)).toBeCloseTo(5);
    expect(distanceMeters(10, 10, 10, 10)).toBe(0);
  });
});

describe('dominantClass', () => {
  it('returns the highest-confidence class', () => {
    const preds: Prediction[] = [
      { classId: 'thrips', confidence: 0.4, bbox: { xMin: 0, yMin: 0, xMax: 0.1, yMax: 0.1 } },
      { classId: 'mirid', confidence: 0.9, bbox: { xMin: 0, yMin: 0, xMax: 0.1, yMax: 0.1 } },
    ];
    expect(dominantClass(preds)).toBe('mirid');
  });

  it('returns null for no predictions', () => {
    expect(dominantClass([])).toBeNull();
  });
});
