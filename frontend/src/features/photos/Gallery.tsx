import { useState } from 'react';
import type { Photo } from '../../shared/types';
import { PhotoCard } from './PhotoCard';
import { PhotoModal } from './PhotoModal';
import { Loader2, AlertTriangle, SearchX } from 'lucide-react';
import styles from './Gallery.module.css';

interface Props {
  photos: Photo[];
  token: string;
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}

export function Gallery({ photos, token, isLoading, isError, onRetry }: Props) {
  const [selected, setSelected] = useState<Photo | null>(null);

  if (isLoading) {
    return (
      <div className={styles.state} data-testid="gallery-loading">
        <Loader2 className={styles.spin} size={26} />
        <p>Loading captures…</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className={styles.state} data-testid="gallery-error">
        <AlertTriangle size={26} color="var(--danger)" />
        <p>We couldn’t load the photos.</p>
        <button type="button" className={styles.retry} onClick={onRetry} data-testid="gallery-retry">
          Try again
        </button>
      </div>
    );
  }

  if (photos.length === 0) {
    return (
      <div className={styles.state} data-testid="gallery-empty">
        <SearchX size={26} />
        <p>No captures match the current filters.</p>
      </div>
    );
  }

  return (
    <>
      <div className={styles.grid} data-testid="gallery-grid">
        {photos.map((p) => (
          <PhotoCard key={p.id} photo={p} token={token} onOpen={setSelected} />
        ))}
      </div>
      {selected && <PhotoModal photo={selected} token={token} onClose={() => setSelected(null)} />}
    </>
  );
}
