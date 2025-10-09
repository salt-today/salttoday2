/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["internal/server/ui/**/*.templ"],
  // Can't have dynamic values via string interpolation
  // Need this safelist for gradients on comments to work, which is awful.
  safelist: Array.from(new Array(20), (_, index) => 'to-' + index * 5 + '%').concat(
    Array.from(new Array(20), (_, index) => 'from-' + index * 5 + '%')).concat(
      Array.from(new Array(101), (_, index) => 'w-[' + index + '%]')
    ),
  theme: {
    fontFamily: {
      bebas: ['Bebas Neue', 'sans-serif'],
    },
    extend: {},
  },
  plugins: [],
}

