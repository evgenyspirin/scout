import type { Photo } from '../../shared/types';
import { ImageWithBoxes } from '../../shared/components/ImageWithBoxes';
import { thumbnailSrcSet, thumbnailUrl } from '../../shared/api/urls';
import { DEFAULT_THUMBNAIL_WIDTH } from '../../shared/constants';
import { MapPin } from 'lucide-react';
import styles from './PhotoCard.module.css';

interface Props {
  photo: Photo;
  token: string;
  onOpen: (photo: Photo) => void;
}

const SIZES = '(max-width: 640px) 100vw, (max-width: 1024px) 50vw, (max-width: 1440px) 33vw, 25vw';

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function PhotoCard({ photo, token, onOpen }: Props) {
  const count = photo.predictions.length;
  return (
    <button
      type="button"
      className={styles.card}
      onClick={() => onOpen(photo)}
      data-testid={`photo-card-${photo.id}`}
    >
      <ImageWithBoxes
        className={styles.image}
        src={thumbnailUrl(photo.id, DEFAULT_THUMBNAIL_WIDTH, token)}
        srcSet={thumbnailSrcSet(photo.id, token)}
        sizes={SIZES}
        alt={`Greenhouse capture ${photo.id}`}
        predictions={photo.predictions}
      />
      <div className={styles.footer}>
        <span className={styles.pos}>
          <MapPin size={13} strokeWidth={2.4} />
          <span className="mono">
            {photo.x.toFixed(1)}, {photo.y.toFixed(1)} m
          </span>
        </span>
        <span className={styles.meta}>
          <span className={styles.count} data-testid={`photo-card-count-${photo.id}`}>
            {count} {count === 1 ? 'detection' : 'detections'}
          </span>
          <span className={styles.time}>{formatTime(photo.capturedAt)}</span>
        </span>
      </div>
    </button>
  );
}
