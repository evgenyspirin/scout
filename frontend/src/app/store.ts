import { configureStore } from '@reduxjs/toolkit';
import authReducer from '../features/auth/authSlice';
import filtersReducer from '../features/filters/filtersSlice';
import { scoutApi } from '../features/photos/photosApi';

export const store = configureStore({
  reducer: {
    auth: authReducer,
    filters: filtersReducer,
    [scoutApi.reducerPath]: scoutApi.reducer,
  },
  middleware: (getDefault) => getDefault().concat(scoutApi.middleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
