import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BBoxOverlay } from './BBoxOverlay';
import type { Prediction } from '../types';

const predictions: Prediction[] = [
  { classId: 'thrips', confidence: 0.82, bbox: { xMin: 0, yMin: 0, xMax: 0.5, yMax: 0.5 } },
  { classId: 'mirid', confidence: 0.6, bbox: { xMin: 0.5, yMin: 0.5, xMax: 1, yMax: 1 } },
];

describe('BBoxOverlay', () => {
  it('positions boxes in pixels relative to the rendered image size', () => {
    render(<BBoxOverlay predictions={predictions} renderedWidth={200} renderedHeight={100} />);
    const box = screen.getByTestId('bbox-thrips');
    expect(box.style.left).toBe('0px');
    expect(box.style.top).toBe('0px');
    expect(box.style.width).toBe('100px');
    expect(box.style.height).toBe('50px');
  });

  it('renders a label with the class name and confidence', () => {
    render(<BBoxOverlay predictions={predictions} renderedWidth={200} renderedHeight={100} />);
    expect(screen.getByText(/Thrips · 82%/)).toBeInTheDocument();
  });

  it('renders nothing until the rendered size is known', () => {
    const { container } = render(
      <BBoxOverlay predictions={predictions} renderedWidth={0} renderedHeight={0} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});
