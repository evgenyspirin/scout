import { useRef, useState } from 'react';
import type { Prediction } from '../types';
import { cx } from '../cx';
import { useRenderedSize } from '../hooks/useRenderedSize';
import { BBoxOverlay } from './BBoxOverlay';
import styles from './ImageWithBoxes.module.css';

interface Props {
  src: string;
  srcSet?: string;
  sizes?: string;
  alt: string;
  predictions: Prediction[];
  className?: string;
  imgClassName?: string;
  showLabels?: boolean;
  lazy?: boolean;
}

/**
 * Renders an image with bounding-box overlays. The overlay is sized from the
 * image element's actual rendered dimensions, so boxes stay aligned across
 * thumbnails, responsive sizes, DPR and the full-size view.
 */
export function ImageWithBoxes({
  src,
  srcSet,
  sizes,
  alt,
  predictions,
  className,
  imgClassName,
  showLabels = true,
  lazy = true,
}: Props) {
  const imgRef = useRef<HTMLImageElement>(null);
  const size = useRenderedSize(imgRef);
  const [status, setStatus] = useState<'loading' | 'loaded' | 'error'>('loading');

  return (
    <div className={cx(styles.wrap, className)}>
      <img
        ref={imgRef}
        src={src}
        srcSet={srcSet}
        sizes={sizes}
        alt={alt}
        loading={lazy ? 'lazy' : 'eager'}
        className={cx(styles.img, imgClassName)}
        onLoad={() => setStatus('loaded')}
        onError={() => setStatus('error')}
        data-testid="bbox-image"
      />
      {status === 'loading' && <div className={styles.skeleton} aria-hidden />}
      {status === 'error' && (
        <div className={styles.error} data-testid="image-error">
          Image unavailable
        </div>
      )}
      {status === 'loaded' && (
        <BBoxOverlay
          predictions={predictions}
          renderedWidth={size.width}
          renderedHeight={size.height}
          showLabels={showLabels}
        />
      )}
    </div>
  );
}
