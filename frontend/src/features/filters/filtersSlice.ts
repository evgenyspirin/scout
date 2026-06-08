import { createSlice, type PayloadAction } from '@reduxjs/toolkit';
import { DEFAULT_MIN_CONFIDENCE } from '../../shared/constants';

export interface NearbyPoint {
  x: number;
  y: number;
}

export interface FiltersState {
  classId: string | null;
  minConfidence: number;
  nearby: NearbyPoint | null;
}

export const initialFiltersState: FiltersState = {
  classId: null,
  minConfidence: DEFAULT_MIN_CONFIDENCE,
  nearby: null,
};

const filtersSlice = createSlice({
  name: 'filters',
  initialState: initialFiltersState,
  reducers: {
    // Toggling the active class (selecting the same class clears it).
    toggleClass(state, action: PayloadAction<string>) {
      state.classId = state.classId === action.payload ? null : action.payload;
    },
    setClass(state, action: PayloadAction<string | null>) {
      state.classId = action.payload;
    },
    setMinConfidence(state, action: PayloadAction<number>) {
      state.minConfidence = action.payload;
    },
    setNearby(state, action: PayloadAction<NearbyPoint | null>) {
      state.nearby = action.payload;
    },
    clearNearby(state) {
      state.nearby = null;
    },
    resetFilters() {
      return initialFiltersState;
    },
  },
});

export const {
  toggleClass,
  setClass,
  setMinConfidence,
  setNearby,
  clearNearby,
  resetFilters,
} = filtersSlice.actions;

export default filtersSlice.reducer;
