// Place two images side by side on an 800 × 400 white canvas.
function compose(url) {
  function panel(path, x, profile) {
    if (!path || !path.startsWith('/')) {
      throw new Error('Each image needs an origin-relative path starting with /');
    }
    // These paths are already decoded request arguments, including literal percent signs.
    return {
      type: 'image',
      url: imageUrl(path.replace(/%/g, '%25'), {profile: profile || undefined}),
      pos: [x, 0],
      dim: [50, 100],
      fit: 'contain',
      alg: 'center center'
    };
  }

  return {
    width: 800,
    height: 400,
    canvas: 'white',
    layout: [
      panel(url.pathname, 0),
      panel(url.params.right, 50, url.params.rightProfile)
    ]
  };
}
