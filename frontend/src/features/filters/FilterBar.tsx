import { useAppDispatch, useAppSelector } from '../../app/hooks';
import { DETECTION_CLASSES, CONFIDENCE_MAX, CONFIDENCE_MIN, CONFIDENCE_STEP } from '../../shared/constants';
import { clearNearby, resetFilters, setMinConfidence, toggleClass } from './filtersSlice';
import { SlidersHorizontal, X, MapPin } from 'lucide-react';
import styles from './FilterBar.module.css';

interface Props {
  resultCount: number;
}

export function FilterBar({ resultCount }: Props) {
  const dispatch = useAppDispatch();
  const { classId, minConfidence, nearby } = useAppSelector((s) => s.filters);
  const hasFilters = classId !== null || minConfidence > 0 || nearby !== null;

  return (
    <div className={styles.bar} data-testid="filter-bar">
      <div className={styles.chips} role="group" aria-label="Filter by detection class">
        {DETECTION_CLASSES.map((c) => {
          const active = classId === c.id;
          return (
            <button
              key={c.id}
              type="button"
              data-testid={`filter-chip-${c.id}`}
              aria-pressed={active}
              className={styles.chip}
              style={
                active
                  ? { background: c.color, borderColor: c.color, color: c.id === 'whitefly_aphid' ? '#111' : '#fff' }
                  : undefined
              }
              onClick={() => dispatch(toggleClass(c.id))}
            >
              <span className={styles.dot} style={{ background: c.color }} />
              {c.name}
            </button>
          );
        })}
      </div>

      <div className={styles.controls}>
        <label className={styles.slider}>
          <SlidersHorizontal size={15} />
          <span className={styles.sliderLabel}>Min confidence</span>
          <input
            type="range"
            min={CONFIDENCE_MIN}
            max={CONFIDENCE_MAX}
            step={CONFIDENCE_STEP}
            value={minConfidence}
            data-testid="filter-confidence"
            onChange={(e) => dispatch(setMinConfidence(Number(e.target.value)))}
          />
          <span className={`mono ${styles.sliderValue}`} data-testid="filter-confidence-value">
            {(minConfidence * 100).toFixed(0)}%
          </span>
        </label>

        {nearby && (
          <button
            type="button"
            className={styles.nearby}
            data-testid="filter-nearby"
            onClick={() => dispatch(clearNearby())}
          >
            <MapPin size={13} />
            <span className="mono">
              near {nearby.x.toFixed(1)}, {nearby.y.toFixed(1)} m
            </span>
            <X size={13} />
          </button>
        )}

        <span className={styles.count} data-testid="filter-result-count">
          {resultCount} {resultCount === 1 ? 'photo' : 'photos'}
        </span>

        {hasFilters && (
          <button
            type="button"
            className={styles.reset}
            data-testid="filter-reset"
            onClick={() => dispatch(resetFilters())}
          >
            Reset
          </button>
        )}
      </div>
    </div>
  );
}
