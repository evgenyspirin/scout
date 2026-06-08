import { useMemo } from 'react';
import { useAppDispatch, useAppSelector } from '../../app/hooks';
import { useListPhotosQuery } from '../photos/photosApi';
import { Gallery } from '../photos/Gallery';
import { GreenhouseMap } from '../map/GreenhouseMap';
import { FilterBar } from '../filters/FilterBar';
import { logout } from '../auth/authSlice';
import { distanceMeters } from '../../shared/bbox';
import { NEARBY_RADIUS_M } from '../../shared/constants';
import { Sprout, LogOut } from 'lucide-react';
import styles from './DashboardPage.module.css';

export function DashboardPage() {
  const dispatch = useAppDispatch();
  const token = useAppSelector((s) => s.auth.token) ?? '';
  const userLogin = useAppSelector((s) => s.auth.login);
  const role = useAppSelector((s) => s.auth.role);
  const { classId, minConfidence, nearby } = useAppSelector((s) => s.filters);

  const { data, isLoading, isFetching, isError, refetch } = useListPhotosQuery({ classId, minConfidence });
  const photos = useMemo(() => data?.items ?? [], [data]);

  // The map shows all filter-matched photos; the gallery additionally narrows
  // to a clicked map point ("nearby" within NEARBY_RADIUS_M meters).
  const galleryPhotos = useMemo(() => {
    if (!nearby) return photos;
    return photos.filter((p) => distanceMeters(p.x, p.y, nearby.x, nearby.y) <= NEARBY_RADIUS_M);
  }, [photos, nearby]);

  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <div className={styles.brand}>
          <span className={styles.logo}>
            <Sprout size={22} /> Scout
          </span>
          <span className={styles.sub}>AlfaGreen greenhouse monitoring</span>
        </div>
        <div className={styles.user}>
          <span className={styles.userName} data-testid="current-user">
            {userLogin}
            {role && <span className={styles.role}>{role}</span>}
          </span>
          <button type="button" className={styles.logout} onClick={() => dispatch(logout())} data-testid="logout-button">
            <LogOut size={15} /> Sign out
          </button>
        </div>
      </header>

      <FilterBar resultCount={galleryPhotos.length} />

      <div className={styles.main}>
        <section className={styles.galleryCol} data-testid="gallery-section">
          <Gallery
            photos={galleryPhotos}
            token={token}
            isLoading={isLoading || (isFetching && photos.length === 0)}
            isError={isError}
            onRetry={refetch}
          />
        </section>
        <aside className={styles.mapCol} data-testid="map-section">
          <GreenhouseMap photos={photos} />
        </aside>
      </div>
    </div>
  );
}
