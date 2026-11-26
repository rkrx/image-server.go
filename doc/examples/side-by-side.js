// Place two images side by side on an 800 × 400 white canvas.
function compose(url) {
  function panel(path, x, profile) {
    if (!path || !path.startsWith('/')) {
      throw new Error('Each image needs an origin-relative path starting with /');
    }
    const imagePath = path.split('/').map(encodeURIComponent).join('/');
    return {
      type: 'image',
      url: profile ? imagePath + '?profile=' + encodeURIComponent(profile) : imagePath,
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
