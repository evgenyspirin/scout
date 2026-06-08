import { useEffect, useState, type RefObject } from 'react';

export interface Size {
  width: number;
  height: number;
}

/**
 * Tracks the live rendered (CSS pixel) size of an element via ResizeObserver.
 * Used to map normalized bounding boxes onto the actual rendered image size.
 */
export function useRenderedSize(ref: RefObject<HTMLElement | null>): Size {
  const [size, setSize] = useState<Size>({ width: 0, height: 0 });

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const update = () => setSize({ width: el.clientWidth, height: el.clientHeight });
    update();
    const observer = new ResizeObserver(update);
    observer.observe(el);
    return () => observer.disconnect();
  }, [ref]);

  return size;
}
