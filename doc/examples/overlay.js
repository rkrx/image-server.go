// Omitting canvas keeps the request image as the background.
function compose(url) {
  return {
    width: 640,
    height: 480,
    layout: [{
      type: 'image',
      url: '/markers/corner.png',
      pos: [98, 98],
      dim: [12, null],
      piv: 'right bottom'
    }]
  };
}
