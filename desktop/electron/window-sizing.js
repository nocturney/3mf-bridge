'use strict';

// Geometry-only helper. All units are physical window pixels except desired
// content dimensions, which are CSS pixels before the Electron zoom factor.
const MIN_ZOOM = 0.72;

function fitContentToDisplay({
  contentWidth,
  contentHeight,
  maxClientWidth,
  maxClientHeight,
  minClientWidth = 720,
  minClientHeight = 520
}) {
  const dimensions = [contentWidth, contentHeight, maxClientWidth, maxClientHeight];
  if (dimensions.some((value) => !Number.isFinite(value) || value <= 0)) {
    throw new TypeError('window dimensions must be positive finite numbers');
  }

  const limitWidth = Math.max(1, Math.floor(maxClientWidth));
  const limitHeight = Math.max(1, Math.floor(maxClientHeight));
  const idealZoom = Math.min(
    1,
    limitWidth / Math.ceil(contentWidth),
    limitHeight / Math.ceil(contentHeight)
  );
  const zoomFactor = Math.max(MIN_ZOOM, Math.floor(idealZoom * 100) / 100);

  const width = Math.min(
    limitWidth,
    Math.max(Math.min(limitWidth, minClientWidth), Math.ceil(contentWidth * zoomFactor))
  );
  const height = Math.min(
    limitHeight,
    Math.max(Math.min(limitHeight, minClientHeight), Math.ceil(contentHeight * zoomFactor))
  );

  return {
    width,
    height,
    zoomFactor,
    fitsWithoutScrolling:
      Math.ceil(contentWidth * zoomFactor) <= width + 1 &&
      Math.ceil(contentHeight * zoomFactor) <= height + 1
  };
}

module.exports = { fitContentToDisplay, MIN_ZOOM };
