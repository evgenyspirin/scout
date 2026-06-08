import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { RootState } from '../../app/store';
import type { LoginResponse, PhotoPage } from '../../shared/types';
import { PAGE_LIMIT } from '../../shared/constants';

const API_BASE = `${import.meta.env.VITE_BACKEND_URL}/api/v1`;

export interface ListPhotosArgs {
  classId: string | null;
  minConfidence: number;
}

export interface LoginArgs {
  login: string;
  password: string;
}

export const scoutApi = createApi({
  reducerPath: 'scoutApi',
  baseQuery: fetchBaseQuery({
    baseUrl: API_BASE,
    prepareHeaders: (headers, { getState }) => {
      const token = (getState() as RootState).auth.token;
      if (token) headers.set('Authorization', `Bearer ${token}`);
      return headers;
    },
  }),
  endpoints: (builder) => ({
    login: builder.mutation<LoginResponse, LoginArgs>({
      query: (body) => ({ url: '/auth/login', method: 'POST', body }),
    }),
    listPhotos: builder.query<PhotoPage, ListPhotosArgs>({
      query: ({ classId, minConfidence }) => {
        const params = new URLSearchParams();
        params.set('limit', String(PAGE_LIMIT));
        if (classId) params.set('classId', classId);
        if (minConfidence > 0) params.set('minConfidence', String(minConfidence));
        return `/photos?${params.toString()}`;
      },
    }),
  }),
});

export const { useLoginMutation, useListPhotosQuery } = scoutApi;
