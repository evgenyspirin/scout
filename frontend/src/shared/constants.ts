import type { DetectionClass } from './types';

// The six detection classes with distinct, high-visibility colors.
export const DETECTION_CLASSES: readonly DetectionClass[] = [
  { id: 'powdery_mildew', name: 'Powdery Mildew', color: '#06B6D4' },
  { id: 'mirid', name: 'Mirid', color: '#F97316' },
  { id: 'whitefly_aphid', name: 'Whitefly / Aphid', color: '#EAB308' },
  { id: 'miner_tuta', name: 'Miner Tuta', color: '#EC4899' },
  { id: 'thrips', name: 'Thrips', color: '#3B82F6' },
  { id: 'spider_mites', name: 'Spider Mites', color: '#EF4444' },
];

const FALLBACK_CLASS_COLOR = '#166534';

export function classColor(classId: string): string {
  return DETECTION_CLASSES.find((c) => c.id === classId)?.color ?? FALLBACK_CLASS_COLOR;
}

export function className(classId: string): string {
  return DETECTION_CLASSES.find((c) => c.id === classId)?.name ?? classId;
}

// Responsive thumbnail widths supported by the backend (used in srcset).
export const THUMBNAIL_WIDTHS = [320, 640, 960, 1280] as const;
export const DEFAULT_THUMBNAIL_WIDTH = 640;
export const THUMBNAIL_QUALITY = 80;

// Greenhouse plane dimensions (meters) and the "nearby" click radius.
export const GREENHOUSE_SIZE_M = 40;
export const NEARBY_RADIUS_M = 2;

// Confidence slider configuration.
export const CONFIDENCE_MIN = 0;
export const CONFIDENCE_MAX = 1;
export const CONFIDENCE_STEP = 0.05;
export const DEFAULT_MIN_CONFIDENCE = 0;

// Page size requested for the gallery + map (backend also supports cursor paging).
export const PAGE_LIMIT = 200;
