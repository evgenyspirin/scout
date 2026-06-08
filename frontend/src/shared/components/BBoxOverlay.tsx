import type { Prediction } from '../types';
import { toRenderedRect } from '../bbox';
import { classColor, className } from '../constants';
import styles from './BBoxOverlay.module.css';

interface Props {
  predictions: Prediction[];
  renderedWidth: number;
  renderedHeight: number;
  showLabels?: boolean;
}

// Classes whose bright fill needs dark label text for legibility.
const DARK_LABEL_CLASSES = new Set(['whitefly_aphid']);

/**
 * Draws detection boxes positioned in pixels relative to the actual rendered
 * image size. This presentational component is intentionally size-driven so it
 * can be unit-tested without browser layout.
 */
export function BBoxOverlay({ predictions, renderedWidth, renderedHeight, showLabels = true }: Props) {
  if (renderedWidth === 0 || renderedHeight === 0) return null;

  return (
    <div className={styles.layer} data-testid="bbox-layer">
      {predictions.map((p, i) => {
        const rect = toRenderedRect(p.bbox, renderedWidth, renderedHeight);
        const color = classColor(p.classId);
        const textColor = DARK_LABEL_CLASSES.has(p.classId) ? '#111' : '#fff';
        return (
          <div
            key={`${p.classId}-${i}`}
            data-testid={`bbox-${p.classId}`}
            className={styles.box}
            style={{
              left: `${rect.left}px`,
              top: `${rect.top}px`,
              width: `${rect.width}px`,
              height: `${rect.height}px`,
              borderColor: color,
            }}
          >
            {showLabels && (
              <span className={styles.label} style={{ background: color, color: textColor }}>
                {className(p.classId)} · {(p.confidence * 100).toFixed(0)}%
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}
