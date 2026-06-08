import { describe, it, expect } from 'vitest';
import reducer, {
  initialFiltersState,
  toggleClass,
  setMinConfidence,
  setNearby,
  clearNearby,
  resetFilters,
} from './filtersSlice';

describe('filtersSlice (shared gallery/map state)', () => {
  it('selects a class and toggles it off when selected again', () => {
    const selected = reducer(initialFiltersState, toggleClass('thrips'));
    expect(selected.classId).toBe('thrips');
    const cleared = reducer(selected, toggleClass('thrips'));
    expect(cleared.classId).toBeNull();
  });

  it('switches to a different class', () => {
    const a = reducer(initialFiltersState, toggleClass('thrips'));
    const b = reducer(a, toggleClass('mirid'));
    expect(b.classId).toBe('mirid');
  });

  it('updates min confidence', () => {
    const next = reducer(initialFiltersState, setMinConfidence(0.7));
    expect(next.minConfidence).toBe(0.7);
  });

  it('sets and clears the nearby point', () => {
    const withNearby = reducer(initialFiltersState, setNearby({ x: 12.5, y: 20 }));
    expect(withNearby.nearby).toEqual({ x: 12.5, y: 20 });
    const cleared = reducer(withNearby, clearNearby());
    expect(cleared.nearby).toBeNull();
  });

  it('resets all filters', () => {
    let state = reducer(initialFiltersState, toggleClass('mirid'));
    state = reducer(state, setMinConfidence(0.5));
    state = reducer(state, setNearby({ x: 1, y: 2 }));
    expect(reducer(state, resetFilters())).toEqual(initialFiltersState);
  });
});
