// The composition transport uses origin-relative paths. Preserve valid %HH path escapes.
globalThis.imageUrl = function imageUrl(path, params = {}) {
  if (typeof path !== 'string' || !path || path.startsWith('//') || /^[a-z][a-z\d+.-]*:/i.test(path)) {
    throw new TypeError('imageUrl requires an image path');
  }
  function encode(value) {
    return encodeURIComponent(value).replace(/[!'()*]/g, c => '%' + c.charCodeAt(0).toString(16).toUpperCase());
  }
  function segment(value) {
    return encode(value).replace(/%25([0-9a-f]{2})/gi, (_, hex) => {
      const byte = String.fromCharCode(parseInt(hex, 16));
      return /[A-Za-z0-9._~-]/.test(byte) ? byte : '%' + hex.toUpperCase();
    });
  }

  const pathname = '/' + path.replace(/^\//, '').split('/').map(segment).join('/');
  const separators = pathname.replace(/%2F/gi, '/');
  if (separators.startsWith('//') || /%5C/i.test(pathname) || separators.split('/').some(p => p === '.' || p === '..')) {
    throw new TypeError('imageUrl requires a path within the origin');
  }
  if (params === null || typeof params !== 'object' || Array.isArray(params)) {
    throw new TypeError('imageUrl parameters must be an object');
  }
  const query = [];
  for (const [name, value] of Object.entries(params)) {
    if (value == null) continue;
    if (typeof value !== 'string' && !Number.isFinite(value)) {
      throw new TypeError('imageUrl parameter values must be strings or finite numbers');
    }
    query.push(encode(name) + '=' + encode(value));
  }
  return query.length ? pathname + '?' + query.join('&') : pathname;
};
