// The site's social preview image (public/og.png, 1200x630). Starlight sets
// og:title/og:description and twitter:card itself but no image, so its pages get
// these from `head` in astro.config.mjs; the standalone pages use SocialCard.astro.
// The landing page and the field notes (BlogLayout.astro) set their own.
export const socialCard = {
  url: 'https://sarathsp06.github.io/sparrow/og.png',
  width: '1200',
  height: '630',
  alt: 'Sparrow — self-hosted webhook delivery',
};

// Starlight `head` entries.
export const socialCardHead = [
  { tag: 'meta', attrs: { property: 'og:image', content: socialCard.url } },
  { tag: 'meta', attrs: { property: 'og:image:width', content: socialCard.width } },
  { tag: 'meta', attrs: { property: 'og:image:height', content: socialCard.height } },
  { tag: 'meta', attrs: { property: 'og:image:alt', content: socialCard.alt } },
  { tag: 'meta', attrs: { name: 'twitter:image', content: socialCard.url } },
];
