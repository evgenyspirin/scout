import { useRef, useState } from 'react';
import { Stage, Layer, Rect, Line, Circle, Text, Group } from 'react-konva';
import type Konva from 'konva';
import type { Photo } from '../../shared/types';
import { useAppDispatch, useAppSelector } from '../../app/hooks';
import { useRenderedSize } from '../../shared/hooks/useRenderedSize';
import { dominantClass } from '../../shared/bbox';
import { classColor, GREENHOUSE_SIZE_M, NEARBY_RADIUS_M } from '../../shared/constants';
import { setNearby } from '../filters/filtersSlice';
import { Crosshair } from 'lucide-react';
import styles from './GreenhouseMap.module.css';

interface Props {
  photos: Photo[];
}

const GRID_STEP_M = 5;
const MIN_ZOOM = 0.6;
const MAX_ZOOM = 6;
const ZOOM_FACTOR = 1.08;
const MARKER_BASE_R = 5;
const MARKER_CONF_R = 7;

function clamp(v: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, v));
}

interface View {
  scale: number;
  x: number;
  y: number;
}

const INITIAL_VIEW: View = { scale: 1, x: 0, y: 0 };

export function GreenhouseMap({ photos }: Props) {
  const dispatch = useAppDispatch();
  const nearby = useAppSelector((s) => s.filters.nearby);
  const containerRef = useRef<HTMLDivElement>(null);
  const { width } = useRenderedSize(containerRef);
  const [view, setView] = useState<View>(INITIAL_VIEW);

  const size = width > 0 ? width : 480;
  const mToPx = size / GREENHOUSE_SIZE_M;

  const handleWheel = (e: Konva.KonvaEventObject<WheelEvent>) => {
    e.evt.preventDefault();
    const stage = e.target.getStage();
    const pointer = stage?.getPointerPosition();
    if (!pointer) return;
    const oldScale = view.scale;
    const worldX = (pointer.x - view.x) / oldScale;
    const worldY = (pointer.y - view.y) / oldScale;
    const newScale = clamp(
      e.evt.deltaY > 0 ? oldScale / ZOOM_FACTOR : oldScale * ZOOM_FACTOR,
      MIN_ZOOM,
      MAX_ZOOM,
    );
    setView({
      scale: newScale,
      x: pointer.x - worldX * newScale,
      y: pointer.y - worldY * newScale,
    });
  };

  const gridLines = [];
  for (let m = 0; m <= GREENHOUSE_SIZE_M; m += GRID_STEP_M) {
    const p = m * mToPx;
    gridLines.push(
      <Line key={`v${m}`} points={[p, 0, p, size]} stroke="#e3ded7" strokeWidth={1} />,
      <Line key={`h${m}`} points={[0, p, size, p]} stroke="#e3ded7" strokeWidth={1} />,
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h3 className={styles.title}>AlfaGreen · {GREENHOUSE_SIZE_M}×{GREENHOUSE_SIZE_M} m</h3>
        <button
          type="button"
          className={styles.reset}
          data-testid="map-reset-view"
          onClick={() => setView(INITIAL_VIEW)}
        >
          <Crosshair size={14} /> Reset view
        </button>
      </div>

      <div className={styles.canvasWrap} ref={containerRef} data-testid="greenhouse-map">
        <Stage
          width={size}
          height={size}
          draggable
          scaleX={view.scale}
          scaleY={view.scale}
          x={view.x}
          y={view.y}
          onWheel={handleWheel}
          onDragEnd={(e) => setView((v) => ({ ...v, x: e.target.x(), y: e.target.y() }))}
        >
          <Layer>
            <Rect x={0} y={0} width={size} height={size} fill="#f7f4ef" stroke="#cdc6bb" strokeWidth={2} />
            {gridLines}

            {nearby && (
              <Circle
                x={nearby.x * mToPx}
                y={nearby.y * mToPx}
                radius={NEARBY_RADIUS_M * mToPx}
                fill="rgba(22,101,52,0.12)"
                stroke="#166534"
                strokeWidth={1.5}
                dash={[4, 4]}
                listening={false}
              />
            )}

            {photos.map((photo) => {
              const cls = dominantClass(photo.predictions);
              const color = cls ? classColor(cls) : '#9ca3af';
              const maxConf = photo.predictions.reduce((m, p) => Math.max(m, p.confidence), 0);
              const r = MARKER_BASE_R + maxConf * MARKER_CONF_R;
              const isSelected =
                nearby !== null &&
                Math.hypot(photo.x - nearby.x, photo.y - nearby.y) <= NEARBY_RADIUS_M;
              return (
                <Circle
                  key={photo.id}
                  data-testid={`map-marker-${photo.id}`}
                  x={photo.x * mToPx}
                  y={photo.y * mToPx}
                  radius={r / view.scale}
                  fill={color}
                  opacity={photo.predictions.length === 0 ? 0.45 : 0.92}
                  stroke={isSelected ? '#166534' : '#ffffff'}
                  strokeWidth={(isSelected ? 3 : 1.5) / view.scale}
                  onMouseEnter={(e) => {
                    const stage = e.target.getStage();
                    if (stage) stage.container().style.cursor = 'pointer';
                  }}
                  onMouseLeave={(e) => {
                    const stage = e.target.getStage();
                    if (stage) stage.container().style.cursor = 'default';
                  }}
                  onClick={() => dispatch(setNearby({ x: photo.x, y: photo.y }))}
                  onTap={() => dispatch(setNearby({ x: photo.x, y: photo.y }))}
                />
              );
            })}

            <Group listening={false}>
              <Text x={6} y={size - 18} text="0 m" fontSize={11} fill="#a8a29e" fontFamily="IBM Plex Mono" />
              <Text x={size - 34} y={size - 18} text={`${GREENHOUSE_SIZE_M} m`} fontSize={11} fill="#a8a29e" fontFamily="IBM Plex Mono" />
            </Group>
          </Layer>
        </Stage>
      </div>

      <p className={styles.hint}>
        Scroll to zoom · drag to pan · click a point to filter the gallery to photos within{' '}
        {NEARBY_RADIUS_M} m.
      </p>
    </div>
  );
}
