// Measures elements in CSS pixels. Called immediately before the screenshot,
// with nothing in between, so boxes and pixels describe the same DOM state.
// The Playwright script of the muninn-gen skill carries the same function.
async (spec) => {
  // A blinking caret would make screenshots differ between runs.
  if (!document.getElementById('__muninn_capture')) {
    const style = document.createElement('style');
    style.id = '__muninn_capture';
    style.textContent = '* { caret-color: transparent !important; }';
    document.head.appendChild(style);
  }
  await document.fonts.ready;
  const rect = (el) => {
    const r = el.getBoundingClientRect();
    return { x: r.x, y: r.y, w: r.width, h: r.height };
  };
  // Form fields are named by their label, never by their value: values can
  // be passwords or personal data and capture.json may be committed.
  const textOf = (el) => {
    let t;
    if (el.tagName === 'SELECT') t = (el.selectedOptions[0] || {}).text || '';
    else if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')
      t = (el.labels && el.labels[0] && el.labels[0].innerText) || el.getAttribute('aria-label') || el.placeholder || '';
    else t = el.innerText || '';
    return t.trim().replace(/\s+/g, ' ').slice(0, 200);
  };
  const elements = spec.elements.map(([key, selector]) => {
    const el = document.querySelector(selector);
    return el ? { key, selector, text: textOf(el), ...rect(el) } : { key, selector, missing: true };
  });
  let crop = null;
  if (spec.cropSelector) {
    const el = document.querySelector(spec.cropSelector);
    crop = el ? rect(el) : { missing: true };
  }
  return { dpr: window.devicePixelRatio || 1, url: location.href, elements, crop };
}
