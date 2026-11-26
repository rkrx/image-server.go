// Bundle layout used by the image-server profile. The request path is the main image.
const bundleRatio = 100 / 58;

function image(source, x, boxWidth, parentRatio) {
  if (source.startsWith('bundle-composition:')) {
    throw new Error('Expand bundle components into individual image paths before signing');
  }
  const sourceRatio = 605 / 500;
  const width = Math.min(boxWidth, 100 * sourceRatio / parentRatio);
  const height = Math.min(100, boxWidth * parentRatio / sourceRatio);
  const pos = [x + (boxWidth - width) / 2, (100 - height) / 2];
  const dim = [width, height];

  if (/^[ \t\n\r\0\v]*$/.test(source) || /^[a-z][a-z0-9+.-]*:|^\/\/|[?#]/i.test(source)) {
    source = 'noimg.jpg';
  }
  return {type: 'image', url: imageUrl(source, {autocrop: '12,3', mode: 'proportional'}), pos, dim, fit: 'contain'};
}

function compose(url) {
  const params = url.params;
  const secondary = (params.images || '').trim().split(/\s+/).filter(Boolean).slice(0, 4);
  const layout = [image(url.pathname.slice(1), 0, 70, bundleRatio)];
  const gap = 4;
  const rows = Math.max(3, secondary.length);
  const rowHeight = (100 - (rows - 1) * gap) / rows;
  const groupHeight = secondary.length * rowHeight + Math.max(0, secondary.length - 1) * gap;
  const startY = (100 - groupHeight) / 2;
  secondary.forEach((source, index) => {
    layout.push({
      type: 'rect',
      pos: [70, startY + index * (rowHeight + gap)],
      dim: [30, rowHeight],
      cld: [
        {
          type: 'image', url: imageUrl('/misc/product-gallery/bundle-plus.png', {profile: 'public-img'}),
          pos: [16, 50], dim: [18, null], piv: 'center center', mod: 'pixelated'
        },
        image(source, 32, 68, 30 / (rowHeight * 0.58))
      ]
    });
  });

  const width = Number(params.width ?? (params.height == null ? 1500 : Math.round(params.height * bundleRatio)));
  const height = Number(params.height ?? Math.round(width / bundleRatio));
  if (![width, height].every(n => Number.isSafeInteger(n) && n > 0)) {
    throw new Error('Invalid canvas size');
  }
  const ratio = width / height;
  const frameWidth = Math.round(Math.min(100, 100 * bundleRatio / ratio) * 1e6) / 1e6;
  const frameHeight = Math.round(Math.min(100, 100 * ratio / bundleRatio) * 1e6) / 1e6;
  return {
    width, height, canvas: 'white',
    layout: [{
      type: 'rect', pos: [(100 - frameWidth) / 2, (100 - frameHeight) / 2],
      dim: [frameWidth, frameHeight], cld: layout
    }]
  };
}
