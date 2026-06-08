// Domain types mirroring the backend API responses.

export interface BBox {
  xMin: number;
  yMin: number;
  xMax: number;
  yMax: number;
}

export interface Prediction {
  classId: string;
  confidence: number;
  bbox: BBox;
}

export interface Photo {
  id: string;
  x: number;
  y: number;
  h: number;
  width: number;
  height: number;
  capturedAt: string;
  originalUrl: string;
  predictions: Prediction[];
}

export interface PhotoPage {
  items: Photo[];
  next_token?: string;
}

export interface LoginResponse {
  access_token: string;
  token_type: string;
}

export interface DetectionClass {
  id: string;
  name: string;
  color: string;
}
