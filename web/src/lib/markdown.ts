import { marked } from 'marked';
import DOMPurify from 'dompurify';

marked.setOptions({
  gfm: true,
  breaks: true,
});

/** Render GFM markdown to sanitized HTML for Plank chat bubbles. */
export function renderMarkdown(src: string): string {
  if (!src) return '';
  const raw = marked.parse(src, { async: false }) as string;
  return DOMPurify.sanitize(raw, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target', 'rel'],
  });
}
