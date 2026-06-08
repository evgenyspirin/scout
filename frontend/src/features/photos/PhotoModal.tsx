import { useEffect } from 'react';
import type { Photo } from '../../shared/types';
import { ImageWithBoxes } from '../../shared/components/ImageWithBoxes';
import { withToken } from '../../shared/api/urls';
import { classColor, className } from '../../shared/constants';
import { X, MapPin, Ruler, Clock } from 'lucide-react';
import styles from './PhotoModal.module.css';

interface Props {
  photo: Photo;
  token: string;
  onClose: () => void;
}

export function PhotoModal({ photo, token, onClose }: Props) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', onKey);
      document.body.style.overflow = prev;
    };
  }, [onClose]);

  const sorted = [...photo.predictions].sort((a, b) => b.confidence - a.confidence);

  return (
    <div className={styles.backdrop} onClick={onClose} data-testid="photo-modal">
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
        <header className={styles.header}>
          <span className={`mono ${styles.id}`}>{photo.id}</span>
          <button type="button" className={styles.close} onClick={onClose} data-testid="photo-modal-close" aria-label="Close">
            <X size={18} />
          </button>
        </header>

        <div className={styles.body}>
          <div className={styles.imageArea}>
            <ImageWithBoxes
              className={styles.image}
              src={withToken(photo.originalUrl, token)}
              alt={`Full-size capture ${photo.id}`}
              predictions={photo.predictions}
              lazy={false}
            />
          </div>

          <aside className={styles.side}>
            <div className={styles.facts}>
              <span className={styles.fact}>
                <MapPin size={14} /> <span className="mono">{photo.x.toFixed(1)}, {photo.y.toFixed(1)} m</span>
              </span>
              <span className={styles.fact}>
                <Ruler size={14} /> cam height <span className="mono">{photo.h.toFixed(1)} m</span>
              </span>
              <span className={styles.fact}>
                <Clock size={14} /> {new Date(photo.capturedAt).toLocaleString()}
              </span>
            </div>

            <h3 className={styles.predTitle}>
              Predictions <span className={styles.predCount}>{photo.predictions.length}</span>
            </h3>
            {sorted.length === 0 ? (
              <p className={styles.noPred}>No detections in this photo.</p>
            ) : (
              <ul className={styles.predList} data-testid="modal-predictions">
                {sorted.map((p, i) => (
                  <li key={`${p.classId}-${i}`} className={styles.predItem}>
                    <span className={styles.predDot} style={{ background: classColor(p.classId) }} />
                    <span className={styles.predName}>{className(p.classId)}</span>
                    <span className={`mono ${styles.predConf}`}>{(p.confidence * 100).toFixed(1)}%</span>
                  </li>
                ))}
              </ul>
            )}
          </aside>
        </div>
      </div>
    </div>
  );
}
